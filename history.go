package main

import (
	"encoding/json"
	"net/http"
)

// historyCap is the per-user cap for the played-episode history.
const historyCap = 300

// recordPlay adds (or refreshes) one played episode for the user. Only real
// playback counts — opening an episode page does not. Replaying an episode just
// moves it to the top (updates played_at). Rows beyond historyCap are dropped.
func recordPlay(uid, episodeID int64) {
	if uid <= 0 || episodeID <= 0 {
		return
	}
	_, err := store.db.Exec(`INSERT INTO history(user_id,episode_id,played_at) VALUES(?,?,?)
		ON CONFLICT(user_id,episode_id) DO UPDATE SET played_at=excluded.played_at`,
		uid, episodeID, nowUnix())
	if err != nil {
		return
	}
	_, _ = store.db.Exec(`DELETE FROM history WHERE user_id=? AND id NOT IN
		(SELECT id FROM history WHERE user_id=? ORDER BY played_at DESC LIMIT ?)`, uid, uid, historyCap)
}

// handleHistory: GET lists the user's played episodes (newest first, at most
// historyCap rows); POST {episode_id} records a play.
func handleHistory(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	if r.Method == http.MethodPost {
		var body struct {
			EpisodeID int64 `json:"episode_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.EpisodeID <= 0 {
			writeJSON(w, map[string]any{"error": "episode_id required"})
			return
		}
		_, err := store.db.Exec(`INSERT INTO history(user_id,episode_id,played_at) VALUES(?,?,?)
			ON CONFLICT(user_id,episode_id) DO UPDATE SET played_at=excluded.played_at`,
			uid, body.EpisodeID, nowUnix())
		if err != nil {
			writeJSON(w, map[string]any{"error": err.Error()})
			return
		}
		_, _ = store.db.Exec(`DELETE FROM history WHERE user_id=? AND id NOT IN
			(SELECT id FROM history WHERE user_id=? ORDER BY played_at DESC LIMIT ?)`, uid, uid, historyCap)
		writeJSON(w, map[string]any{"ok": true})
		return
	}
	limit := atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = historyCap
	}
	if limit > historyCap {
		limit = historyCap
	}
	// The real episode status (not a hardcoded "downloaded") plus the account's
	// progress, so the history view shows the truth: a played-but-not-downloaded
	// episode shows its actual status, and the progress bar reflects the breakpoint.
	rows, err := store.db.Query(`SELECT h.played_at, e.id, COALESCE(e.title,''), COALESCE(f.title,''),
		COALESCE(NULLIF(e.cover_url,''), COALESCE(NULLIF(f.cover_url,''),'')),
		COALESCE(e.status,''), COALESCE(e.duration_sec,0), COALESCE(p.position_sec,0), COALESCE(p.completed,0)
		FROM history h JOIN episodes e ON e.id=h.episode_id JOIN feeds f ON f.id=e.feed_id
		LEFT JOIN progress p ON p.episode_id=e.id AND p.user_id=? AND p.rowid = (SELECT p2.rowid FROM progress p2 WHERE p2.episode_id=e.id AND p2.user_id=p.user_id `+progressWinOrder()+` LIMIT 1)
		WHERE h.user_id=? ORDER BY h.played_at DESC LIMIT ?`, uid, uid, limit)
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var ts, id, dur, pos int64
		var title, feedTitle, cover, status string
		var completed int
		if err := rows.Scan(&ts, &id, &title, &feedTitle, &cover, &status, &dur, &pos, &completed); err != nil {
			continue
		}
		out = append(out, map[string]any{"episode_id": id, "title": title, "feed_title": feedTitle,
			"cover_url": cover, "played_at": ts, "status": status, "duration_sec": dur,
			"position": pos, "completed": completed == 1})
	}
	writeJSON(w, out)
}
