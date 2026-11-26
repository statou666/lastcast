package main

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Two separate on-disk roots:
//
//	cache/<feed slug>/<episode id>/<n>.<ext>   disposable (segments, TTS chunks), evictable
//	downloads/<feed slug>/<episode id>.<ext>   persistent, survives feed deletion
//
// Both are configurable (settings `cache_dir` / `download_dir`, env CACHE_DIR /
// DOWNLOAD_DIR). Relative values resolve against DATA_DIR.
func dirSetting(key, envKey, def string) string {
	d := store.getSetting(key, env(envKey, ""))
	if d == "" {
		d = def
	}
	if !filepath.IsAbs(d) {
		d = filepath.Join(cfg.DataDir, d)
	}
	return filepath.Clean(d)
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

func cacheRoot() string    { return dirSetting("cache_dir", "CACHE_DIR", "cache") }
func downloadRoot() string { return dirSetting("download_dir", "DOWNLOAD_DIR", "downloads") }

func ensureRoots() {
	_ = os.MkdirAll(cacheRoot(), 0o755)
	_ = os.MkdirAll(downloadRoot(), 0o755)
}

// feedSlugOf resolves the feed title of an episode into its directory name.
func feedSlugOf(id int64) string {
	var t string
	if err := store.db.QueryRow(`SELECT COALESCE(f.title,'') FROM episodes e JOIN feeds f ON f.id=e.feed_id WHERE e.id=?`, id).Scan(&t); err != nil || t == "" {
		t = "feed"
	}
	return slug(t)
}

func segmentDirFor(id int64) string {
	// The "p3" suffix invalidates segment caches created before the longer lead-in
	// pad (the old -p2 files carry a 120ms pad that is too short to cover the
	// audio element's cold-start latency, which clips the opening syllable after
	// a pause). Old files are disposable cache and are removed by normal eviction.
	return filepath.Join(cacheRoot(), feedSlugOf(id), strconv.FormatInt(id, 10)+"-p4")
}

func segmentFileFor(id int64, n int) string {
	return filepath.Join(segmentDirFor(id), strconv.Itoa(n)+segmentExt(id))
}

func removeEpisodeCache(id int64) {
	_ = os.RemoveAll(segmentDirFor(id))
}

// dirStats walks a directory tree and reports file count and total bytes.
func dirStats(dir string) (files int, bytes int64) {
	_ = filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		files++
		bytes += info.Size()
		return nil
	})
	return
}

// evictCache enforces cache_limit_mb by deleting the oldest segment files first.
// Episodes whose segment folder becomes empty get their counters reset.
func evictCache() (freed int64) {
	limitMB := atoi(store.getSetting("cache_limit_mb", env("CACHE_LIMIT_MB", "0")))
	if limitMB <= 0 {
		return 0
	}
	limit := int64(limitMB) * 1024 * 1024
	root := cacheRoot()
	type entry struct {
		path string
		size int64
		mt   time.Time
	}
	var files []entry
	var total int64
	_ = filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		files = append(files, entry{p, info.Size(), info.ModTime()})
		total += info.Size()
		return nil
	})
	if total <= limit {
		return 0
	}
	sort.Slice(files, func(i, j int) bool { return files[i].mt.Before(files[j].mt) })
	affected := map[int64]string{}
	for _, x := range files {
		if total <= limit {
			break
		}
		if err := os.Remove(x.path); err == nil {
			total -= x.size
			freed += x.size
		}
		rel, err := filepath.Rel(root, x.path)
		if err != nil {
			continue
		}
		parts := strings.Split(rel, string(filepath.Separator))
		if len(parts) >= 3 {
			if id, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
				affected[id] = filepath.Join(root, parts[0], parts[1])
			}
		}
	}
	for id, dir := range affected {
		if countSegments(dir) == 0 {
			_, _ = store.db.Exec(`UPDATE episodes SET segments=0, segments_done=0 WHERE id=?`, id)
			_ = os.Remove(dir)
		}
	}
	return freed
}

// clearCache deletes every cache file (segments are always re-derivable) and
// resets segment counters for episodes that had no permanent file.
func clearCache() (freed int64) {
	root := cacheRoot()
	entries, err := os.ReadDir(root)
	if err != nil {
		return 0
	}
	for _, e := range entries {
		p := filepath.Join(root, e.Name())
		_, b := dirStats(p)
		freed += b
		_ = os.RemoveAll(p)
	}
	_, _ = store.db.Exec(`UPDATE episodes SET segments=0, segments_done=0`)
	return freed
}

// migrateMediaLayout moves the pre-cache layout (<data>/media/...) to the split
// layout exactly once: segment folders into cache, finished files into downloads.
func migrateMediaLayout() {
	old := filepath.Join(cfg.DataDir, "media")
	if st, err := os.Stat(old); err != nil || !st.IsDir() {
		return
	}
	cache, dl := cacheRoot(), downloadRoot()
	_ = os.MkdirAll(cache, 0o755)
	_ = os.MkdirAll(dl, 0o755)
	slugs, err := os.ReadDir(old)
	if err != nil {
		return
	}
	for _, s := range slugs {
		if !s.IsDir() {
			continue
		}
		slugDir := filepath.Join(old, s.Name())
		items, err := os.ReadDir(slugDir)
		if err != nil {
			continue
		}
		for _, it := range items {
			name := it.Name()
			if it.IsDir() {
				dst := filepath.Join(cache, s.Name(), name)
				_ = os.MkdirAll(filepath.Dir(dst), 0o755)
				_ = os.Rename(filepath.Join(slugDir, name), dst)
				continue
			}
			id, err := strconv.ParseInt(strings.TrimSuffix(name, filepath.Ext(name)), 10, 64)
			if err != nil {
				continue
			}
			dst := filepath.Join(dl, s.Name(), name)
			_ = os.MkdirAll(filepath.Dir(dst), 0o755)
			if err := os.Rename(filepath.Join(slugDir, name), dst); err == nil {
				_, _ = store.db.Exec(`UPDATE episodes SET file_path=? WHERE id=? AND (COALESCE(file_path,'')='' OR file_path LIKE ?)`, dst, id, old+"%")
				_ = registerDownload(id)
			}
		}
		_ = os.Remove(slugDir) // only if empty
	}
	_ = os.Remove(old)
}
