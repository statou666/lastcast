package main

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Downloads are independent of subscriptions: deleting a feed keeps its finished
// files and their metadata (title/feed_title are snapshotted into this table), so
// they can still be played, packed or deleted from the download manager.
type DownloadJSON struct {
	ID          int64  `json:"id"`
	EpisodeID   int64  `json:"episode_id"`
	FeedID      int64  `json:"feed_id"`
	Title       string `json:"title"`
	FeedTitle   string `json:"feed_title"`
	Kind        string `json:"kind"`
	Path        string `json:"path"`
	Ext         string `json:"ext"`
	Bytes       int64  `json:"bytes"`
	DurationSec int64  `json:"duration_sec"`
	CreatedAt   int64  `json:"created_at"`
}

var downloadMuxer = map[string]string{
	".mp3": "mp3", ".m4a": "ipod", ".mp4": "ipod", ".aac": "adts",
	".ogg": "ogg", ".opus": "opus", ".webm": "webm", ".flac": "flac", ".wav": "wav",
}

// rawPathFor is the scratch location a job writes to before tagging/naming.
func rawPathFor(id int64, ext string) string {
	return filepath.Join(cacheRoot(), "tmp", fmt.Sprintf("%d%s", id, ext))
}

// albumDir is the per-user, per-subscription folder inside the downloads root.
func albumDir(uid int64, feedTitle string) string {
	user := safeName(usernameOf(uid))
	if user == "" || user == "track" {
		user = "u" + itoa(uid)
	}
	return filepath.Join(downloadRoot(), user, safeName(feedTitle))
}

// registerDownload records (or refreshes) the finished file of an episode from its
// current episodes.file_path. Called after any successful download/render.
func registerDownload(episodeID int64) error {
	var path, title, feedTitle, kind, guid string
	var feedID, dur, uid int64
	err := store.db.QueryRow(`SELECT COALESCE(e.file_path,''), COALESCE(e.title,''), COALESCE(f.title,''),
		COALESCE(e.kind,''), e.guid, e.feed_id, COALESCE(e.duration_sec,0), COALESCE(f.user_id,0)
		FROM episodes e JOIN feeds f ON f.id=e.feed_id WHERE e.id=?`, episodeID).
		Scan(&path, &title, &feedTitle, &kind, &guid, &feedID, &dur, &uid)
	if err != nil {
		return err
	}
	if path == "" {
		return fmt.Errorf("episode %d has no file", episodeID)
	}
	st, err := os.Stat(path)
	if err != nil || st.Size() == 0 {
		return fmt.Errorf("file missing: %s", path)
	}
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
	_, err = store.db.Exec(`INSERT INTO downloads(user_id,episode_id,feed_id,guid,title,feed_title,kind,path,ext,bytes,duration_sec,created_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(path) DO UPDATE SET
			user_id=excluded.user_id, episode_id=excluded.episode_id, feed_id=excluded.feed_id, guid=excluded.guid,
			title=excluded.title, feed_title=excluded.feed_title, kind=excluded.kind,
			ext=excluded.ext, bytes=excluded.bytes, duration_sec=excluded.duration_sec`,
		uid, episodeID, feedID, guid, title, feedTitle, kind, path, ext, st.Size(), dur, nowUnix())
	return err
}

func listDownloads(uid int64) ([]DownloadJSON, error) {
	rows, err := store.db.Query(`SELECT id,COALESCE(episode_id,0),COALESCE(feed_id,0),COALESCE(title,''),
		COALESCE(feed_title,''),COALESCE(kind,''),path,COALESCE(ext,''),bytes,COALESCE(duration_sec,0),created_at
		FROM downloads WHERE user_id=? ORDER BY created_at DESC, id DESC`, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []DownloadJSON{}
	for rows.Next() {
		var d DownloadJSON
		if err := rows.Scan(&d.ID, &d.EpisodeID, &d.FeedID, &d.Title, &d.FeedTitle, &d.Kind,
			&d.Path, &d.Ext, &d.Bytes, &d.DurationSec, &d.CreatedAt); err != nil {
			continue
		}
		if st, err := os.Stat(d.Path); err == nil {
			d.Bytes = st.Size()
		}
		out = append(out, d)
	}
	return out, nil
}

func downloadStats(uid int64) (count int, bytes int64) {
	rows, err := store.db.Query(`SELECT path, bytes FROM downloads WHERE user_id=?`, uid)
	if err != nil {
		return 0, 0
	}
	defer rows.Close()
	for rows.Next() {
		var p string
		var b int64
		if err := rows.Scan(&p, &b); err != nil {
			continue
		}
		count++
		if st, err := os.Stat(p); err == nil {
			b = st.Size()
		}
		bytes += b
	}
	return
}

// handleDownloads: GET /api/downloads lists the download manager.
func handleDownloads(w http.ResponseWriter, r *http.Request) {
	list, err := listDownloads(userID(r))
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, list)
}

