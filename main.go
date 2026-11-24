package main

import (
	"archive/zip"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

type Config struct {
	Port          string
	DataDir       string
	WebDir        string
	FetchInterval time.Duration
}

var (
	cfg     Config
	store   *Store
	dl      *Downloader
	hclient = &http.Client{Timeout: 60 * time.Second}
	proxyAt = atomic.Value{} // *url.URL or nil
)

// setProxy reconfigures the shared transports; "" clears the proxy.
func setProxy(raw string) {
	raw = strings.TrimSpace(raw)
	var u *url.URL
	if raw != "" {
		if parsed, err := url.Parse(raw); err == nil && parsed.Host != "" {
			u = parsed
		}
	}
	proxyAt.Store(u)
}

func proxyFunc(req *http.Request) (*url.URL, error) {
	if v := proxyAt.Load(); v != nil {
		if u, _ := v.(*url.URL); u != nil {
			return u, nil
		}
	}
	return http.ProxyFromEnvironment(req)
}

func newTransport() *http.Transport {
	return &http.Transport{
		Proxy:                 proxyFunc,
		ResponseHeaderTimeout: 30 * time.Second,
		ForceAttemptHTTP2:     true,
	}
}

func main() {
	cfg = Config{
		Port:          env("PORT", "8563"),
		DataDir:       env("DATA_DIR", "./data"),
		WebDir:        env("WEB_DIR", "./web"),
		FetchInterval: time.Duration(atoi(env("FETCH_INTERVAL_MIN", "45"))) * time.Minute,
	}
	var err error
	store, err = openStore(filepath.Join(cfg.DataDir, "podcastman.db"))
	if err != nil {
		fatal(err)
	}
	hclient.Transport = newTransport()
	streamClient.Transport = newTransport()
	setProxy(store.getSetting("http_proxy", ""))
	ensureRoots()
	migrateMediaLayout()
	backfillCovers()
	evictCache()
	dl = NewDownloader(store, cfg.DataDir)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go dl.Run(ctx)
	go scheduler(ctx)
	go fullTextWorkers(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"ok": true, "time": time.Now().Unix()})
	})
	mux.HandleFunc("GET /api/feeds", handleListFeeds)
	mux.HandleFunc("POST /api/feeds", handleAddFeed)
	mux.HandleFunc("GET /api/episodes", handleListEpisodes)
	mux.HandleFunc("GET /api/episodes/{id}", handleGetEpisode)
	mux.HandleFunc("GET /api/feeds/{id}", handleGetFeed)
	mux.HandleFunc("POST /api/feeds/{id}", handleUpdateFeed)
	mux.HandleFunc("POST /api/download", handleDownload)
	mux.HandleFunc("GET /api/progress", handleGetProgress)
	mux.HandleFunc("POST /api/progress", handlePostProgress)
	mux.HandleFunc("GET /api/sync", handleSync)
	mux.HandleFunc("GET /api/export/{feed_id}", handleExport)
	mux.HandleFunc("GET /api/pack/{feed_id}", handlePack)
	mux.HandleFunc("GET /media/{episode_id}", handleMedia)
	mux.HandleFunc("GET /media/{episode_id}/{seg}", handleSegment)
	mux.HandleFunc("GET /api/segments/{id}", handleSegments)
	mux.HandleFunc("GET /api/events", handleEvents)
	mux.HandleFunc("GET /api/settings", handleSettings)
	mux.HandleFunc("POST /api/settings", handleSettings)
	mux.HandleFunc("POST /api/tts", handleTTS)
	mux.HandleFunc("POST /api/episodes/{id}/fulltext", handleEpisodeFullText)
	mux.HandleFunc("POST /api/episodes/{id}/segments", handleEpisodeSegments)
	mux.HandleFunc("POST /api/tts/test", handleTTS)
	mux.HandleFunc("GET /api/config", handleConfig)
	mux.HandleFunc("GET /api/devices", handleDevices)
	mux.HandleFunc("GET /api/stats", handleStats)
	mux.HandleFunc("GET /api/opml", handleOpmlExport)
	mux.HandleFunc("POST /api/opml", handleOpmlImport)
	mux.HandleFunc("DELETE /api/feeds/{feed_id}", handleDeleteFeed)
	mux.HandleFunc("POST /api/refetch", handleRefetch)
	mux.HandleFunc("/api/queue", handleQueue)
	mux.HandleFunc("/api/queue/{id}", handleQueue)
	mux.HandleFunc("POST /api/queue/clear", handleQueueClear)
	mux.HandleFunc("/login", handleLogin)
	mux.HandleFunc("/setup", handleSetup)
	mux.HandleFunc("/api/login", handleLogin)
	mux.HandleFunc("/api/setup", handleSetup)
	mux.HandleFunc("/api/logout", handleLogout)
	mux.HandleFunc("/api/me", handleMe)
	mux.HandleFunc("POST /api/me/password", handlePassword)
	mux.HandleFunc("GET /api/tts/voices", handleTTSVoices)
	mux.HandleFunc("POST /api/batch", handleBatch)
	mux.HandleFunc("GET /api/gpodder", handleGpodderInfo)
	mux.HandleFunc("/api/2/", gpodRouter)
	mux.HandleFunc("/gpodder/feed/", gpodFeed)
	mux.HandleFunc("GET /speak/{id}", handleSpeak)
	mux.HandleFunc("/api/users", handleUsers)
	mux.HandleFunc("DELETE /api/users/{id}", handleUsers)
	mux.HandleFunc("GET /stream/{id}", handleStream)
	mux.HandleFunc("GET /api/downloads", handleDownloads)
	mux.HandleFunc("GET /api/downloads/{id}/file", handleDownloadItem)
	mux.HandleFunc("DELETE /api/downloads/{id}", handleDownloadItem)
	mux.HandleFunc("POST /api/downloads/retag", handleRetagDownloads)
	mux.HandleFunc("GET /api/favorites", handleFavorites)
	mux.HandleFunc("POST /api/favorites", handleFavorites)
	mux.HandleFunc("DELETE /api/favorites/{id}", handleFavorites)
	mux.HandleFunc("POST /api/queue/{id}/move", handleQueueMove)
	mux.HandleFunc("POST /api/queue/shuffle", handleQueueShuffle)
	mux.HandleFunc("/api/cache", handleCache)
	if err := store.ensureQueue(); err != nil {
		fatal(err)
	}
	ensureTokens()
	// The SPA root is served explicitly and every static file falls back to the
	// embedded copy, so an unreadable host web directory can never leave users on
	// a bare directory listing.
	mux.HandleFunc("GET /{$}", handleIndex)
	mux.HandleFunc("GET /index.html", handleIndex)
	mux.Handle("/", http.FileServer(http.FS(diskThenEmbed{dir: cfg.WebDir})))

	fatal(http.ListenAndServe(":"+cfg.Port, withAuth(mux)))
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func nowUnix() int64 { return time.Now().Unix() }

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(v)
}

