package main

import (
	"encoding/json"
	"net/http"
)

// Favorites work like an RSS reader's starred items: a bookmark that does not
// download audio unless `favorite_auto_download` is on.
func handleFavorites(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		rows, err := store.db.Query(`SELECT e.id,e.feed_id,COALESCE(f.title,''),e.guid,COALESCE(e.title,''),COALESCE(e.kind,''),
			COALESCE(e.body,''),COALESCE(e.video_url,''),COALESCE(e.description,''),COALESCE(e.enclosure_url,''),
			COALESCE(e.published_at,0),COALESCE(e.duration_sec,0),COALESCE(e.status,''),COALESCE(e.file_path,''),
			COALESCE(e.segments,0),COALESCE(e.segments_done,0),COALESCE(f.cover_url,''),COALESCE(e.cover_url,'')
			FROM favorites fv JOIN episodes e ON e.id=fv.episode_id JOIN feeds f ON f.id=e.feed_id
			WHERE fv.user_id=? ORDER BY fv.created_at DESC`, userID(r))
		if err != nil {
			writeJSON(w, map[string]any{"error": err.Error()})
			return
		}
		defer rows.Close()
		out := []EpisodeJSON{}
		for rows.Next() {
			var e EpisodeJSON
			var body, video, desc, file, cover, epCover string
			if err := rows.Scan(&e.ID, &e.FeedID, &e.FeedTitle, &e.Guid, &e.Title, &e.Kind,
				&body, &video, &desc, &e.EnclosureURL, &e.PublishedAt, &e.DurationSec, &e.Status, &file,
				&e.Segments, &e.SegmentsDone, &cover, &epCover); err != nil {
				continue
			}
			e.Body, e.Description, e.VideoURL, e.File = body, desc, video, file
			e.CoverURL = epCover
			if e.CoverURL == "" {
				e.CoverURL = cover
			}
			e.Favorite = true
			out = append(out, e)
		}
		writeJSON(w, out)
	case http.MethodPost:
		var body struct {
			EpisodeID int64 `json:"episode_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.EpisodeID == 0 {
			writeJSON(w, map[string]any{"error": "episode_id required"})
			return
		}
		var exists int
		if err := store.db.QueryRow(`SELECT COUNT(*) FROM episodes WHERE id=?`, body.EpisodeID).Scan(&exists); err != nil || exists == 0 {
			writeJSON(w, map[string]any{"error": "episode not found"})
			return
		}
		uid := userID(r)
		if !episodeOwned(uid, body.EpisodeID) {
			writeJSON(w, map[string]any{"error": "episode not found"})
			return
		}
		if _, err := store.db.Exec(`INSERT OR IGNORE INTO favorites(user_id, episode_id, created_at) VALUES(?,?,?)`, uid, body.EpisodeID, nowUnix()); err != nil {
			writeJSON(w, map[string]any{"error": err.Error()})
			return
		}
		queued := false
		if store.getSetting("favorite_auto_download", "0") == "1" {
			dl.Enqueue(body.EpisodeID)
			queued = true
		}
		writeJSON(w, map[string]any{"ok": true, "favorite": true, "queued": queued})
	case http.MethodDelete:
		id := atoi(r.PathValue("id"))
		if id == 0 {
			writeJSON(w, map[string]any{"error": "episode id required"})
			return
		}
		_, _ = store.db.Exec(`DELETE FROM favorites WHERE episode_id=? AND user_id=?`, id, userID(r))
		writeJSON(w, map[string]any{"ok": true, "favorite": false})
	default:
		writeJSON(w, map[string]any{"error": "method not supported"})
	}
}
