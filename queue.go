package main

import (
	"encoding/json"
	"math/rand"
	"net/http"
)

// The playlist lives on the server so it syncs across devices and behaves like a
// music player queue: ordered positions, "play next", move up/down, shuffle.
func (s *Store) ensureQueue() error {
	if _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS queue(
		id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER NOT NULL DEFAULT 0,
		episode_id INTEGER NOT NULL, device_id TEXT NOT NULL,
		position INTEGER NOT NULL DEFAULT 0, added_at INTEGER NOT NULL,
		UNIQUE(user_id, episode_id, device_id))`); err != nil {
		return err
	}
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('queue') WHERE name='position'`).Scan(&n); err == nil && n == 0 {
		_, _ = s.db.Exec(`ALTER TABLE queue ADD COLUMN position INTEGER NOT NULL DEFAULT 0`)
	}
	_, err := s.db.Exec(`UPDATE queue SET position=id WHERE position IS NULL OR position=0`)
	return err
}

func queuePositions(uid int64, device string) (max, min int64) {
	_ = store.db.QueryRow(`SELECT COALESCE(MAX(position),0), COALESCE(MIN(position),0) FROM queue WHERE user_id=? AND device_id=?`, uid, device).Scan(&max, &min)
	return
}

func handleQueue(w http.ResponseWriter, r *http.Request) {
	device := r.URL.Query().Get("device_id")

	if r.Method == http.MethodPost {
		var body struct {
			EpisodeID int64  `json:"episode_id"`
			DeviceID  string `json:"device_id"`
			Next      bool   `json:"next"` // music-player "play next"
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, map[string]any{"error": "invalid json"})
			return
		}
		if body.DeviceID == "" {
			body.DeviceID = device
		}
		if body.EpisodeID == 0 || body.DeviceID == "" {
			writeJSON(w, map[string]any{"error": "episode_id and device_id required"})
			return
		}
		uid := userID(r)
		if !episodeOwned(uid, body.EpisodeID) {
			writeJSON(w, map[string]any{"error": "episode not found"})
			return
		}
		max, min := queuePositions(uid, body.DeviceID)
		pos := max + 1
		if body.Next {
			pos = min - 1
		}
		if _, err := store.db.Exec(`INSERT OR IGNORE INTO queue(user_id, episode_id, device_id, position, added_at) VALUES(?,?,?,?,?)`,
			uid, body.EpisodeID, body.DeviceID, pos, nowUnix()); err != nil {
			writeJSON(w, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, map[string]any{"ok": true, "position": pos})
		return
	}

	if device == "" {
		writeJSON(w, map[string]any{"error": "device_id required"})
		return
	}
	switch r.Method {
	case http.MethodGet:
		uid := userID(r)
		rows, err := store.db.Query(`SELECT q.id, e.id, COALESCE(e.title,''), COALESCE(e.duration_sec,0), COALESCE(e.status,''),
			COALESCE(e.file_path,''), COALESCE(e.enclosure_url,''), COALESCE(f.title,''),
			COALESCE(NULLIF(e.cover_url,''), COALESCE(NULLIF(f.cover_url,''),'')),
			COALESCE(p.position_sec,0), COALESCE(p.completed,0), COALESCE(e.kind,''), q.position,
			EXISTS(SELECT 1 FROM favorites fv WHERE fv.episode_id=e.id AND fv.user_id=?)
			FROM queue q JOIN episodes e ON e.id = q.episode_id JOIN feeds f ON f.id = e.feed_id
			LEFT JOIN progress p ON p.episode_id = e.id AND p.user_id = ? AND p.rowid = (SELECT p2.rowid FROM progress p2 WHERE p2.episode_id = e.id AND p2.user_id = p.user_id `+progressWinOrder()+` LIMIT 1)
			WHERE q.device_id = ? AND q.user_id = ? ORDER BY q.position, q.id`, uid, uid, device, uid)
		if err != nil {
			writeJSON(w, map[string]any{"error": err.Error()})
			return
		}
		defer rows.Close()
		out := []EpisodeJSON{}
		for rows.Next() {
			var ep EpisodeJSON
			var qid, pos, order int64
			var title, feedTitle, cover, file, enc, kind string
			var completed, fav int
			if err := rows.Scan(&qid, &ep.ID, &title, &ep.DurationSec, &ep.Status, &file, &enc,
				&feedTitle, &cover, &pos, &completed, &kind, &order, &fav); err != nil {
				continue
			}
			ep.QueueID, ep.Title, ep.FeedTitle, ep.CoverURL, ep.File, ep.Position, ep.Completed =
				qid, title, feedTitle, cover, file, pos, completed == 1
			ep.Kind, ep.Order, ep.Favorite = kind, int(order), fav == 1
			out = append(out, ep)
		}
		writeJSON(w, out)
	case http.MethodDelete:
		id := r.PathValue("id")
		if id == "" {
			writeJSON(w, map[string]any{"error": "queue id required"})
			return
		}
		if _, err := store.db.Exec(`DELETE FROM queue WHERE id = ? AND device_id = ? AND user_id = ?`, atoi(id), device, userID(r)); err != nil {
			writeJSON(w, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, map[string]any{"ok": true})
	default:
		writeJSON(w, map[string]any{"error": "method not supported"})
	}
}

func handleQueueClear(w http.ResponseWriter, r *http.Request) {
	device := r.URL.Query().Get("device_id")
	if device == "" {
		writeJSON(w, map[string]any{"error": "device_id required"})
		return
	}
	if _, err := store.db.Exec(`DELETE FROM queue WHERE device_id = ? AND user_id = ?`, device, userID(r)); err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

// handleQueueMove swaps a playlist entry with its neighbour (dir=up|down).
func handleQueueMove(w http.ResponseWriter, r *http.Request) {
	id := atoi(r.PathValue("id"))
	var body struct {
		DeviceID string `json:"device_id"`
		Dir      string `json:"dir"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body.DeviceID == "" || id == 0 || (body.Dir != "up" && body.Dir != "down") {
		writeJSON(w, map[string]any{"error": "id, device_id and dir required"})
		return
	}
	uid := userID(r)
	var pos int64
	if err := store.db.QueryRow(`SELECT position FROM queue WHERE id=? AND device_id=? AND user_id=?`, id, body.DeviceID, uid).Scan(&pos); err != nil {
		writeJSON(w, map[string]any{"error": "not found"})
		return
	}
	var nid, npos int64
	q := `SELECT id, position FROM queue WHERE user_id=? AND device_id=? AND position < ? ORDER BY position DESC LIMIT 1`
	args := []any{uid, body.DeviceID, pos}
	if body.Dir == "down" {
		q = `SELECT id, position FROM queue WHERE user_id=? AND device_id=? AND position > ? ORDER BY position ASC LIMIT 1`
	}
	if err := store.db.QueryRow(q, args...).Scan(&nid, &npos); err != nil {
		writeJSON(w, map[string]any{"ok": true, "moved": false})
		return
	}
	tx, err := store.db.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	_, _ = tx.Exec(`UPDATE queue SET position=? WHERE id=?`, npos, id)
	_, _ = tx.Exec(`UPDATE queue SET position=? WHERE id=?`, pos, nid)
	_ = tx.Commit()
	writeJSON(w, map[string]any{"ok": true, "moved": true})
}

// handleQueueShuffle randomizes the playlist order for one device.
func handleQueueShuffle(w http.ResponseWriter, r *http.Request) {
	device := r.URL.Query().Get("device_id")
	if device == "" {
		writeJSON(w, map[string]any{"error": "device_id required"})
		return
	}
	uid := userID(r)
	rows, err := store.db.Query(`SELECT id FROM queue WHERE device_id=? AND user_id=? ORDER BY position, id`, device, uid)
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
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
	rand.Shuffle(len(ids), func(i, j int) { ids[i], ids[j] = ids[j], ids[i] })
	for i, id := range ids {
		_, _ = store.db.Exec(`UPDATE queue SET position=? WHERE id=? AND user_id=?`, i+1, id, uid)
	}
	writeJSON(w, map[string]any{"ok": true, "shuffled": len(ids)})
}