func fatal(err error) {
	if err != nil {
		panic(err)
	}
}

type FeedJSON struct {
	ID            int64  `json:"id"`
	URL           string `json:"url"`
	Kind          string `json:"kind"`    // what the app will actually use
	Setting       string `json:"setting"` // what the user chose (auto = follow detection)
	Detected      string `json:"detected,omitempty"`
	Title         string `json:"title"`
	Description   string `json:"description"`
	CoverURL      string `json:"cover_url"`
	LastFetched   int64  `json:"last_fetched"`
	Episodes      int64  `json:"episodes"`
	Downloaded    int64  `json:"downloaded"`
	Category      string `json:"category,omitempty"`
	AutoDownload  bool   `json:"auto_download"`
	FetchFulltext bool   `json:"fetch_fulltext"`
}

type EpisodeJSON struct {
	ID            int64    `json:"id"`
	QueueID       int64    `json:"queue_id,omitempty"`
	FeedID        int64    `json:"feed_id"`
	FeedTitle     string   `json:"feed_title,omitempty"`
	CoverURL      string   `json:"cover_url,omitempty"`
	Guid          string   `json:"guid"`
	Link          string   `json:"link,omitempty"`
	Title         string   `json:"title"`
	Kind          string   `json:"kind"`
	Body          string   `json:"body,omitempty"`
	Chunks        int      `json:"chunks,omitempty"`
	VideoURL      string   `json:"video_url,omitempty"`
	Description   string   `json:"description,omitempty"`
	EnclosureURL  string   `json:"enclosure_url"`
	PublishedAt   int64    `json:"published_at"`
	DurationSec   int64    `json:"duration_sec"`
	Status        string   `json:"status"`
	File          string   `json:"file,omitempty"`
	Position      int64    `json:"position"`
	Completed     bool     `json:"completed"`
	Segments      int      `json:"segments,omitempty"`
	SegmentsDone  int      `json:"segments_done,omitempty"`
	SegmentIdx    int      `json:"segment_idx,omitempty"`
	UpdatedAt     int64    `json:"updated_at,omitempty"`
	Favorite      bool     `json:"favorite,omitempty"`
	Order         int      `json:"order,omitempty"`
	AddedAt       int64    `json:"added_at,omitempty"`
	FeedUpdatedAt int64    `json:"feed_updated_at,omitempty"`
	FullText      string   `json:"full_text,omitempty"`
	Images        []string `json:"images,omitempty"`
}

func handleListFeeds(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	rows, err := store.db.Query(`SELECT f.id,f.url,f.kind,COALESCE(f.detected,''),f.title,f.description,f.cover_url,f.last_fetched,
		COALESCE(c.n,0), COALESCE(c.d,0), COALESCE(f.category,''), COALESCE(f.auto_download,0), COALESCE(f.fetch_fulltext,0)
		FROM feeds f
		LEFT JOIN (SELECT feed_id, COUNT(*) AS n, SUM(CASE WHEN status='downloaded' THEN 1 ELSE 0 END) AS d
		           FROM episodes GROUP BY feed_id) c ON c.feed_id=f.id
		WHERE f.user_id=? ORDER BY f.title COLLATE NOCASE`, uid)
	if err != nil {
		fatal(err)
	}
	defer rows.Close()
	var out []FeedJSON
	for rows.Next() {
		var f FeedJSON
		var title, desc, cover, det sql.NullString
		var auto, full int
		if err := rows.Scan(&f.ID, &f.URL, &f.Setting, &det, &title, &desc, &cover, &f.LastFetched, &f.Episodes, &f.Downloaded, &f.Category, &auto, &full); err != nil {
			writeJSON(w, map[string]any{"error": err.Error()})
			return
		}
		f.Detected = det.String
		f.Kind = f.Setting // what the user chose; 'auto' means follow the parser
		if f.Kind == "auto" && f.Detected != "" {
			f.Kind = f.Detected
		}
		f.Title, f.Description, f.CoverURL = title.String, desc.String, cover.String
		f.AutoDownload = auto == 1
		f.FetchFulltext = full == 1
		out = append(out, f)
	}
	writeJSON(w, out)
}

