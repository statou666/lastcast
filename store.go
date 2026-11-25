package main

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

type Store struct{ db *sql.DB }

func openStore(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, err
	}
	const schema = `
CREATE TABLE IF NOT EXISTS users(
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  username TEXT NOT NULL UNIQUE COLLATE NOCASE,
  pass_hash TEXT NOT NULL,
  salt TEXT NOT NULL,
  role TEXT NOT NULL DEFAULT 'user',
  api_token TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS sessions(
  token TEXT PRIMARY KEY,
  user_id INTEGER NOT NULL,
  created_at INTEGER NOT NULL,
  expires_at INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS feeds(
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL DEFAULT 0,
  url TEXT NOT NULL,
  kind TEXT NOT NULL DEFAULT 'auto',
  title TEXT, description TEXT, cover_url TEXT,
  etag TEXT, last_fetched INTEGER,
  UNIQUE(user_id, url)
);
CREATE TABLE IF NOT EXISTS episodes(
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  feed_id INTEGER NOT NULL REFERENCES feeds(id),
  guid TEXT NOT NULL, title TEXT, description TEXT, body TEXT NOT NULL DEFAULT '',
  enclosure_url TEXT, enclosure_type TEXT, audio_url TEXT, video_url TEXT,
  link TEXT NOT NULL DEFAULT '', full_text TEXT NOT NULL DEFAULT '', images TEXT NOT NULL DEFAULT '',
  kind TEXT NOT NULL DEFAULT 'audio',
  published_at INTEGER, duration_sec INTEGER,
  file_path TEXT, status TEXT NOT NULL DEFAULT 'new',
  UNIQUE(feed_id, guid)
);
CREATE TABLE IF NOT EXISTS devices(
  user_id INTEGER NOT NULL DEFAULT 0,
  id TEXT NOT NULL, name TEXT, client_type TEXT, last_seen INTEGER,
  PRIMARY KEY(user_id, id)
);
CREATE TABLE IF NOT EXISTS progress(
  user_id INTEGER NOT NULL DEFAULT 0,
  episode_id INTEGER NOT NULL,
  device_id TEXT NOT NULL,
  position_sec INTEGER NOT NULL DEFAULT 0,
  completed INTEGER NOT NULL DEFAULT 0,
  updated_at INTEGER NOT NULL,
  PRIMARY KEY(user_id, episode_id, device_id)
);
CREATE TABLE IF NOT EXISTS events(
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL DEFAULT 0,
  entity TEXT NOT NULL, entity_id TEXT NOT NULL,
  action TEXT NOT NULL, created_at INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS settings(
  key TEXT PRIMARY KEY, value TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS downloads(
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL DEFAULT 0,
  episode_id INTEGER, feed_id INTEGER, guid TEXT,
  title TEXT, feed_title TEXT, kind TEXT,
  path TEXT NOT NULL UNIQUE, ext TEXT,
  bytes INTEGER NOT NULL DEFAULT 0, duration_sec INTEGER NOT NULL DEFAULT 0,
  created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS downloads_episode ON downloads(episode_id);
CREATE TABLE IF NOT EXISTS favorites(
  user_id INTEGER NOT NULL DEFAULT 0,
  episode_id INTEGER NOT NULL,
  created_at INTEGER NOT NULL,
  PRIMARY KEY(user_id, episode_id)
);
CREATE INDEX IF NOT EXISTS idx_episodes_feed ON episodes(feed_id);
CREATE INDEX IF NOT EXISTS idx_episodes_feed_pub ON episodes(feed_id, published_at DESC);
CREATE INDEX IF NOT EXISTS idx_episodes_updated ON episodes(updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_progress_user_ep_dev ON progress(user_id, episode_id, device_id);
CREATE INDEX IF NOT EXISTS idx_favorites_user_ep ON favorites(user_id, episode_id);
CREATE INDEX IF NOT EXISTS idx_queue_user_dev ON queue(user_id, device_id, position);
CREATE INDEX IF NOT EXISTS idx_downloads_user ON downloads(user_id);
CREATE TABLE IF NOT EXISTS dl_progress(
  episode_id INTEGER PRIMARY KEY,
  user_id INTEGER NOT NULL DEFAULT 0,
  bytes INTEGER NOT NULL DEFAULT 0,
  total INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'queued',
  updated_at INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS history(
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL DEFAULT 0,
  episode_id INTEGER NOT NULL,
  played_at INTEGER NOT NULL,
  UNIQUE(user_id, episode_id)
);
CREATE INDEX IF NOT EXISTS idx_history_user_played ON history(user_id, played_at DESC);
CREATE INDEX IF NOT EXISTS idx_feeds_user ON feeds(user_id);`
	if _, err := db.Exec(schema); err != nil {
		return nil, err
	}
	if err := migrate(db); err != nil {
		return nil, err
	}
	return &Store{db: db}, err
}

