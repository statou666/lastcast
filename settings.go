package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func (s *Store) getSetting(key, def string) string {
	var v string
	err := s.db.QueryRow(`SELECT value FROM settings WHERE key=?`, key).Scan(&v)
	if err != nil || v == "" {
		return def
	}
	return v
}

func (s *Store) setSetting(key, val string) {
	_, _ = s.db.Exec(`INSERT INTO settings(key,value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, key, val)
}

// runtime settings (env vars seed defaults; UI edits persist here)
func settingsMap() map[string]string {
	return map[string]string{
		"fetch_interval_min":     env("FETCH_INTERVAL_MIN", "45"),
		"download_concurrency":   env("DOWNLOAD_CONCURRENCY", "2"),
		"conflict_rule":          env("CONFLICT_RULE", "max"), // max | latest
		"auto_download_new":      env("AUTO_DOWNLOAD_NEW", "0"),
		"storage_limit_mb":       env("STORAGE_LIMIT_MB", "0"),
		"ui_theme":               env("UI_THEME", "dark"),
		"segment_seconds":        env("SEGMENT_SECONDS", "30"),
		"ffmpeg_path":            env("FFMPEG_PATH", "ffmpeg"),
		"tts_provider":           env("TTS_PROVIDER", "off"), // off | openai (OpenAI-compatible /audio/speech)
		"tts_base_url":           env("TTS_BASE_URL", "https://api.openai.com/v1"),
		"tts_key":                env("TTS_KEY", ""),
		"tts_model":              env("TTS_MODEL", "tts-1"),
		"tts_voice":              env("TTS_VOICE", "alloy"),
		"tts_speed":              env("TTS_SPEED", "1.0"),
		"tts_max_chars":          env("TTS_MAX_CHARS", "3000"),
		"tts_auto":               env("TTS_AUTO", "1"),
		"play_mode":              env("PLAY_MODE", "stream"), // stream = Range proxy, no disk; segment = ffmpeg cache
		"cache_dir":              env("CACHE_DIR", "cache"),
		"download_dir":           env("DOWNLOAD_DIR", "downloads"),
		"cache_limit_mb":         env("CACHE_LIMIT_MB", "0"),
		"download_format":        env("DOWNLOAD_FORMAT", "auto"), // auto | mp3 | m4a
		"favorite_auto_download": env("FAVORITE_AUTO_DOWNLOAD", "0"),
		"http_proxy":             env("PROXY_URL", ""),
	}
}

func handleSettings(w http.ResponseWriter, r *http.Request) {
	admin := isAdmin(r)
	if r.Method == http.MethodPost {
		if !admin {
			writeJSONStatus(w, http.StatusForbidden, map[string]any{"error": "admin only"})
			return
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, map[string]any{"error": "invalid json"})
			return
		}
		for k, v := range body {
			store.setSetting(k, v)
		}
		if _, ok := body["http_proxy"]; ok {
			setProxy(store.getSetting("http_proxy", ""))
		}
		writeJSON(w, body)
		return
	}
	out := settingsMap()
	for k := range out {
		out[k] = store.getSetting(k, out[k])
	}
	out["is_admin"] = map[bool]string{true: "1", false: "0"}[admin]
	if !admin {
		out["tts_key"] = ""
		out["http_proxy"] = ""
	}
	writeJSON(w, out)
}

func handleConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{
		"port": cfg.Port, "data_dir": cfg.DataDir, "web_dir": cfg.WebDir,
		"env": map[string]string{
			"PORT": os.Getenv("PORT"), "DATA_DIR": os.Getenv("DATA_DIR"), "WEB_DIR": os.Getenv("WEB_DIR"),
			"FETCH_INTERVAL_MIN": os.Getenv("FETCH_INTERVAL_MIN"), "DOWNLOAD_CONCURRENCY": os.Getenv("DOWNLOAD_CONCURRENCY"),
			"CONFLICT_RULE": os.Getenv("CONFLICT_RULE"), "STORAGE_LIMIT_MB": os.Getenv("STORAGE_LIMIT_MB"),
			"CACHE_DIR": os.Getenv("CACHE_DIR"), "DOWNLOAD_DIR": os.Getenv("DOWNLOAD_DIR"),
			"PLAY_MODE": os.Getenv("PLAY_MODE"), "CACHE_LIMIT_MB": os.Getenv("CACHE_LIMIT_MB"),
		},
	})
}

func handleDevices(w http.ResponseWriter, r *http.Request) {
	rows, err := store.db.Query(`SELECT id,name,client_type,last_seen FROM devices WHERE user_id=? ORDER BY last_seen DESC`, userID(r))
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var id, name, ct sql.NullString
		var ts int64
		if err := rows.Scan(&id, &name, &ct, &ts); err != nil {
			continue
		}
		out = append(out, map[string]any{"id": id.String, "name": name.String, "client_type": ct.String, "last_seen": ts})
	}
	writeJSON(w, out)
}

func handleStats(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	var nEp, nDl int64
	_ = store.db.QueryRow(`SELECT COUNT(*) FROM episodes e JOIN feeds f ON f.id=e.feed_id WHERE f.user_id=?`, uid).Scan(&nEp)
	_ = store.db.QueryRow(`SELECT COUNT(*) FROM episodes e JOIN feeds f ON f.id=e.feed_id WHERE f.user_id=? AND e.status='downloaded'`, uid).Scan(&nDl)
	// Downloads are the persistent files; cache (segments/TTS chunks) is separate.
	dlCount, dlBytes := downloadStats(uid)
	cacheDir, cFiles, cBytes := cacheInfo()
	dbSize := int64(0)
	for _, name := range []string{"podcastman.db", "podcastman.db-wal"} {
		if st, err := os.Stat(filepath.Join(cfg.DataDir, name)); err == nil {
			dbSize += st.Size()
		}
	}
	limit := atoi(store.getSetting("cache_limit_mb", env("CACHE_LIMIT_MB", "0")))
	writeJSON(w, map[string]any{
		"episodes": nEp, "downloaded": nDl,
		"downloads": dlCount, "downloads_bytes": dlBytes, "media_bytes": dlBytes,
		"cache_dir": cacheDir, "cache_files": cFiles, "cache_bytes": cBytes, "cache_limit_mb": limit,
		"download_dir": downloadRoot(), "db_bytes": dbSize,
	})
}

// OPML export
func handleOpmlExport(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>` + "\n<opml version=\"2.0\">\n<head><title>Podcastman</title></head>\n<body>\n"))
	rows, err := store.db.Query(`SELECT COALESCE(title,''),url FROM feeds WHERE user_id=? ORDER BY title`, userID(r))
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var t, u string
		if err := rows.Scan(&t, &u); err != nil {
			continue
		}
		fmt.Fprintf(w, "<outline type=\"rss\" text=%q xmlUrl=%q/>\n", t, u)
	}
	w.Write([]byte("</body>\n</opml>\n"))
}