func handleAddFeed(w http.ResponseWriter, r *http.Request) {
	var body struct {
		URL      string `json:"url"`
		Kind     string `json:"kind"` // auto | audio | video | text
		Category string `json:"category"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, map[string]any{"error": "invalid json"})
		return
	}
	if body.URL == "" {
		writeJSON(w, map[string]any{"error": "url required"})
		return
	}
	kind := body.Kind
	if kind == "" {
		kind = "auto"
	}
	if kind != "auto" && kind != "audio" && kind != "video" && kind != "text" {
		writeJSON(w, map[string]any{"error": "kind must be auto, audio, video or text"})
		return
	}
	uid := userID(r)
	var id int64
	err := store.db.QueryRow(`SELECT id FROM feeds WHERE url=? AND user_id=?`, body.URL, uid).Scan(&id)
	if err != nil {
		res, err := store.db.Exec(`INSERT INTO feeds(user_id,url,kind,category,last_fetched) VALUES(?,?,?,?,?)`, uid, body.URL, kind, body.Category, nowUnix())
		if err != nil {
			fatal(err)
		}
		id, _ = res.LastInsertId()
	} else {
		if body.Kind != "" {
			_, _ = store.db.Exec(`UPDATE feeds SET kind=? WHERE id=? AND user_id=?`, kind, id, uid)
		}
		if body.Category != "" {
			_, _ = store.db.Exec(`UPDATE feeds SET category=? WHERE id=? AND user_id=?`, body.Category, id, uid)
		}
	}
	if _, err := fetchAndUpsertFeed(id, body.URL); err != nil {
		writeJSON(w, map[string]any{"id": id, "error": err.Error()})
		return
	}
	out, ok := loadFeedJSON(id, userID(r))
	if !ok {
		writeJSON(w, map[string]any{"id": id, "error": "feed not found"})
		return
	}
	writeJSON(w, out)
}

// loadFeedJSON is the single place that renders a feed row for the API.
func loadFeedJSON(id, uid int64) (FeedJSON, bool) {
	var f FeedJSON
	var det, title, desc, cover sql.NullString
	var auto, full int
	err := store.db.QueryRow(`SELECT id,COALESCE(url,''),kind,COALESCE(detected,''),COALESCE(title,''),COALESCE(description,''),COALESCE(cover_url,''),last_fetched,
		(SELECT COUNT(*) FROM episodes e WHERE e.feed_id=f.id),
		(SELECT COUNT(*) FROM episodes e WHERE e.feed_id=f.id AND e.status='downloaded'),
		COALESCE(category,''), COALESCE(auto_download,0), COALESCE(fetch_fulltext,0)
		FROM feeds f WHERE id=? AND user_id=?`, id, uid).
		Scan(&f.ID, &f.URL, &f.Setting, &det, &title, &desc, &cover, &f.LastFetched, &f.Episodes, &f.Downloaded, &f.Category, &auto, &full)
	if err != nil {
		return f, false
	}
	f.Detected, f.Title, f.Description, f.CoverURL = det.String, title.String, desc.String, cover.String
	f.AutoDownload = auto == 1
	f.FetchFulltext = full == 1
	f.Kind = f.Setting
	if f.Kind == "auto" && f.Detected != "" {
		f.Kind = f.Detected
	}
	return f, true
}

// handleUpdateFeed edits a subscription: url, category, kind and its own
// auto-download switch (independent of the global auto_download_new setting).
func handleUpdateFeed(w http.ResponseWriter, r *http.Request) {
	id := int64(atoi(r.PathValue("id")))
	if id == 0 {
		writeJSON(w, map[string]any{"error": "bad id"})
		return
	}
	var body struct {
		URL           *string `json:"url"`
		Category      *string `json:"category"`
		Kind          *string `json:"kind"`
		AutoDownload  *bool   `json:"auto_download"`
		FetchFulltext *bool   `json:"fetch_fulltext"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, map[string]any{"error": "invalid json"})
		return
	}
	uid := userID(r)
	var oldURL string
	if err := store.db.QueryRow(`SELECT COALESCE(url,'') FROM feeds WHERE id=? AND user_id=?`, id, uid).Scan(&oldURL); err != nil {
		writeJSON(w, map[string]any{"error": "feed not found"})
		return
	}
	refetch := false
	url := oldURL
	if body.URL != nil {
		u := strings.TrimSpace(*body.URL)
		if u != "" && u != oldURL {
			if _, err := store.db.Exec(`UPDATE feeds SET url=?, etag='', last_fetched=0 WHERE id=? AND user_id=?`, u, id, uid); err != nil {
				writeJSON(w, map[string]any{"error": err.Error()})
				return
			}
			url, refetch = u, true
		}
	}
	if body.Category != nil {
		_, _ = store.db.Exec(`UPDATE feeds SET category=? WHERE id=? AND user_id=?`, strings.TrimSpace(*body.Category), id, uid)
	}
	if body.Kind != nil {
		k := *body.Kind
		if k != "auto" && k != "audio" && k != "video" && k != "text" {
			writeJSON(w, map[string]any{"error": "kind must be auto, audio, video or text"})
			return
		}
		_, _ = store.db.Exec(`UPDATE feeds SET kind=? WHERE id=? AND user_id=?`, k, id, uid)
		if k != "auto" {
			// an explicit choice also reclassifies already-known episodes
			_, _ = store.db.Exec(`UPDATE episodes SET kind=? WHERE feed_id=?`, k, id)
		}
	}
	if body.AutoDownload != nil {
		v := 0
		if *body.AutoDownload {
			v = 1
		}
		_, _ = store.db.Exec(`UPDATE feeds SET auto_download=? WHERE id=? AND user_id=?`, v, id, uid)
	}
	if body.FetchFulltext != nil {
		v := 0
		if *body.FetchFulltext {
			v = 1
		}
		_, _ = store.db.Exec(`UPDATE feeds SET fetch_fulltext=? WHERE id=? AND user_id=?`, v, id, uid)
		if v == 1 {
			rows, err := store.db.Query(`SELECT id FROM episodes WHERE feed_id=? AND kind='text' AND COALESCE(full_text,'')='' ORDER BY published_at DESC LIMIT 10`, id)
			if err == nil {
				for rows.Next() {
					var eid int64
					if rows.Scan(&eid) == nil {
						enqueueFullText(eid)
					}
				}
				rows.Close()
			}
		}
	}
	note := ""
	if refetch {
		if n, err := fetchAndUpsertFeed(id, url); err != nil {
			note = err.Error()
		} else {
			note = fmt.Sprintf("new_items=%d", n)
		}
	}
	out, ok := loadFeedJSON(id, uid)
	if !ok {
		writeJSON(w, map[string]any{"error": "feed not found"})
		return
	}
	writeJSON(w, map[string]any{"feed": out, "note": note})
}