// logEvent records an event; pass a user id for user actions (defaults to system).
func (s *Store) logEvent(entity, entityID, action string, uid ...int64) {
	var user int64
	if len(uid) > 0 {
		user = uid[0]
	}
	_, _ = s.db.Exec(`INSERT INTO events(user_id, entity, entity_id, action, created_at) VALUES(?,?,?,?,?)`,
		user, entity, entityID, action, nowUnix())
}

func tableHasColumn(db *sql.DB, table, col string) bool {
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info(?) WHERE name=?`, table, col).Scan(&n); err != nil {
		return false
	}
	return n > 0
}

// migrate adds columns introduced after the first release. SQLite cannot change a
// primary key or a unique constraint with ALTER TABLE, so the per-user tables are
// rebuilt once (existing rows are kept and assigned to user 0, i.e. "unclaimed").
func migrate(db *sql.DB) error {
	adds := []struct{ table, col, def string }{
		{"feeds", "kind", "TEXT NOT NULL DEFAULT 'auto'"},
		{"feeds", "detected", "TEXT NOT NULL DEFAULT ''"},
		{"episodes", "audio_url", "TEXT NOT NULL DEFAULT ''"},
		{"episodes", "video_url", "TEXT NOT NULL DEFAULT ''"},
		{"episodes", "kind", "TEXT NOT NULL DEFAULT 'audio'"},
		{"episodes", "body", "TEXT NOT NULL DEFAULT ''"},
		{"episodes", "segments", "INTEGER NOT NULL DEFAULT 0"},
		{"episodes", "segments_done", "INTEGER NOT NULL DEFAULT 0"},
		{"episodes", "cover_url", "TEXT NOT NULL DEFAULT ''"},
		{"episodes", "author", "TEXT NOT NULL DEFAULT ''"},
		{"feeds", "author", "TEXT NOT NULL DEFAULT ''"},
		{"feeds", "category", "TEXT NOT NULL DEFAULT ''"},
		{"feeds", "auto_download", "INTEGER NOT NULL DEFAULT 0"},
		{"feeds", "fetch_fulltext", "INTEGER NOT NULL DEFAULT 0"},
		{"episodes", "link", "TEXT NOT NULL DEFAULT ''"},
		{"episodes", "full_text", "TEXT NOT NULL DEFAULT ''"},
		{"episodes", "images", "TEXT NOT NULL DEFAULT ''"},
		{"episodes", "added_at", "INTEGER NOT NULL DEFAULT 0"},
		{"episodes", "updated_at", "INTEGER NOT NULL DEFAULT 0"},
		{"progress", "segment_idx", "INTEGER NOT NULL DEFAULT 0"},
		{"queue", "position", "INTEGER NOT NULL DEFAULT 0"},
		{"users", "api_token", "TEXT NOT NULL DEFAULT ''"},
		{"downloads", "user_id", "INTEGER NOT NULL DEFAULT 0"},
		{"events", "user_id", "INTEGER NOT NULL DEFAULT 0"},
	}
	for _, a := range adds {
		var exists int
		if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, a.table).Scan(&exists); err != nil {
			return err
		}
		if exists == 0 {
			continue
		}
		var n int
		if err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info(?) WHERE name=?`, a.table, a.col).Scan(&n); err != nil {
			return err
		}
		if n == 0 {
			if _, err := db.Exec(fmt.Sprintf(`ALTER TABLE %s ADD COLUMN %s %s`, a.table, a.col, a.def)); err != nil {
				return err
			}
		}
	}

	stmts := []string{}
	if !tableHasColumn(db, "feeds", "user_id") {
		stmts = append(stmts,
			`ALTER TABLE feeds RENAME TO feeds_old`,
			`CREATE TABLE feeds(
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				user_id INTEGER NOT NULL DEFAULT 0,
				url TEXT NOT NULL,
				kind TEXT NOT NULL DEFAULT 'auto',
				title TEXT, description TEXT, cover_url TEXT,
				etag TEXT, last_fetched INTEGER,
				detected TEXT NOT NULL DEFAULT '', author TEXT NOT NULL DEFAULT '',
				category TEXT NOT NULL DEFAULT '', auto_download INTEGER NOT NULL DEFAULT 0,
				UNIQUE(user_id, url))`,
			`INSERT INTO feeds(id,user_id,url,kind,title,description,cover_url,etag,last_fetched,detected,author,category,auto_download)
			 SELECT id,0,url,kind,title,description,cover_url,etag,last_fetched,
				COALESCE(detected,''),COALESCE(author,''),COALESCE(category,''),COALESCE(auto_download,0) FROM feeds_old`,
			`DROP TABLE feeds_old`)
	}
	if !tableHasColumn(db, "favorites", "user_id") {
		stmts = append(stmts,
			`ALTER TABLE favorites RENAME TO favorites_old`,
			`CREATE TABLE favorites(user_id INTEGER NOT NULL DEFAULT 0, episode_id INTEGER NOT NULL, created_at INTEGER NOT NULL, PRIMARY KEY(user_id,episode_id))`,
			`INSERT INTO favorites(user_id,episode_id,created_at) SELECT 0,episode_id,created_at FROM favorites_old`,
			`DROP TABLE favorites_old`)
	}
	if !tableHasColumn(db, "progress", "user_id") {
		stmts = append(stmts,
			`ALTER TABLE progress RENAME TO progress_old`,
			`CREATE TABLE progress(user_id INTEGER NOT NULL DEFAULT 0, episode_id INTEGER NOT NULL, device_id TEXT NOT NULL,
				position_sec INTEGER NOT NULL DEFAULT 0, completed INTEGER NOT NULL DEFAULT 0, segment_idx INTEGER NOT NULL DEFAULT 0,
				updated_at INTEGER NOT NULL, PRIMARY KEY(user_id, episode_id, device_id))`,
			`INSERT INTO progress(user_id,episode_id,device_id,position_sec,completed,updated_at,segment_idx)
			 SELECT 0,episode_id,device_id,position_sec,completed,updated_at,COALESCE(segment_idx,0) FROM progress_old`,
			`DROP TABLE progress_old`)
	}
	if tableHasColumn(db, "devices", "id") && !tableHasColumn(db, "devices", "user_id") {
		stmts = append(stmts,
			`ALTER TABLE devices RENAME TO devices_old`,
			`CREATE TABLE devices(user_id INTEGER NOT NULL DEFAULT 0, id TEXT NOT NULL, name TEXT, client_type TEXT, last_seen INTEGER, PRIMARY KEY(user_id,id))`,
			`INSERT INTO devices(user_id,id,name,client_type,last_seen) SELECT 0,id,name,client_type,last_seen FROM devices_old`,
			`DROP TABLE devices_old`)
	}
	if tableHasColumn(db, "queue", "id") && !tableHasColumn(db, "queue", "user_id") {
		stmts = append(stmts,
			`ALTER TABLE queue RENAME TO queue_old`,
			`CREATE TABLE queue(
				id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER NOT NULL DEFAULT 0,
				episode_id INTEGER NOT NULL, device_id TEXT NOT NULL,
				position INTEGER NOT NULL DEFAULT 0, added_at INTEGER NOT NULL,
				UNIQUE(user_id, episode_id, device_id))`,
			`INSERT INTO queue(user_id,episode_id,device_id,position,added_at)
			 SELECT 0,episode_id,device_id,COALESCE(position,0),added_at FROM queue_old`,
			`DROP TABLE queue_old`)
	}
	for _, q := range stmts {
		if _, err := db.Exec(q); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}

	// backfill the timeline columns for rows created before they existed
	_, _ = db.Exec(`UPDATE episodes SET
		added_at=COALESCE(NULLIF(added_at,0), published_at, strftime('%s','now')),
		updated_at=COALESCE(NULLIF(updated_at,0), published_at, strftime('%s','now'))`)
	return nil
}