// handleOpmlImport walks nested outlines: a parent outline without xmlUrl is a
// category/folder, and its children inherit that name as their category.
// handleOpmlImport imports feeds from an OPML file and detects each feed's type
// (audio / video / text) from what it actually contains. Categories are left to
// the user (they are never inferred from OPML folders).
func handleOpmlImport(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(io.LimitReader(r.Body, 20<<20))
	if err != nil {
		writeJSON(w, map[string]any{"error": "cannot read request body"})
		return
	}
	if strings.TrimSpace(string(data)) == "" {
		writeJSON(w, map[string]any{"error": "empty body: choose a local .opml file or paste its content"})
		return
	}
	var urls []string
	dec := xml.NewDecoder(bytes.NewReader(data))
	seen := map[string]bool{}
	for {
		t, err := dec.Token()
		if err != nil {
			break
		}
		se, ok := t.(xml.StartElement)
		if !ok || se.Name.Local != "outline" {
			continue
		}
		u := attr(se.Attr, "xmlUrl")
		if u == "" || seen[u] {
			continue
		}
		seen[u] = true
		urls = append(urls, u)
	}
	uid := userID(r)
	added, existing := 0, 0
	kinds := map[string]int{"audio": 0, "video": 0, "text": 0, "unknown": 0}
	for _, u := range urls {
		var id int64
		if err := store.db.QueryRow(`SELECT id FROM feeds WHERE url=? AND user_id=?`, u, uid).Scan(&id); err != nil {
			res, err := store.db.Exec(`INSERT INTO feeds(user_id,url,kind,last_fetched) VALUES(?,?,'auto',?)`, uid, u, nowUnix())
			if err != nil {
				continue
			}
			id, _ = res.LastInsertId()
			added++
		} else {
			existing++
		}
		// Fetch once so the type can be detected from real items.
		if _, err := fetchAndUpsertFeed(id, u); err != nil {
			kinds["unknown"]++
			continue
		}
		var detected string
		var count int
		_ = store.db.QueryRow(`SELECT COALESCE(detected,''), (SELECT COUNT(*) FROM episodes WHERE feed_id=?) FROM feeds WHERE id=?`, id, id).
			Scan(&detected, &count)
		if count == 0 || (detected != "audio" && detected != "video" && detected != "text") {
			kinds["unknown"]++
			continue
		}
		// Pin the detected type (only while the feed is still automatic).
		_, _ = store.db.Exec(`UPDATE feeds SET kind=? WHERE id=? AND user_id=? AND kind='auto'`, detected, id, uid)
		kinds[detected]++
	}
	writeJSON(w, map[string]any{
		"parsed": len(urls), "added": added, "existing": existing, "kinds": kinds,
		"audio": kinds["audio"], "video": kinds["video"], "text": kinds["text"], "unknown": kinds["unknown"],
	})
}