// needsDownload reports whether an episode still has no finished file anywhere.
func needsDownload(id int64) bool {
	var path, status string
	if err := store.db.QueryRow(`SELECT COALESCE(file_path,''), COALESCE(status,'') FROM episodes WHERE id=?`, id).
		Scan(&path, &status); err != nil {
		return false
	}
	if path != "" {
		if st, err := os.Stat(path); err == nil && st.Size() > 0 {
			return false
		}
	}
	if status == "working" || status == "queued" {
		return false
	}
	var n int
	_ = store.db.QueryRow(`SELECT COUNT(*) FROM downloads WHERE episode_id=? AND bytes>0`, id).Scan(&n)
	return n == 0
}

// shouldAuto decides whether a newly seen episode is downloaded automatically:
// the feed's own switch wins, otherwise the global one; text items also need TTS.
func shouldAuto(kind string, feedAuto bool) bool {
	if !feedAuto && store.getSetting("auto_download_new", "0") != "1" {
		return false
	}
	if kind == "text" {
		return ttsConfig().Provider != "off" && store.getSetting("tts_auto", "1") == "1"
	}
	return true
}

func fetchAndUpsertFeed(feedID int64, url string) (int, error) {
	var etag, feedKind string
	var auto, fetchFull int
	if err := store.db.QueryRow(`SELECT COALESCE(etag,''), COALESCE(kind,'auto'), COALESCE(auto_download,0), COALESCE(fetch_fulltext,0) FROM feeds WHERE id=?`, feedID).
		Scan(&etag, &feedKind, &auto, &fetchFull); err != nil {
		feedKind = "auto"
	}
	feedAuto := auto == 1
	fetchFullText := fetchFull == 1
	meta, items, newEtag, notMod, err := fetchFeed(hclient, url, etag)
	if err != nil {
		return 0, err
	}
	if notMod {
		return 0, nil
	}
	setting := feedKind // 'auto' or an explicit choice from the subscribe form
	if feedKind == "auto" {
		feedKind = meta.Kind
	}
	_, _ = store.db.Exec(`UPDATE feeds SET detected=?,
		title=COALESCE(NULLIF(?,''),title), description=COALESCE(NULLIF(?,''),description),
		cover_url=COALESCE(NULLIF(?,''),cover_url), author=COALESCE(NULLIF(?,''),author),
		etag=?, last_fetched=? WHERE id=?`,
		meta.Kind, meta.Title, meta.Description, meta.CoverURL, meta.Author, newEtag, nowUnix(), feedID)
	new := 0
	ftEnqueued := 0
	for _, it := range items {
		kind := it.Kind
		if setting != "auto" {
			kind = setting // an explicit choice in the subscribe form wins over detection
		}
		now := nowUnix()
		imgs := dedupeImages(extractImages(it.Body+" "+it.Description, it.Link, ""))
		cover := it.CoverURL
		if cover == "" && len(imgs) > 0 {
			cover = imgs[0]
		}
		res, err := store.db.Exec(`INSERT OR IGNORE INTO episodes(feed_id,guid,title,description,body,enclosure_url,enclosure_type,video_url,kind,published_at,duration_sec,author,cover_url,link,images,added_at,updated_at)
			VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, feedID, it.Guid, it.Title, it.Description, it.Body, it.EnclosureURL, it.EnclosureType, it.VideoURL, kind, it.PublishedAt, it.DurationSec, it.Author, cover, it.Link, encodeImages(imgs), now, now)
		if err != nil {
			continue
		}
		if n, _ := res.RowsAffected(); n > 0 {
			new++
			id, _ := res.LastInsertId()
			store.logEvent("episode", it.Guid, "added")
			if shouldAuto(kind, feedAuto) && needsDownload(id) {
				dl.Enqueue(id)
			}
			if fetchFullText && kind == "text" && ftEnqueued < 8 {
				enqueueFullText(id)
				ftEnqueued++
			}
			continue
		}
		// Existing item: refresh content, and bump updated_at only when it changed.
		_, _ = store.db.Exec(`UPDATE episodes SET
			title=?, description=?, body=?, enclosure_url=?, enclosure_type=?, video_url=?, kind=?,
			published_at=CASE WHEN ?>0 THEN ? ELSE published_at END,
			duration_sec=CASE WHEN duration_sec=0 AND ?>0 THEN ? ELSE duration_sec END,
			author=CASE WHEN COALESCE(author,'')='' THEN ? ELSE author END,
			cover_url=CASE WHEN COALESCE(cover_url,'')='' THEN ? ELSE cover_url END,
			link=CASE WHEN COALESCE(link,'')='' THEN ? ELSE link END,
			images=CASE WHEN COALESCE(images,'')='' THEN ? ELSE images END,
			updated_at=?
			WHERE feed_id=? AND guid=? AND (
				COALESCE(title,'')<>? OR COALESCE(description,'')<>? OR COALESCE(body,'')<>? OR
				COALESCE(enclosure_url,'')<>? OR COALESCE(enclosure_type,'')<>? OR
				COALESCE(video_url,'')<>? OR COALESCE(kind,'')<>?)`,
			it.Title, it.Description, it.Body, it.EnclosureURL, it.EnclosureType, it.VideoURL, kind,
			it.PublishedAt, it.PublishedAt,
			it.DurationSec, it.DurationSec,
			it.Author, cover, it.Link, encodeImages(imgs), now,
			feedID, it.Guid,
			it.Title, it.Description, it.Body, it.EnclosureURL, it.EnclosureType, it.VideoURL, kind)
		if fetchFullText && kind == "text" && ftEnqueued < 8 {
			var id int64
			if store.db.QueryRow(`SELECT id FROM episodes WHERE feed_id=? AND guid=? AND COALESCE(full_text,'')=''`, feedID, it.Guid).Scan(&id) == nil {
				enqueueFullText(id)
				ftEnqueued++
			}
		}
	}
	return new, nil
}

func scheduler(ctx context.Context) {
	for {
		rows, err := store.db.Query(`SELECT id,url,etag FROM feeds`)
		if err != nil {
			return
		}
		type fe struct {
			id   int64
			url  string
			etag string
		}
		var feeds []fe
		for rows.Next() {
			var f fe
			if err := rows.Scan(&f.id, &f.url, &f.etag); err == nil {
				feeds = append(feeds, f)
			}
		}
		rows.Close()
		for _, f := range feeds {
			if ctx.Err() != nil {
				return
			}
			_, _ = fetchAndUpsertFeed(f.id, f.url)
			if len(feeds) > 20 {
				time.Sleep(2 * time.Second) // politeness gap for big feeds
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Duration(atoi(store.getSetting("fetch_interval_min", "45"))) * time.Minute):
		}
	}
}

func handleListEpisodes(w http.ResponseWriter, r *http.Request) {
	feedID := atoi(r.URL.Query().Get("feed_id"))
	limit := atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 2000 {
		limit = 500
	}
	uid := userID(r)
	q := `SELECT e.id,e.feed_id,COALESCE(f.title,''),e.guid,COALESCE(e.title,''),e.kind,COALESCE(e.video_url,''),COALESCE(e.description,''),COALESCE(e.enclosure_url,''),e.published_at,e.duration_sec,e.status,COALESCE(e.file_path,''),COALESCE(e.segments,0),COALESCE(e.segments_done,0),
		EXISTS(SELECT 1 FROM favorites fv WHERE fv.episode_id=e.id AND fv.user_id=?), COALESCE(e.added_at,0), COALESCE(e.updated_at,0),
		COALESCE(NULLIF(e.cover_url,''), COALESCE(NULLIF(f.cover_url,''),''))
		FROM episodes e JOIN feeds f ON f.id=e.feed_id WHERE f.user_id=?`
	var rows *sql.Rows
	var err error
	if feedID > 0 {
		rows, err = store.db.Query(q+` AND e.feed_id=? ORDER BY (e.published_at IS NULL OR e.published_at = 0), e.published_at DESC, e.added_at DESC, e.id DESC LIMIT ?`, uid, uid, feedID, limit)
	} else {
		rows, err = store.db.Query(q+` ORDER BY (e.published_at IS NULL OR e.published_at = 0), e.published_at DESC, e.added_at DESC, e.id DESC LIMIT ?`, uid, uid, limit)
	}
	if err != nil {
		fatal(err)
	}
	defer rows.Close()
	var out []EpisodeJSON
	for rows.Next() {
		var e EpisodeJSON
		var desc, file, video sql.NullString
		var fav int
		if err := rows.Scan(&e.ID, &e.FeedID, &e.FeedTitle, &e.Guid, &e.Title, &e.Kind, &video, &desc, &e.EnclosureURL,
			&e.PublishedAt, &e.DurationSec, &e.Status, &file, &e.Segments, &e.SegmentsDone, &fav, &e.AddedAt, &e.FeedUpdatedAt, &e.CoverURL); err != nil {
			writeJSON(w, map[string]any{"error": err.Error()})
			return
		}
		e.Description, e.File, e.VideoURL = desc.String, file.String, video.String
		e.Favorite = fav == 1
		out = append(out, e)
	}
	writeJSON(w, out)
}

func handleDownload(w http.ResponseWriter, r *http.Request) {
	var body struct {
		EpisodeID int64 `json:"episode_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, map[string]any{"error": "invalid json"})
		return
	}
	var id int64
	if err := store.db.QueryRow(`SELECT e.id FROM episodes e JOIN feeds f ON f.id=e.feed_id WHERE e.id=? AND f.user_id=?`,
		body.EpisodeID, userID(r)).Scan(&id); err != nil {
		writeJSON(w, map[string]any{"error": "episode not found"})
		return
	}
	dl.Enqueue(id)
	writeJSON(w, map[string]any{"queued": true, "episode_id": id})
}

func handlePostProgress(w http.ResponseWriter, r *http.Request) {
	var raw map[string]any
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		writeJSON(w, map[string]any{"error": "invalid json"})
		return
	}
	num := func(v any) int64 {
		switch n := v.(type) {
		case float64:
			return int64(n)
		case bool:
			if n {
				return 1
			}
			return 0
		}
		return 0
	}
	deviceID, _ := raw["device_id"].(string)
	episodeID := num(raw["episode_id"])
	position := num(raw["position"])
	if position == 0 {
		position = num(raw["position_sec"])
	}
	completed := num(raw["completed"])
	segIdx := num(raw["segment_idx"])
	if deviceID == "" || episodeID == 0 {
		writeJSON(w, map[string]any{"error": "device_id and episode_id required"})
		return
	}
	uid := userID(r)
	if !episodeOwned(uid, episodeID) {
		writeJSON(w, map[string]any{"error": "episode not found"})
		return
	}
	rule := store.getSetting("conflict_rule", "max")
	var err error
	if rule == "latest" {
		_, err = store.db.Exec(`INSERT INTO progress(user_id,episode_id,device_id,position_sec,completed,segment_idx,updated_at)
			VALUES(?,?,?,?,?,?,?)
			ON CONFLICT(user_id,episode_id,device_id) DO UPDATE SET
				position_sec=excluded.position_sec,
				completed=MAX(completed, excluded.completed),
				segment_idx=excluded.segment_idx,
				updated_at=excluded.updated_at`,
			uid, episodeID, deviceID, position, completed, segIdx, nowUnix())
	} else {
		_, err = store.db.Exec(`INSERT INTO progress(user_id,episode_id,device_id,position_sec,completed,segment_idx,updated_at)
			VALUES(?,?,?,?,?,?,?)
			ON CONFLICT(user_id,episode_id,device_id) DO UPDATE SET
				position_sec=MAX(position_sec, excluded.position_sec),
				completed=MAX(completed, excluded.completed),
				segment_idx=MAX(segment_idx, excluded.segment_idx),
				updated_at=excluded.updated_at`,
			uid, episodeID, deviceID, position, completed, segIdx, nowUnix())
	}
	if err != nil {
		fatal(err)
	}
	// Players report the media duration once they know it: free backfill for
	// feeds whose RSS carries no duration (and for streamed video).
	if dur := num(raw["duration_sec"]); dur > 0 {
		_, _ = store.db.Exec(`UPDATE episodes SET duration_sec=? WHERE id=? AND (duration_sec IS NULL OR duration_sec=0)`, dur, episodeID)
	}
	writeJSON(w, map[string]any{"ok": true})
}