// handleDownloadItem: DELETE /api/downloads/{id} removes file + record;
// GET /api/downloads/{id}/file streams it (Range capable).
func handleDownloadItem(w http.ResponseWriter, r *http.Request) {
	id := atoi(r.PathValue("id"))
	if id == 0 {
		writeJSON(w, map[string]any{"error": "bad id"})
		return
	}
	uid := userID(r)
	var path, title string
	if err := store.db.QueryRow(`SELECT path, COALESCE(title,'') FROM downloads WHERE id=? AND user_id=?`, id, uid).Scan(&path, &title); err != nil {
		writeJSONStatus(w, http.StatusNotFound, map[string]any{"error": "not found"})
		return
	}
	if r.Method == http.MethodDelete {
		_ = os.Remove(path)
		_, _ = store.db.Exec(`DELETE FROM downloads WHERE id=?`, id)
		_, _ = store.db.Exec(`UPDATE episodes SET file_path='', status=CASE WHEN COALESCE(segments,0)>0 THEN status ELSE 'new' END WHERE file_path=?`, path)
		store.logEvent("download", itoa(int64(id)), "deleted", uid)
		writeJSON(w, map[string]any{"ok": true, "deleted": id})
		return
	}
	if r.URL.Path != "/api/downloads/"+itoa(int64(id))+"/file" {
		writeJSON(w, map[string]any{"error": "not found"})
		return
	}
	f, err := os.Open(path)
	if err != nil {
		writeJSON(w, map[string]any{"error": "file missing"})
		return
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		writeJSON(w, map[string]any{"error": "file missing"})
		return
	}
	name := filepath.Base(path)
	if title != "" {
		name = safeName(title) + filepath.Ext(path)
	}
	w.Header().Set("Content-Disposition", `inline; filename="`+name+`"`)
	http.ServeContent(w, r, name, st.ModTime(), f)
}

// ensureDownloadFile returns the single finished audio file for an episode,
// materializing it from cache segments (video/ffmpeg cuts or TTS chunks) when the
// episode was never downloaded as one file. This is what makes the audiobook pack
// include video episodes.
func ensureDownloadFile(ctx context.Context, episodeID int64) (string, error) {
	var path, feedTitle, kind string
	var segs, done int
	err := store.db.QueryRow(`SELECT COALESCE(e.file_path,''), COALESCE(f.title,''), COALESCE(e.kind,''),
		COALESCE(e.segments,0), COALESCE(e.segments_done,0) FROM episodes e JOIN feeds f ON f.id=e.feed_id WHERE e.id=?`, episodeID).
		Scan(&path, &feedTitle, &kind, &segs, &done)
	if err == nil && path != "" {
		if st, e := os.Stat(path); e == nil && st.Size() > 0 {
			return path, nil
		}
	}
	if segs == 0 {
		return "", fmt.Errorf("episode %d is not downloaded", episodeID)
	}
	if done < segs {
		return "", fmt.Errorf("episode %d is still rendering (%d/%d segments)", episodeID, done, segs)
	}
	ext := ".mp3"
	if kind == "video" {
		ext = ".m4a"
	}
	raw := rawPathFor(episodeID, ext)
	if err := concatSegments(ctx, segmentDirFor(episodeID), ext, raw); err != nil {
		return "", err
	}
	return finishDownload(ctx, episodeID, raw)
}