// deleteFeedForUser removes one subscription (episodes, progress, favorites,
// queue, cache) but never touches finished downloads.
func deleteFeedForUser(uid, id int64) (int, bool) {
	if !feedOwned(uid, id) {
		return 0, false
	}
	var kept int
	_ = store.db.QueryRow(`SELECT COUNT(*) FROM downloads WHERE feed_id=? AND user_id=?`, id, uid).Scan(&kept)
	eps, err := store.db.Query(`SELECT id FROM episodes WHERE feed_id=?`, id)
	if err == nil {
		var ids []int64
		for eps.Next() {
			var eid int64
			if err := eps.Scan(&eid); err == nil {
				ids = append(ids, eid)
			}
		}
		eps.Close()
		for _, eid := range ids {
			removeEpisodeCache(eid)
			_, _ = store.db.Exec(`DELETE FROM queue WHERE episode_id=? AND user_id=?`, eid, uid)
		}
	}
	_, _ = store.db.Exec(`DELETE FROM favorites WHERE user_id=? AND episode_id IN (SELECT id FROM episodes WHERE feed_id=?)`, uid, id)
	_, _ = store.db.Exec(`DELETE FROM progress WHERE user_id=? AND episode_id IN (SELECT id FROM episodes WHERE feed_id=?)`, uid, id)
	_, _ = store.db.Exec(`DELETE FROM episodes WHERE feed_id=?`, id)
	_, _ = store.db.Exec(`DELETE FROM feeds WHERE id=? AND user_id=?`, id, uid)
	store.logEvent("feed", itoa(id), "deleted", uid)
	return kept, true
}

func handleDeleteFeed(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	id := int64(atoi(r.PathValue("feed_id")))
	kept, ok := deleteFeedForUser(uid, id)
	if !ok {
		writeJSON(w, map[string]any{"error": "feed not found"})
		return
	}
	writeJSON(w, map[string]any{"deleted": id, "kept_downloads": kept})
}

func handleRefetch(w http.ResponseWriter, r *http.Request) {
	var body struct {
		FeedID int64 `json:"feed_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, map[string]any{"error": "invalid json"})
		return
	}
	var refetchURL string
	if err := store.db.QueryRow(`SELECT url FROM feeds WHERE id=? AND user_id=?`, body.FeedID, userID(r)).Scan(&refetchURL); err != nil {
		writeJSON(w, map[string]any{"error": "feed not found"})
		return
	}
	newItems, err := fetchAndUpsertFeed(body.FeedID, refetchURL)
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]any{"feed_id": body.FeedID, "new_items": newItems})
}