func handleGetProgress(w http.ResponseWriter, r *http.Request) {
	ep := atoi(r.URL.Query().Get("episode_id"))
	rows, err := store.db.Query(`SELECT device_id,position_sec,completed,COALESCE(segment_idx,0),updated_at FROM progress WHERE user_id=? AND episode_id=?`, userID(r), ep)
	if err != nil {
		fatal(err)
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var d string
		var pos, comp, seg int
		var up int64
		if err := rows.Scan(&d, &pos, &comp, &seg, &up); err != nil {
			fatal(err)
		}
		out = append(out, map[string]any{"device_id": d, "position": pos, "completed": comp == 1, "segment": seg, "updated_at": up})
	}
	writeJSON(w, out)
}

func handleGetEpisode(w http.ResponseWriter, r *http.Request) {
	id := atoi(r.PathValue("id"))
	dev := r.URL.Query().Get("device_id")
	var e EpisodeJSON
	var ft, body, video, desc, file, cover sql.NullString
	var pos, comp, segIdx int64
	var fav int
	uid := userID(r)
	var link, fullText, imagesJSON sql.NullString
	err := store.db.QueryRow(`SELECT e.id,e.feed_id,f.title,e.guid,e.title,e.kind,e.body,e.video_url,e.description,e.enclosure_url,e.published_at,e.duration_sec,e.status,e.file_path,COALESCE(p.position_sec,0),COALESCE(p.completed,0),COALESCE(p.segment_idx,0),COALESCE(NULLIF(e.cover_url,''), COALESCE(NULLIF(f.cover_url,''),'')),COALESCE(e.segments,0),COALESCE(e.segments_done,0),
		EXISTS(SELECT 1 FROM favorites fv WHERE fv.episode_id=e.id AND fv.user_id=?), COALESCE(e.added_at,0), COALESCE(e.updated_at,0),
		COALESCE(e.link,''), COALESCE(e.full_text,''), COALESCE(e.images,'')
		FROM episodes e JOIN feeds f ON f.id=e.feed_id
		LEFT JOIN progress p ON p.episode_id=e.id AND p.device_id=? AND p.user_id=? WHERE e.id=? AND f.user_id=?`, uid, dev, uid, id, uid).
		Scan(&e.ID, &e.FeedID, &ft, &e.Guid, &e.Title, &e.Kind, &body, &video, &desc, &e.EnclosureURL,
			&e.PublishedAt, &e.DurationSec, &e.Status, &file, &pos, &comp, &segIdx, &cover, &e.Segments, &e.SegmentsDone, &fav, &e.AddedAt, &e.FeedUpdatedAt,
			&link, &fullText, &imagesJSON)
	if err != nil {
		writeJSON(w, map[string]any{"error": "episode not found"})
		return
	}
	e.FeedTitle = plainText(ft.String)
	e.Link = link.String
	e.FullText = fullText.String
	e.Images = decodeImages(imagesJSON.String)
	e.Body = plainText(e.FullText)
	if e.Body == "" {
		e.Body = plainText(body.String)
	}
	if e.Body == "" {
		e.Body = plainText(desc.String)
	}
	e.Chunks = len(chunks(e.Body, ttsConfig().MaxChars))
	e.Description, e.File, e.VideoURL, e.CoverURL = desc.String, file.String, video.String, cover.String
	e.Position, e.Completed, e.SegmentIdx = pos, comp == 1, int(segIdx)
	e.Favorite = fav == 1
	writeJSON(w, e)
}

