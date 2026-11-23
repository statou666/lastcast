package main

import (
	"encoding/json"
	"net/http"
	"os"
)

// handleBatch applies one management action to many rows at once (multi-select).
// All ids are checked against the signed-in user before touching anything.
func handleBatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONStatus(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	var body struct {
		Op       string  `json:"op"`
		IDs      []int64 `json:"ids"`
		DeviceID string  `json:"device_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONStatus(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}
	uid := userID(r)
	if len(body.IDs) == 0 {
		writeJSON(w, map[string]any{"ok": true, "count": 0})
		return
	}
	n := 0
	switch body.Op {
	case "download":
		for _, id := range body.IDs {
			if episodeOwned(uid, id) && needsDownload(id) {
				dl.Enqueue(id)
				n++
			}
		}
	case "queue":
		device := body.DeviceID
		if device == "" {
			device = "web"
		}
		max, _ := queuePositions(uid, device)
		for i, id := range body.IDs {
			if !episodeOwned(uid, id) {
				continue
			}
			if _, err := store.db.Exec(`INSERT OR IGNORE INTO queue(user_id,episode_id,device_id,position,added_at) VALUES(?,?,?,?,?)`,
				uid, id, device, max+int64(i)+1, nowUnix()); err == nil {
				n++
			}
		}
	case "favorite":
		auto := store.getSetting("favorite_auto_download", "0") == "1"
		for _, id := range body.IDs {
			if !episodeOwned(uid, id) {
				continue
			}
			if _, err := store.db.Exec(`INSERT OR IGNORE INTO favorites(user_id,episode_id,created_at) VALUES(?,?,?)`,
				uid, id, nowUnix()); err == nil {
				n++
				if auto && needsDownload(id) {
					dl.Enqueue(id)
				}
			}
		}
	case "unfavorite":
		for _, id := range body.IDs {
			res, err := store.db.Exec(`DELETE FROM favorites WHERE user_id=? AND episode_id=?`, uid, id)
			if err == nil {
				if k, _ := res.RowsAffected(); k > 0 {
					n++
				}
			}
		}
	case "done":
		for _, id := range body.IDs {
			if !episodeOwned(uid, id) {
				continue
			}
			_, _ = store.db.Exec(`INSERT INTO progress(user_id,episode_id,device_id,position_sec,completed,segment_idx,updated_at)
				VALUES(?,?,?,0,1,0,?)
				ON CONFLICT(user_id,episode_id,device_id) DO UPDATE SET completed=1, updated_at=excluded.updated_at`,
				uid, id, body.DeviceID, nowUnix())
			n++
		}
	case "delfeed":
		for _, id := range body.IDs {
			if _, ok := deleteFeedForUser(uid, id); ok {
				n++
			}
		}
	case "delqueue":
		for _, id := range body.IDs {
			res, err := store.db.Exec(`DELETE FROM queue WHERE id=? AND user_id=?`, id, uid)
			if err == nil {
				if k, _ := res.RowsAffected(); k > 0 {
					n++
				}
			}
		}
	case "deldl":
		for _, id := range body.IDs {
			var path string
			if store.db.QueryRow(`SELECT path FROM downloads WHERE id=? AND user_id=?`, id, uid).Scan(&path) != nil {
				continue
			}
			_ = os.Remove(path)
			_, _ = store.db.Exec(`DELETE FROM downloads WHERE id=? AND user_id=?`, id, uid)
			_, _ = store.db.Exec(`UPDATE episodes SET file_path='', status=CASE WHEN COALESCE(segments,0)>0 THEN status ELSE 'new' END WHERE file_path=?`, path)
			n++
		}
		store.logEvent("download", "batch", "deleted", uid)
	default:
		writeJSONStatus(w, http.StatusBadRequest, map[string]any{"error": "unknown op"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "count": n})
}
