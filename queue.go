package main

import (
	"encoding/json"
	"math/rand"
	"net/http"
)

// The playlist lives on the server, scoped per ACCOUNT (not per device), so every
// device sees and edits the same queue. Positions give the order; "play next"
// inserts at the top (position = min-1), plain add appends at the bottom
// (position = max+1).
func (s *Store) ensureQueue() error {
	// One-time migration: the first releases scoped the queue by device_id, which
	// gave every device its own playlist. Rebuild as per-account, merging each
	// episode across devices (keep the topmost position and the earliest add).
	if tableHasColumn(s.db, "queue", "device_id") {
		tx, err := s.db.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(`CREATE TABLE queue_new(
			id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER NOT NULL,
			episode_id INTEGER NOT NULL, position INTEGER NOT NULL, added_at INTEGER NOT NULL,
			UNIQUE(user_id, episode_id))`); err != nil {
			tx.Rollback()
			return err
		}
		if _, err := tx.Exec(`INSERT INTO queue_new(user_id, episode_id, position, added_at)
			SELECT user_id, episode_id, MIN(position), MIN(added_at) FROM queue
			GROUP BY user_id, episode_id`); err != nil {
			tx.Rollback()
			return err
		}
		if _, err := tx.Exec(`DROP TABLE queue`); err != nil {
			tx.Rollback()
			return err
		}
		if _, err := tx.Exec(`ALTER TABLE queue_new RENAME TO queue`); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		// Renumber each user's queue to a clean 1..n sequence in the merged order,
		// so the list starts with stable positions (no negatives, no duplicates).
		uids, err := s.db.Query(`SELECT DISTINCT user_id FROM queue`)
		if err != nil {
			return err
		}
		var list []int64
		for uids.Next() {
			var u int64
			if uids.Scan(&u) == nil {
				list = append(list, u)
			}
		}
		uids.Close()
		for _, u := range list {
			rows, err := s.db.Query(`SELECT id FROM queue WHERE user_id=? ORDER BY position, id`, u)
			if err != nil {
				return err
			}
			var ids []int64
			for rows.Next() {
				var id int64
				if rows.Scan(&id) == nil {
					ids = append(ids, id)
				}
			}
			rows.Close()
			for i, id := range ids {
				if _, err := s.db.Exec(`UPDATE queue SET position=? WHERE id=?`, i+1, id); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func queuePositions(uid int64) (max, min int64) {
	_ = store.db.QueryRow(`SELECT COALESCE(MAX(position),0), COALESCE(MIN(position),0) FROM queue WHERE user_id=?`, uid).Scan(&max, &min)
	return
}

func handleQueue(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var body struct {
			EpisodeID int64  `json:"episode_id"`
			Next      bool   `json:"next"` // music-player "play next"
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, map[string]any{"error": "invalid json"})
			return
		}
		if body.EpisodeID == 0 {
			writeJSON(w, map[string]any{"error": "episode_id required"})
			return
		}
		uid := userID(r)
		if !episodeOwned(uid, body.EpisodeID) {
			writeJSON(w, map[string]any{"error": "episode not found"})
			return
		}
		max, min := queuePositions(uid)
		pos := max + 1
		if body.Next {
			pos = min - 1
		}
		if _, err := store.db.Exec(`INSERT OR IGNORE INTO queue(user_id, episode_id, position, added_at) VALUES(?,?,?,?)`,
			uid, body.EpisodeID, pos, nowUnix()); err != nil {
			writeJSON(w, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, map[string]any{"ok": true, "position": pos})
		return
	}

	switch r.Method {
	case http.MethodGet:
		uid := userID(r)
		rows, err := store.db.Query(`SELECT q.id, e.id, COALESCE(e.title,''), COALESCE(e.duration_sec,0), COALESCE(e.status,''),
			COALESCE(e.file_path,''), COALESCE(e.enclosure_url,''), COALESCE(f.title,''),
			COALESCE(NULLIF(e.cover_url,''), COALESCE(NULLIF(f.cover_url,''),'')),
			COALESCE(p.position_sec,0), COALESCE(p.completed,0), COALESCE(e.kind,''), q.position,
			COALESCE(p.segment_idx,0),
			EXISTS(SELECT 1 FROM favorites fv WHERE fv.episode_id=e.id AND fv.user_id=?)
			FROM queue q JOIN episodes e ON e.id = q.episode_id JOIN feeds f ON f.id = e.feed_id
			LEFT JOIN progress p ON p.episode_id = e.id AND p.user_id = ? AND p.rowid = (SELECT p2.rowid FROM progress p2 WHERE p2.episode_id = e.id AND p2.user_id = p.user_id `+progressWinOrder()+` LIMIT 1)
			WHERE q.user_id = ? ORDER BY q.position, q.id`, uid, uid, uid)
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
			var segIdx int64
			if err := rows.Scan(&qid, &ep.ID, &title, &ep.DurationSec, &ep.Status, &file, &enc,
				&feedTitle, &cover, &pos, &completed, &kind, &order, &segIdx, &fav); err != nil {
				continue
			}
			ep.QueueID, ep.Title, ep.FeedTitle, ep.CoverURL, ep.File, ep.Position, ep.Completed =
				qid, title, feedTitle, cover, file, pos, completed == 1
			ep.Kind, ep.Order, ep.Favorite = kind, int(order), fav == 1
			ep.SegmentIdx = int(segIdx)
			out = append(out, ep)
		}
		writeJSON(w, out)
	case http.MethodDelete:
		id := r.PathValue("id")
		if id == "" {
			writeJSON(w, map[string]any{"error": "queue id required"})
			return
		}
		if _, err := store.db.Exec(`DELETE FROM queue WHERE id = ? AND user_id = ?`, atoi(id), userID(r)); err != nil {
			writeJSON(w, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, map[string]any{"ok": true})
	default:
		writeJSON(w, map[string]any{"error": "method not supported"})
	}
}

func handleQueueClear(w http.ResponseWriter, r *http.Request) {
	if _, err := store.db.Exec(`DELETE FROM queue WHERE user_id = ?`, userID(r)); err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

// handleQueueMove swaps a playlist entry with its neighbour (dir=up|down).
func handleQueueMove(w http.ResponseWriter, r *http.Request) {
	id := atoi(r.PathValue("id"))
	var body struct {
		Dir string `json:"dir"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if id == 0 || (body.Dir != "up" && body.Dir != "down") {
		writeJSON(w, map[string]any{"error": "id and dir required"})
		return
	}
	uid := userID(r)
	var pos int64
	if err := store.db.QueryRow(`SELECT position FROM queue WHERE id=? AND user_id=?`, id, uid).Scan(&pos); err != nil {
		writeJSON(w, map[string]any{"error": "not found"})
		return
	}
	var nid, npos int64
	q := `SELECT id, position FROM queue WHERE user_id=? AND position < ? ORDER BY position DESC LIMIT 1`
	args := []any{uid, pos}
	if body.Dir == "down" {
		q = `SELECT id, position FROM queue WHERE user_id=? AND position > ? ORDER BY position ASC LIMIT 1`
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

// handleQueueShuffle randomizes the playlist order for the account.
func handleQueueShuffle(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	rows, err := store.db.Query(`SELECT id FROM queue WHERE user_id=? ORDER BY position, id`, uid)
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