func handleGetFeed(w http.ResponseWriter, r *http.Request) {
	f, ok := loadFeedJSON(int64(atoi(r.PathValue("id"))), userID(r))
	if !ok {
		writeJSON(w, map[string]any{"error": "feed not found"})
		return
	}
	writeJSON(w, f)
}

func handleSync(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	dev := q.Get("device_id")
	feedFilter := atoi(q.Get("feed_id"))
	uid := userID(r)
	if dev != "" {
		_, _ = store.db.Exec(`INSERT INTO devices(user_id,id,name,client_type,last_seen) VALUES(?,?,?,?,?)
			ON CONFLICT(user_id,id) DO UPDATE SET name=COALESCE(NULLIF(excluded.name,''),name), last_seen=excluded.last_seen`,
			uid, dev, q.Get("device_name"), q.Get("client_type"), nowUnix())
	}
	limit := atoi(q.Get("limit"))
	if limit <= 0 {
		limit = 400
	}
	if limit > 2000 {
		limit = 2000
	}
	offset := atoi(q.Get("offset"))
	if offset < 0 {
		offset = 0
	}
	rows, err := store.db.Query(`SELECT e.id,e.feed_id,COALESCE(f.title,''),e.guid,COALESCE(e.title,''),COALESCE(e.kind,''),
		e.published_at,e.duration_sec,e.status,COALESCE(e.file_path,''),COALESCE(e.segments,0),COALESCE(e.segments_done,0),
		COALESCE(p.position_sec,0),COALESCE(p.completed,0),COALESCE(p.segment_idx,0),
		COALESCE(NULLIF(e.cover_url,''), COALESCE(NULLIF(f.cover_url,''),'')),
		EXISTS(SELECT 1 FROM favorites fv WHERE fv.episode_id=e.id AND fv.user_id=?),
		COALESCE(e.added_at,0), COALESCE(e.updated_at,0)
		FROM episodes e JOIN feeds f ON f.id=e.feed_id
		LEFT JOIN progress p ON p.episode_id=e.id AND p.user_id=? AND p.device_id=?
		WHERE f.user_id=? AND (? = 0 OR e.feed_id = ?)
		ORDER BY (e.published_at IS NULL OR e.published_at = 0), e.published_at DESC, e.added_at DESC, e.id DESC
		LIMIT ? OFFSET ?`, uid, uid, dev, uid, feedFilter, feedFilter, limit, offset)
	if err != nil {
		fatal(err)
	}
	defer rows.Close()
	var out []EpisodeJSON
	for rows.Next() {
		var e EpisodeJSON
		var ft, file, cover sql.NullString
		var kindv string
		if err := rows.Scan(&e.ID, &e.FeedID, &ft, &e.Guid, &e.Title, &kindv,
			&e.PublishedAt, &e.DurationSec, &e.Status, &file, &e.Segments, &e.SegmentsDone,
			&e.Position, &e.Completed, &e.SegmentIdx,
			&cover, &e.Favorite, &e.AddedAt, &e.FeedUpdatedAt); err != nil {
			writeJSON(w, map[string]any{"error": err.Error()})
			return
		}
		e.FeedTitle, e.File, e.CoverURL = ft.String, file.String, cover.String
		e.Kind = kindv
		out = append(out, e)
	}
	writeJSON(w, map[string]any{"device_id": dev, "episodes": out, "limit": limit, "offset": offset, "has_more": len(out) == limit})
}

