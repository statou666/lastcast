package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// gPodder API v2 + proxy feeds.
//
// A client (AntennaPod etc.) syncs its subscriptions and episode actions with us,
// but it fetches and plays RSS itself. So the subscription list we hand out points
// at OUR per-feed proxy XML: each item's enclosure is /stream/{id} (Range proxy,
// instant, no download) or /speak/{id} (TTS rendered while streaming). The feed
// URL carries a per-user token because podcast clients do not send cookies.

func apiToken(uid int64) string {
	var t string
	_ = store.db.QueryRow(`SELECT COALESCE(api_token,'') FROM users WHERE id=?`, uid).Scan(&t)
	if t == "" {
		t = randHex(24)
		_, _ = store.db.Exec(`UPDATE users SET api_token=? WHERE id=?`, t, uid)
	}
	return t
}

func randHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return hex.EncodeToString(b)
}

func ensureTokens() {
	rows, err := store.db.Query(`SELECT id FROM users WHERE COALESCE(api_token,'')=''`)
	if err != nil {
		return
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()
	for _, id := range ids {
		apiToken(id)
	}
}

func tokenUser(r *http.Request) *User {
	t := r.URL.Query().Get("t")
	if t == "" {
		t = r.URL.Query().Get("token")
	}
	if t == "" {
		t = r.Header.Get("X-Api-Token")
	}
	if t == "" {
		return nil
	}
	var u User
	if err := store.db.QueryRow(`SELECT id,username,role FROM users WHERE api_token=? AND api_token<>''`, t).
		Scan(&u.ID, &u.Name, &u.Role); err != nil {
		return nil
	}
	return &u
}

func baseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

func proxyFeedURL(r *http.Request, uid, feedID int64) string {
	return fmt.Sprintf("%s/gpodder/feed/%s/%d.xml", baseURL(r), apiToken(uid), feedID)
}

// feedIDFromProxyURL accepts our own proxy URL (…/gpodder/feed/<token>/<id>.xml).
func feedIDFromProxyURL(u string) int64 {
	i := strings.Index(u, "/gpodder/feed/")
	if i < 0 {
		return 0
	}
	rest := u[i+len("/gpodder/feed/"):]
	if j := strings.IndexByte(rest, '/'); j >= 0 {
		rest = rest[j+1:]
	}
	rest = strings.TrimSuffix(rest, ".xml")
	if k := strings.IndexAny(rest, "?#"); k >= 0 {
		rest = rest[:k]
	}
	n, _ := strconv.ParseInt(rest, 10, 64)
	return n
}

// gpodAuth validates HTTP Basic credentials against the user named in the path.
func gpodAuth(w http.ResponseWriter, r *http.Request, username string) *User {
	if name, pass, ok := r.BasicAuth(); ok {
		if u := verifyUser(name, pass); u != nil && strings.EqualFold(u.Name, username) {
			return u
		}
	}
	if u := currentUser(r); u != nil && strings.EqualFold(u.Name, username) {
		return u
	}
	w.Header().Set("WWW-Authenticate", `Basic realm="Lastcast"`)
	writeJSONStatus(w, http.StatusUnauthorized, map[string]any{"error": "unauthorized"})
	return nil
}

// gpodRouter dispatches /api/2/... (gPodder's URL shape uses .json suffixes).
func gpodRouter(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/2/"), "/")
	if len(parts) < 2 {
		writeJSONStatus(w, http.StatusNotFound, map[string]any{"error": "not found"})
		return
	}
	username := strings.TrimSuffix(parts[1], ".json")
	switch parts[0] {
	case "subscriptions":
		u := gpodAuth(w, r, username)
		if u == nil {
			return
		}
		gpodSubscriptions(w, r, u)
	case "episode-actions":
		u := gpodAuth(w, r, username)
		if u == nil {
			return
		}
		gpodEpisodeActions(w, r, u)
	case "devices":
		u := gpodAuth(w, r, username)
		if u == nil {
			return
		}
		if len(parts) >= 3 {
			gpodDeviceUpdate(w, r, u, strings.TrimSuffix(parts[2], ".json"))
		} else {
			gpodDevices(w, r, u)
		}
	case "auth":
		u := gpodAuth(w, r, username)
		if u == nil {
			return
		}
		writeJSON(w, map[string]any{"ok": true, "user": u.Name})
	default:
		writeJSONStatus(w, http.StatusNotFound, map[string]any{"error": "not found"})
	}
}

func gpodSubscriptions(w http.ResponseWriter, r *http.Request, u *User) {
	switch r.Method {
	case http.MethodGet:
		rows, err := store.db.Query(`SELECT id FROM feeds WHERE user_id=? ORDER BY id`, u.ID)
		if err != nil {
			writeJSONStatus(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		add := []string{}
		for rows.Next() {
			var id int64
			if rows.Scan(&id) == nil {
				add = append(add, proxyFeedURL(r, u.ID, id))
			}
		}
		rows.Close()
		writeJSON(w, map[string]any{"add": add, "remove": []string{}})
	case http.MethodPut, http.MethodPost:
		var body struct {
			Add    []string `json:"add"`
			Remove []string `json:"remove"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		added := 0
		for _, uu := range body.Add {
			if feedIDFromProxyURL(uu) > 0 {
				continue // already one of ours
			}
			if uu == "" {
				continue
			}
			var id int64
			if err := store.db.QueryRow(`SELECT id FROM feeds WHERE url=? AND user_id=?`, uu, u.ID).Scan(&id); err != nil {
				res, err := store.db.Exec(`INSERT INTO feeds(user_id,url,last_fetched) VALUES(?,?,?)`, u.ID, uu, nowUnix())
				if err != nil {
					continue
				}
				id, _ = res.LastInsertId()
				_, _ = fetchAndUpsertFeed(id, uu)
				added++
			}
		}
		for _, uu := range body.Remove {
			if id := feedIDFromProxyURL(uu); id > 0 {
				deleteFeedForUser(u.ID, id)
				continue
			}
			var id int64
			if store.db.QueryRow(`SELECT id FROM feeds WHERE url=? AND user_id=?`, uu, u.ID).Scan(&id) == nil {
				deleteFeedForUser(u.ID, id)
			}
		}
		writeJSON(w, map[string]any{"ok": true, "added": added})
	default:
		writeJSONStatus(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
	}
}

// gpodDevices lists this user's devices for the client's device picker.
func gpodDevices(w http.ResponseWriter, r *http.Request, u *User) {
	rows, err := store.db.Query(`SELECT id,COALESCE(name,''),last_seen FROM devices WHERE user_id=? ORDER BY last_seen DESC`, u.ID)
	if err != nil {
		writeJSONStatus(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, name string
		var seen int64
		if rows.Scan(&id, &name, &seen) == nil {
			out = append(out, map[string]any{"id": id, "caption": name, "type": "other", "subscriptions": 0})
		}
	}
	writeJSON(w, out)
}

func gpodDeviceUpdate(w http.ResponseWriter, r *http.Request, u *User, device string) {
	var body struct {
		Caption string `json:"caption"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if device != "" {
		_, _ = store.db.Exec(`INSERT INTO devices(user_id,id,name,client_type,last_seen) VALUES(?,?,?,?,?)
			ON CONFLICT(user_id,id) DO UPDATE SET name=COALESCE(NULLIF(excluded.name,''),name), last_seen=excluded.last_seen`,
			u.ID, device, body.Caption, "gpodder", nowUnix())
	}
	writeJSON(w, map[string]any{"ok": true})
}

// gpodEpisodeActions syncs playback progress both ways.
func gpodEpisodeActions(w http.ResponseWriter, r *http.Request, u *User) {
	switch r.Method {
	case http.MethodGet:
		since, _ := strconv.ParseInt(r.URL.Query().Get("since"), 10, 64)
		rows, err := store.db.Query(`SELECT e.guid, f.id, COALESCE(p.position_sec,0), COALESCE(p.completed,0),
			COALESCE(p.updated_at,0), COALESCE(p.device_id,''), COALESCE(e.duration_sec,0)
			FROM progress p JOIN episodes e ON e.id=p.episode_id JOIN feeds f ON f.id=e.feed_id
			WHERE f.user_id=? AND p.updated_at>? ORDER BY p.updated_at DESC LIMIT 500`, u.ID, since)
		if err != nil {
			writeJSONStatus(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		defer rows.Close()
		out := []map[string]any{}
		for rows.Next() {
			var guid, device string
			var feedID, pos, done, at, dur int64
			if rows.Scan(&guid, &feedID, &pos, &done, &at, &device, &dur) == nil {
				out = append(out, map[string]any{
					"podcast": proxyFeedURL(r, u.ID, feedID), "episode": guid, "action": "play",
					"timestamp": at, "started": 0, "position": pos, "total": dur, "device": device,
				})
			}
		}
		writeJSON(w, out)
	case http.MethodPost:
		var actions []struct {
			Podcast   string `json:"podcast"`
			Episode   string `json:"episode"`
			Action    string `json:"action"`
			Timestamp int64  `json:"timestamp"`
			Position  int64  `json:"position"`
			Total     int64  `json:"total"`
			Device    string `json:"device"`
		}
		if err := json.NewDecoder(r.Body).Decode(&actions); err != nil {
			writeJSONStatus(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
			return
		}
		for _, a := range actions {
			feedID := feedIDFromProxyURL(a.Podcast)
			if feedID == 0 || !feedOwned(u.ID, feedID) {
				continue
			}
			var epID int64
			if store.db.QueryRow(`SELECT id FROM episodes WHERE feed_id=? AND guid=?`, feedID, a.Episode).Scan(&epID) != nil {
				continue
			}
			device := a.Device
			if device == "" {
				device = "gpodder"
			}
			switch a.Action {
			case "play":
				pos := a.Position
				done := int64(0)
				if a.Total > 0 && pos >= a.Total-1 {
					done = 1
				}
				_, _ = store.db.Exec(`INSERT INTO progress(user_id,episode_id,device_id,position_sec,completed,segment_idx,updated_at)
					VALUES(?,?,?,?,?,0,?)
					ON CONFLICT(user_id,episode_id,device_id) DO UPDATE SET
						position_sec=MAX(position_sec, excluded.position_sec),
						completed=MAX(completed, excluded.completed),
						updated_at=excluded.updated_at`,
					u.ID, epID, device, pos, done, nowUnix())
			case "download":
				if needsDownload(epID) {
					dl.Enqueue(epID)
				}
			case "delete", "new", "reset":
				_, _ = store.db.Exec(`DELETE FROM progress WHERE user_id=? AND episode_id=?`, u.ID, epID)
			}
		}
		writeJSON(w, map[string]any{"ok": true, "updated": len(actions)})
	default:
		writeJSONStatus(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
	}
}

// gpodFeed serves the per-feed proxy RSS: enclosures point back at /stream or
// /speak, so the client can play video and text items immediately (no pre-download).
func gpodFeed(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/gpodder/feed/")
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}
	token := parts[0]
	feedID, _ := strconv.ParseInt(strings.TrimSuffix(parts[1], ".xml"), 10, 64)
	var u User
	if store.db.QueryRow(`SELECT id,username,role FROM users WHERE api_token=? AND api_token<>''`, token).
		Scan(&u.ID, &u.Name, &u.Role) != nil || feedID == 0 || !feedOwned(u.ID, feedID) {
		http.NotFound(w, r)
		return
	}
	var title, desc, cover, author string
	var url string
	if err := store.db.QueryRow(`SELECT COALESCE(title,''),COALESCE(description,''),COALESCE(cover_url,''),
		COALESCE(author,''),url FROM feeds WHERE id=?`, feedID).Scan(&title, &desc, &cover, &author, &url); err != nil {
		http.NotFound(w, r)
		return
	}
	type item struct {
		id       int64
		guid     string
		title    string
		desc     string
		kind     string
		cover    string
		pub      int64
		dur      int64
		file     string
		enclType string
		encURL   string
	}
	rows, err := store.db.Query(`SELECT id,guid,COALESCE(title,''),COALESCE(description,''),COALESCE(kind,''),
		COALESCE(cover_url,''),COALESCE(published_at,0),COALESCE(duration_sec,0),COALESCE(file_path,''),COALESCE(enclosure_type,'')
		FROM episodes WHERE feed_id=? ORDER BY COALESCE(published_at,0) DESC, id DESC`, feedID)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	var items []item
	for rows.Next() {
		var it item
		if rows.Scan(&it.id, &it.guid, &it.title, &it.desc, &it.kind, &it.cover, &it.pub, &it.dur, &it.file, &it.enclType) == nil {
			items = append(items, it)
		}
	}
	rows.Close()
	base := baseURL(r)
	tok := apiToken(u.ID)
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<rss version="2.0" xmlns:itunes="http://www.itunes.com/dtds/podcast-1.0.dtd">` + "\n<channel>\n")
	writeElem(&b, "title", title)
	writeElem(&b, "link", url)
	writeElem(&b, "description", desc)
	writeElem(&b, "language", "zh-cn")
	writeElem(&b, "itunes:author", author)
	if cover != "" {
		fmt.Fprintf(&b, "<itunes:image href=%q/>\n", cover)
		fmt.Fprintf(&b, "<image><url>%s</url><title>%s</title><link>%s</link></image>\n",
			xmlEscape(cover), xmlEscape(title), xmlEscape(url))
	}
	for _, it := range items {
		b.WriteString("<item>\n")
		writeElem(&b, "title", it.title)
		writeElem(&b, "guid", it.guid)
		if it.pub > 0 {
			writeElem(&b, "pubDate", time.Unix(it.pub, 0).UTC().Format(time.RFC1123Z))
		}
		writeElem(&b, "description", it.desc)
		enclosure := base + "/stream/" + strconv.FormatInt(it.id, 10) + "?t=" + tok
		ctype := "audio/mpeg"
		size := int64(0)
		if it.kind == "text" {
			enclosure = base + "/speak/" + strconv.FormatInt(it.id, 10) + "?t=" + tok
			ctype = "audio/mpeg"
		} else if strings.Contains(strings.ToLower(it.enclType), "video") || it.kind == "video" {
			ctype = "audio/mp4"
		}
		if it.file != "" {
			if st, err := os.Stat(it.file); err == nil {
				size = st.Size()
				if strings.HasSuffix(strings.ToLower(it.file), ".mp3") {
					ctype = "audio/mpeg"
				} else {
					ctype = "audio/mp4"
				}
			}
		}
		fmt.Fprintf(&b, "<enclosure url=%q type=%q length=%d/>\n", enclosure, ctype, size)
		if it.dur > 0 {
			fmt.Fprintf(&b, "<itunes:duration>%d</itunes:duration>\n", it.dur)
		}
		if it.cover != "" {
			fmt.Fprintf(&b, "<itunes:image href=%q/>\n", it.cover)
		}
		b.WriteString("</item>\n")
	}
	b.WriteString("</channel>\n</rss>\n")
	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	_, _ = w.Write([]byte(b.String()))
}

func xmlEscape(s string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

func writeElem(b *strings.Builder, name, value string) {
	if value == "" {
		return
	}
	b.WriteString("<" + name + ">" + xmlEscape(value) + "</" + name + ">\n")
}

// handleGpodderInfo exposes what the settings page needs to show users how to
// connect a podcast client (sync URL + the per-feed proxy URLs).
func handleGpodderInfo(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	if u == nil {
		writeJSONStatus(w, http.StatusUnauthorized, map[string]any{"error": "unauthorized"})
		return
	}
	rows, err := store.db.Query(`SELECT id,COALESCE(title,''),url FROM feeds WHERE user_id=? ORDER BY title COLLATE NOCASE`, u.ID)
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	defer rows.Close()
	feeds := []map[string]any{}
	for rows.Next() {
		var id int64
		var title, url string
		if rows.Scan(&id, &title, &url) == nil {
			feeds = append(feeds, map[string]any{"id": id, "title": title, "url": proxyFeedURL(r, u.ID, id)})
		}
	}
	writeJSON(w, map[string]any{
		"sync_url": baseURL(r),
		"username": u.Name,
		"feeds":    feeds,
	})
}