// concatSegments joins numeric segment files into one raw file with ffmpeg.
func concatSegments(ctx context.Context, dir, ext, out string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	type seg struct {
		n int
		p string
	}
	var segs []seg
	for _, e := range entries {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ext) {
			continue
		}
		n, err := strconv.Atoi(strings.TrimSuffix(e.Name(), filepath.Ext(e.Name())))
		if err != nil {
			continue
		}
		segs = append(segs, seg{n, filepath.Join(dir, e.Name())})
	}
	if len(segs) == 0 {
		return fmt.Errorf("no %s segments in %s", ext, dir)
	}
	sort.Slice(segs, func(i, j int) bool { return segs[i].n < segs[j].n })
	if _, err := exec.LookPath(ffmpegPath()); err != nil {
		return fmt.Errorf("concat needs ffmpeg: %w", err)
	}
	list := filepath.Join(dir, "concat.txt")
	var b strings.Builder
	for _, s := range segs {
		b.WriteString("file '" + s.p + "'\n")
	}
	if err := os.WriteFile(list, []byte(b.String()), 0o644); err != nil {
		return err
	}
	defer os.Remove(list)
	_ = os.MkdirAll(filepath.Dir(out), 0o755)
	muxer := downloadMuxer[strings.ToLower(ext)]
	if muxer == "" {
		muxer = "ipod"
	}
	tmp := out + ".part"
	attempts := [][]string{
		{"-c", "copy"},
		{"-c:a", "aac", "-b:a", "96k"},
	}
	var lastErr error
	for _, args := range attempts {
		base := []string{"-hide_banner", "-loglevel", "error", "-y", "-f", "concat", "-safe", "0", "-i", list}
		cmd := exec.CommandContext(ctx, ffmpegPath(), append(append(base, args...), "-f", muxer, tmp)...)
		if outb, err := cmd.CombinedOutput(); err != nil {
			lastErr = fmt.Errorf("ffmpeg concat: %v: %s", err, strings.TrimSpace(string(outb)))
			continue
		}
		if st, err := os.Stat(tmp); err == nil && st.Size() > 0 {
			if err := os.Rename(tmp, out); err != nil {
				return err
			}
			return nil
		}
		lastErr = fmt.Errorf("ffmpeg concat produced no file")
	}
	_ = os.Remove(tmp)
	return lastErr
}