func bookDoc(feedID, uid int64) map[string]any {
	var f FeedJSON
	err := store.db.QueryRow(`SELECT id,url,COALESCE(title,''),COALESCE(description,''),COALESCE(cover_url,''),last_fetched FROM feeds WHERE id=? AND user_id=?`, feedID, uid).
		Scan(&f.ID, &f.URL, &f.Title, &f.Description, &f.CoverURL, &f.LastFetched)
	if err != nil {
		return nil
	}
	rows, err := store.db.Query(`SELECT id,guid,title,published_at,duration_sec,description,status,file_path
		FROM episodes WHERE feed_id=? ORDER BY published_at`, feedID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var eps []map[string]any
	for rows.Next() {
		var id int64
		var guid, title, desc, status, file sql.NullString
		var pub, dur int64
		if err := rows.Scan(&id, &guid, &title, &pub, &dur, &desc, &status, &file); err != nil {
			continue
		}
		eps = append(eps, map[string]any{"id": id, "guid": guid.String, "title": title.String, "published_at": pub,
			"duration_sec": dur, "description": desc.String, "status": status.String, "file": file.String})
	}
	return map[string]any{"title": f.Title, "source": f.URL, "cover_url": f.CoverURL,
		"generated_at": time.Now().UTC().Format(time.RFC3339), "episodes": eps}
}

func handleExport(w http.ResponseWriter, r *http.Request) {
	id := int64(atoi(r.PathValue("feed_id")))
	doc := bookDoc(id, userID(r))
	if doc == nil {
		writeJSON(w, map[string]any{"error": "feed not found"})
		return
	}
	writeJSON(w, doc)
}

// handlePack builds the audiobook zip for one feed. Episodes that were only
// segmented (video audio cuts, TTS chunks) are joined into their finished file
// first, so the archive really contains audio for every downloaded episode.
func handlePack(w http.ResponseWriter, r *http.Request) {
	id := int64(atoi(r.PathValue("feed_id")))
	doc := bookDoc(id, userID(r))
	if doc == nil {
		writeJSON(w, map[string]any{"error": "feed not found"})
		return
	}
	type epRow struct {
		id     int64
		title  string
		status string
		file   string
		segs   int
	}
	rows, err := store.db.Query(`SELECT id, COALESCE(title,''), COALESCE(status,''), COALESCE(file_path,''), COALESCE(segments,0)
		FROM episodes WHERE feed_id=? ORDER BY published_at`, id)
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	var eps []epRow
	for rows.Next() {
		var e epRow
		if err := rows.Scan(&e.id, &e.title, &e.status, &e.file, &e.segs); err != nil {
			continue
		}
		eps = append(eps, e)
	}
	rows.Close()

	type packed struct{ path, name string }
	var files []packed
	nameByID := map[int64]string{}
	for _, e := range eps {
		if e.file == "" && e.segs == 0 {
			continue
		}
		p, err := ensureDownloadFile(r.Context(), e.id)
		if err != nil {
			continue
		}
		ext := filepath.Ext(p)
		name := fmt.Sprintf("%02d - %s%s", len(files)+1, safeName(e.title), ext)
		files = append(files, packed{p, name})
		nameByID[e.id] = name
	}
	if len(files) == 0 {
		writeJSON(w, map[string]any{"error": "no audio for this feed yet - download some episodes first"})
		return
	}
	if list, ok := doc["episodes"].([]map[string]any); ok {
		for _, e := range list {
			if n, ok := nameByID[e["id"].(int64)]; ok {
				e["file"] = n
			} else {
				e["file"] = ""
			}
		}
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="book-`+itoa(id)+`.zip"`)
	zw := zip.NewWriter(w)
	defer zw.Close()
	j, _ := json.MarshalIndent(doc, "", "  ")
	if f, err := zw.Create("book.json"); err == nil {
		_, _ = f.Write(j)
	}
	for _, f := range files {
		hf, err := os.Open(f.path)
		if err != nil {
			continue
		}
		w2, err := zw.Create(f.name)
		if err != nil {
			hf.Close()
			continue
		}
		_, _ = io.Copy(w2, hf)
		hf.Close()
	}
}

func handleMedia(w http.ResponseWriter, r *http.Request) {
	id := atoi(r.PathValue("episode_id"))
	var file string
	if err := store.db.QueryRow(`SELECT COALESCE(e.file_path,'') FROM episodes e JOIN feeds f ON f.id=e.feed_id WHERE e.id=? AND f.user_id=?`, id, userID(r)).Scan(&file); err != nil || file == "" {
		writeJSONStatus(w, http.StatusNotFound, map[string]any{"error": "not downloaded"})
		return
	}
	if !serveFileRange(w, r, file) {
		writeJSON(w, map[string]any{"error": "file missing"})
	}
}

// handleSegment serves one rendered segment. Segments exist as soon as they are
// generated, so the player can start on segment 1 while the rest of the episode
// is still being spoken.
func handleSegment(w http.ResponseWriter, r *http.Request) {
	id := atoi(r.PathValue("episode_id"))
	n := atoi(r.PathValue("seg"))
	if !episodeOwned(userID(r), int64(id)) {
		writeJSONStatus(w, http.StatusNotFound, map[string]any{"error": "not found"})
		return
	}
	if !serveFileRange(w, r, segmentFileFor(int64(id), n)) {
		writeJSON(w, map[string]any{"error": "segment not ready", "segment": n})
	}
}

// handleSegments lists which segments of a text episode are already on disk, so the
// player can start on the first rendered one instead of waiting for the whole file.
func handleSegments(w http.ResponseWriter, r *http.Request) {
	id := atoi(r.PathValue("id"))
	var total int
	if err := store.db.QueryRow(`SELECT e.segments FROM episodes e JOIN feeds f ON f.id=e.feed_id WHERE e.id=? AND f.user_id=?`, id, userID(r)).Scan(&total); err != nil {
		writeJSONStatus(w, http.StatusNotFound, map[string]any{"error": "episode not found"})
		return
	}
	dir := segmentDirFor(int64(id))
	ready := []int{}
	for n := 1; n <= total; n++ {
		if st, err := os.Stat(filepath.Join(dir, strconv.Itoa(n)+segmentExt(int64(id)))); err == nil && st.Size() > 0 {
			ready = append(ready, n)
		}
	}
	writeJSON(w, map[string]any{"segments": total, "ready": ready})
}

func handleEvents(w http.ResponseWriter, r *http.Request) {
	rows, err := store.db.Query(`SELECT id,entity,entity_id,action,created_at FROM events WHERE user_id IN (0,?) ORDER BY id DESC LIMIT 100`, userID(r))
	if err != nil {
		fatal(err)
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var id int64
		var en, eid, act string
		var ts int64
		if err := rows.Scan(&id, &en, &eid, &act, &ts); err != nil {
			fatal(err)
		}
		out = append(out, map[string]any{"id": id, "entity": en, "entity_id": eid, "action": act, "created_at": ts})
	}
	writeJSON(w, out)
}

// backfillCovers gives old video/text rows a thumbnail: the first image in the
// item body, which is how RSSHub (bilibili etc.) ships its cover.
func backfillCovers() {
	rows, err := store.db.Query(`SELECT id, COALESCE(body,''), COALESCE(description,'') FROM episodes
		WHERE COALESCE(cover_url,'')='' AND (COALESCE(body,'')<>'' OR COALESCE(description,'')<>'')`)
	if err != nil {
		return
	}
	type row struct {
		id       int64
		coverURL string
	}
	var todo []row
	for rows.Next() {
		var id int64
		var body, desc string
		if rows.Scan(&id, &body, &desc) != nil {
			continue
		}
		if u := coverFromText(body + " " + desc); u != "" {
			todo = append(todo, row{id, u})
		}
	}
	rows.Close()
	for _, r := range todo {
		_, _ = store.db.Exec(`UPDATE episodes SET cover_url=? WHERE id=? AND COALESCE(cover_url,'')=''`, r.coverURL, r.id)
	}
}