// probeFileDuration reads a local file's duration with ffprobe (0 when unknown).
func probeFileDuration(path string) int64 {
	p, err := exec.LookPath(ffprobePath())
	if err != nil {
		return 0
	}
	out, err := exec.Command(p, "-v", "error", "-show_entries", "format=duration",
		"-of", "default=nw=1:nk=1", path).Output()
	if err != nil {
		return 0
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	if err != nil || f <= 0 {
		return 0
	}
	return int64(f + 0.5)
}

func probeAudioCodec(path string) string {
	p, err := exec.LookPath(ffprobePath())
	if err != nil {
		return ""
	}
	out, err := exec.Command(p, "-v", "error", "-select_streams", "a:0",
		"-show_entries", "stream=codec_name", "-of", "default=nw=1:nk=1", path).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// handleCache: GET /api/cache reports cache usage; POST {action:"clear"|"evict"}.
func handleCache(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		if !isAdmin(r) {
			writeJSONStatus(w, http.StatusForbidden, map[string]any{"error": "admin only"})
			return
		}
		var body struct {
			Action string `json:"action"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		freed := int64(0)
		if body.Action == "clear" {
			freed = clearCache()
		} else {
			freed = evictCache()
		}
		_, f, b := cacheInfo()
		writeJSON(w, map[string]any{"ok": true, "action": body.Action, "freed": freed, "files": f, "bytes": b})
		return
	}
	dir, files, bytes := cacheInfo()
	limit := atoi(store.getSetting("cache_limit_mb", env("CACHE_LIMIT_MB", "0")))
	writeJSON(w, map[string]any{"dir": dir, "files": files, "bytes": bytes, "limit_mb": limit})
}

func cacheInfo() (string, int, int64) {
	dir := cacheRoot()
	files, bytes := dirStats(dir)
	return dir, files, bytes
}

// io copy helper that flushes as it goes (used by the stream proxy).
func copyFlushing(w http.ResponseWriter, r io.Reader) (int64, error) {
	fl, _ := w.(http.Flusher)
	buf := make([]byte, 32*1024)
	var n int64
	for {
		k, err := r.Read(buf)
		if k > 0 {
			if _, werr := w.Write(buf[:k]); werr != nil {
				return n, werr
			}
			n += int64(k)
			if fl != nil {
				fl.Flush()
			}
		}
		if err == io.EOF {
			return n, nil
		}
		if err != nil {
			return n, err
		}
	}
}

var fsUnsafe = regexp.MustCompile(`[\\/:*?"<>|\x00-\x1f]`)

// safeName makes a title usable as a directory / audiobook track file name.
func safeName(s string) string {
	s = strings.TrimSpace(fsUnsafe.ReplaceAllString(s, "_"))
	s = strings.TrimSpace(strings.Trim(s, "."))
	if s == "" {
		s = "track"
	}
	r := []rune(s)
	if len(r) > 90 {
		s = strings.TrimSpace(string(r[:90]))
	}
	return s
}

// ---------- download finalization: album folder, file name, format, tags, cover ----------

type dlMeta struct {
	UserID      int64
	ID          int64
	Title       string
	FeedID      int64
	FeedTitle   string
	FeedAuthor  string
	Author      string
	Cover       string
	FeedCover   string
	Guid        string
	PublishedAt int64
	Track       int
	Total       int
}

func downloadMeta(id int64) (dlMeta, error) {
	var m dlMeta
	err := store.db.QueryRow(`SELECT e.id, COALESCE(e.title,''), e.feed_id, COALESCE(f.title,''),
		COALESCE(f.author,''), COALESCE(e.author,''), COALESCE(e.cover_url,''), COALESCE(f.cover_url,''),
		e.guid, COALESCE(e.published_at,0), COALESCE(f.user_id,0)
		FROM episodes e JOIN feeds f ON f.id=e.feed_id WHERE e.id=?`, id).
		Scan(&m.ID, &m.Title, &m.FeedID, &m.FeedTitle, &m.FeedAuthor, &m.Author, &m.Cover, &m.FeedCover, &m.Guid, &m.PublishedAt, &m.UserID)
	if err != nil {
		return m, err
	}
	_ = store.db.QueryRow(`SELECT COUNT(*) FROM episodes WHERE feed_id=?`, m.FeedID).Scan(&m.Total)
	_ = store.db.QueryRow(`SELECT COUNT(*) FROM episodes WHERE feed_id=? AND published_at<?`, m.FeedID, m.PublishedAt).Scan(&m.Track)
	m.Track++
	return m, nil
}

// uniqueDownloadPath keeps the file name equal to the episode title, adding a
// numeric suffix only when another episode already owns that name.
func uniqueDownloadPath(dir, title, ext string, ownPath string) string {
	base := safeName(title)
	path := filepath.Join(dir, base+ext)
	if ownPath == path {
		return path
	}
	if _, err := os.Stat(path); err != nil {
		return path
	}
	for i := 2; i < 100; i++ {
		p := filepath.Join(dir, fmt.Sprintf("%s (%d)%s", base, i, ext))
		if _, err := os.Stat(p); err != nil {
			return p
		}
	}
	return filepath.Join(dir, fmt.Sprintf("%s-%d%s", base, time.Now().Unix(), ext))
}

// ensureCover caches a cover image under the cache root and returns its path.
func ensureCover(ctx context.Context, url string) string {
	url = strings.TrimSpace(url)
	if url == "" {
		return ""
	}
	sum := md5.Sum([]byte(url))
	ext := ".jpg"
	if i := strings.LastIndexAny(url, "."); i > 0 {
		e := strings.ToLower(strings.TrimRight(url[i:], "/"))
		if j := strings.IndexByte(e, '?'); j > 0 {
			e = e[:j]
		}
		switch e {
		case ".png", ".jpg", ".jpeg", ".webp", ".gif":
			if e == ".jpeg" {
				e = ".jpg"
			}
			ext = e
		}
	}
	dst := filepath.Join(cacheRoot(), "covers", fmt.Sprintf("%x%s", sum, ext))
	if st, err := os.Stat(dst); err == nil && st.Size() > 512 {
		return dst
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return ""
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", uaPodcastman)
	req.Header.Set("Referer", refererFor(url))
	resp, err := streamClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	f, err := os.Create(dst + ".part")
	if err != nil {
		return ""
	}
	n, err := io.Copy(f, io.LimitReader(resp.Body, 8<<20))
	f.Close()
	if err != nil || n < 512 {
		_ = os.Remove(dst + ".part")
		return ""
	}
	if err := os.Rename(dst+".part", dst); err != nil {
		return ""
	}
	return dst
}

// writeAlbumCover saves a player-friendly cover.jpg next to the tracks.
func writeAlbumCover(dir, cover string) {
	if cover == "" {
		return
	}
	dst := filepath.Join(dir, "cover.jpg")
	if st, err := os.Stat(dst); err == nil && st.Size() > 512 {
		return
	}
	if strings.EqualFold(filepath.Ext(cover), ".jpg") {
		if b, err := os.ReadFile(cover); err == nil && len(b) > 512 {
			_ = os.WriteFile(dst, b, 0o644)
			return
		}
	}
	if _, err := exec.LookPath(ffmpegPath()); err != nil {
		return
	}
	cmd := exec.Command(ffmpegPath(), "-hide_banner", "-loglevel", "error", "-y",
		"-i", cover, "-frames:v", "1", "-q:v", "2", dst)
	_ = cmd.Run()
}

func audioCodecArgs(target, rawCodec string) []string {
	switch target {
	case "mp3":
		if rawCodec == "mp3" {
			return []string{"-c:a", "copy"}
		}
		return []string{"-c:a", "libmp3lame", "-b:a", "192k", "-ar", "44100"}
	case "m4a":
		if rawCodec == "aac" {
			return []string{"-c:a", "copy"}
		}
		return []string{"-c:a", "aac", "-b:a", "128k"}
	}
	return []string{"-c:a", "copy"}
}

func containerArgs(target string) []string {
	switch target {
	case "mp3":
		return []string{"-f", "mp3", "-id3v2_version", "3", "-write_id3v1", "1"}
	case "m4a":
		return []string{"-f", "ipod"}
	case "ogg":
		return []string{"-f", "ogg"}
	case "flac":
		return []string{"-f", "flac"}
	case "webm":
		return []string{"-f", "webm"}
	case "aac":
		return []string{"-f", "adts"}
	}
	return nil
}

// finishDownload turns the raw job output into the final album track: folder per
// subscription, file name = title, chosen format, embedded cover + metadata.
func finishDownload(ctx context.Context, id int64, raw string) (string, error) {
	m, err := downloadMeta(id)
	if err != nil {
		return "", err
	}
	if st, err := os.Stat(raw); err != nil || st.Size() == 0 {
		return "", fmt.Errorf("raw file missing")
	}
	rawExt := strings.ToLower(filepath.Ext(raw))
	if rawExt == ".m4s" || rawExt == ".mp4" || rawExt == ".m4v" || rawExt == ".mov" {
		rawExt = ".m4a"
	}
	target := store.getSetting("download_format", "auto")
	if target != "mp3" && target != "m4a" && target != "auto" {
		target = "auto"
	}
	if target == "auto" {
		switch rawExt {
		case ".webm", ".opus", ".oga", ".ogg", ".flac", ".m4s", ".mp4", ".m4v", ".mov":
			target = "m4a" // containers that cannot carry album tags/cover
		default:
			target = strings.TrimPrefix(rawExt, ".")
		}
	}
	if target != "mp3" && target != "m4a" && target != "ogg" && target != "flac" {
		target = "m4a"
	}
	ext := "." + target
	dir := albumDir(m.UserID, m.FeedTitle)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	var own string
	_ = store.db.QueryRow(`SELECT COALESCE(file_path,'') FROM episodes WHERE id=?`, id).Scan(&own)
	final := uniqueDownloadPath(dir, m.Title, ext, own)
	cover := ensureCover(ctx, m.Cover)
	if cover == "" {
		cover = ensureCover(ctx, m.FeedCover)
	}
	writeAlbumCover(dir, cover)

	rawCodec := probeAudioCodec(raw)
	artist := m.Author
	if artist == "" {
		artist = m.FeedAuthor
	}
	if artist == "" {
		artist = m.FeedTitle
	}
	albumArtist := m.FeedAuthor
	if albumArtist == "" {
		albumArtist = m.FeedTitle
	}
	metaArgs := tagArgs(m, artist, albumArtist)
	tmp := rawPathFor(id, ext) + ".out.part"
	args := []string{"-hide_banner", "-loglevel", "error", "-y", "-i", raw}
	mapArgs := []string{"-map", "0:a"}
	if cover != "" {
		args = append(args, "-i", cover)
		mapArgs = append(mapArgs, "-map", "1:v", "-c:v", "mjpeg", "-disposition:v", "attached_pic")
	}
	args = append(args, mapArgs...)
	args = append(args, audioCodecArgs(target, rawCodec)...)
	args = append(args, metaArgs...)
	args = append(args, containerArgs(target)...)
	args = append(args, tmp)

	ok := false
	if _, err := exec.LookPath(ffmpegPath()); err == nil {
		cmd := exec.CommandContext(ctx, ffmpegPath(), args...)
		if outb, err := cmd.CombinedOutput(); err == nil {
			if st, e := os.Stat(tmp); e == nil && st.Size() > 0 {
				ok = true
			}
		} else {
			_ = outb
		}
	}
	var finalPath string
	if ok {
		if err := os.Rename(tmp, final); err == nil {
			finalPath = final
		}
	}
	if finalPath == "" {
		// No ffmpeg (or a tag/format failure): fall back to a plain copy so the
		// download is never lost.
		_ = os.Remove(tmp)
		final = uniqueDownloadPath(dir, m.Title, rawExt, own)
		if err := os.Rename(raw, final); err == nil {
			finalPath = final
		} else if b, err := os.ReadFile(raw); err == nil {
			if err := os.WriteFile(final, b, 0o644); err != nil {
				return "", err
			}
			finalPath = final
		} else {
			return "", fmt.Errorf("cannot place download")
		}
		_ = os.Remove(raw)
	} else if raw != finalPath {
		_ = os.Remove(raw)
	}
	dur := probeFileDuration(finalPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	_, _ = store.db.Exec(`UPDATE episodes SET status='downloaded', file_path=?,
		duration_sec=CASE WHEN duration_sec IS NULL OR duration_sec=0 THEN ? ELSE duration_sec END WHERE id=?`,
		finalPath, dur, id)
	if err := registerDownload(id); err != nil {
		return finalPath, err
	}
	return finalPath, nil
}

// tagArgs builds the album metadata block shared by downloads and retags.
func tagArgs(m dlMeta, artist, albumArtist string) []string {
	args := []string{
		"-map_metadata", "-1",
		"-metadata", "title=" + m.Title,
		"-metadata", "album=" + m.FeedTitle,
		"-metadata", "artist=" + artist,
		"-metadata", "album_artist=" + albumArtist,
		"-metadata", "genre=Podcast",
		"-metadata", fmt.Sprintf("track=%d/%d", m.Track, m.Total),
	}
	if m.PublishedAt > 0 {
		args = append(args, "-metadata", "date="+time.Unix(m.PublishedAt, 0).Format("2006-01-02"))
	}
	if strings.HasPrefix(m.Guid, "http") {
		args = append(args, "-metadata", "comment="+m.Guid)
	}
	return args
}

// retagFile re-applies cover art and metadata to an existing track WITHOUT
// re-encoding (audio is copied). Used by the download manager's "retag" action for
// files that were downloaded before album tagging existed.
func retagFile(ctx context.Context, id int64, path string) error {
	if _, err := exec.LookPath(ffmpegPath()); err != nil {
		return err
	}
	m, err := downloadMeta(id)
	if err != nil {
		return err
	}
	ext := strings.ToLower(filepath.Ext(path))
	target := strings.TrimPrefix(ext, ".")
	if target == "m4s" || target == "mp4" {
		target = "m4a"
	}
	cover := ensureCover(ctx, m.Cover)
	if cover == "" {
		cover = ensureCover(ctx, m.FeedCover)
	}
	writeAlbumCover(filepath.Dir(path), cover)
	artist := m.Author
	if artist == "" {
		artist = m.FeedAuthor
	}
	if artist == "" {
		artist = m.FeedTitle
	}
	albumArtist := m.FeedAuthor
	if albumArtist == "" {
		albumArtist = m.FeedTitle
	}
	tmp := rawPathFor(id, ext) + ".retag"
	_ = os.MkdirAll(filepath.Dir(tmp), 0o755)
	args := []string{"-hide_banner", "-loglevel", "error", "-y", "-i", path}
	mapArgs := []string{"-map", "0:a"}
	if cover != "" {
		args = append(args, "-i", cover)
		mapArgs = append(mapArgs, "-map", "1:v", "-c:v", "mjpeg", "-disposition:v", "attached_pic")
	}
	args = append(args, mapArgs...)
	args = append(args, "-c:a", "copy")
	args = append(args, tagArgs(m, artist, albumArtist)...)
	args = append(args, containerArgs(target)...)
	args = append(args, tmp)
	if outb, err := exec.CommandContext(ctx, ffmpegPath(), args...).CombinedOutput(); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("retag: %v: %s", err, strings.TrimSpace(string(outb)))
	}
	if st, err := os.Stat(tmp); err != nil || st.Size() == 0 {
		_ = os.Remove(tmp)
		return fmt.Errorf("retag produced no file")
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	return registerDownload(id)
}

// handleRetagDownloads re-tags every finished download in the background.
func handleRetagDownloads(w http.ResponseWriter, r *http.Request) {
	list, err := listDownloads(userID(r))
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		for _, d := range list {
			if d.EpisodeID > 0 {
				enrichIfBili(ctx, d.EpisodeID)
				_ = retagFile(ctx, d.EpisodeID, d.Path)
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()
	writeJSON(w, map[string]any{"ok": true, "queued": len(list)})
}
