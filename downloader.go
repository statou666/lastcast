package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Downloader struct {
	store   *Store
	dataDir string
	client  *http.Client
	queue   chan int64
}

func NewDownloader(s *Store, dataDir string) *Downloader {
	return &Downloader{store: s, dataDir: dataDir,
		client: &http.Client{Timeout: 0, Transport: newTransport()}, // downloads: no overall timeout, resume covers it
		queue:  make(chan int64, 256)}
}

func (d *Downloader) Enqueue(id int64) {
	d.queue <- id
}

func (d *Downloader) Run(ctx context.Context) {
	n := atoi(store.getSetting("download_concurrency", "2"))
	if n < 1 {
		n = 1
	}
	if n > 8 {
		n = 8
	}
	for i := 0; i < n; i++ {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case id := <-d.queue:
					if err := d.download(ctx, id); err != nil {
						_, _ = d.store.db.Exec(`UPDATE episodes SET status='error' WHERE id=?`, id)
						d.store.logEvent("episode", itoa(id), "download_error: "+err.Error())
					}
					evictCache()
				}
			}
		}()
	}
}

// overLimit enforces storage_limit_mb across the persistent downloads root and the
// disposable cache root (0 = unlimited).
func (d *Downloader) overLimit() bool {
	limit := atoi(store.getSetting("storage_limit_mb", "0"))
	if limit <= 0 {
		return false
	}
	_, dlBytes := dirStats(downloadRoot())
	_, cBytes := dirStats(cacheRoot())
	return dlBytes+cBytes >= int64(limit)*1024*1024
}

func extFor(ctype, url string) string {
	switch strings.ToLower(ctype) {
	case "audio/mpeg", "audio/mp3":
		return ".mp3"
	case "audio/mp4", "audio/x-m4a", "audio/m4a":
		return ".m4a"
	case "audio/ogg":
		return ".ogg"
	case "audio/aac":
		return ".aac"
	case "audio/flac":
		return ".flac"
	}
	if e := strings.ToLower(path.Ext(strings.Split(url, "?")[0])); len(e) == 4 {
		return e
	}
	return ".mp3"
}

func slug(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else if r == ' ' || r == '-' || r == '_' {
			b.WriteRune('_')
		}
	}
	out := b.String()
	if out == "" {
		return "feed"
	}
	if len(out) > 64 {
		out = out[:64]
	}
	return out
}

func (d *Downloader) download(ctx context.Context, id int64) error {
	var feedTitle, url, ctype, guid, kind, desc, existing string
	err := d.store.db.QueryRow(`SELECT e.id, f.title, e.enclosure_url, COALESCE(e.enclosure_type,''), e.guid, e.kind, COALESCE(e.description,''), COALESCE(e.file_path,'')
		FROM episodes e JOIN feeds f ON f.id=e.feed_id WHERE e.id=?`, id).
		Scan(new(int64), &feedTitle, &url, &ctype, &guid, &kind, &desc, &existing)
	if err == sql.ErrNoRows {
		return err
	}
	if err != nil {
		return err
	}
	if existing != "" {
		if st, e := os.Stat(existing); e == nil && st.Size() > 0 {
			return nil // already downloaded
		}
	}
	// text-only items have no media file: render them through the configured TTS endpoint
	if kind == "text" {
		return d.synthesize(ctx, id)
	}
	if d.overLimit() {
		return fmt.Errorf("storage limit reached")
	}
	if kind == "video" {
		// Playback streams the remote media (see /stream/{id}); a download is one
		// finished audio-only file, which is what the audiobook pack needs.
		enrichIfBili(ctx, id)
		page := pageURLFor(url, guid)
		urls, dur, err := resolveAudioCandidates(ctx, url, ctype, guid, desc)
		if err != nil {
			if page != "" && ytdlpAvailable() {
				if raw, yerr := d.ytdlpDownload(ctx, id, page); yerr == nil {
					if _, ferr := finishDownload(ctx, id, raw); ferr != nil {
						return ferr
					}
					return nil
				}
			}
			return err
		}
		if dur > 0 {
			_, _ = d.store.db.Exec(`UPDATE episodes SET duration_sec=? WHERE id=? AND (duration_sec IS NULL OR duration_sec=0)`, dur, id)
		}
		if len(urls) == 0 {
			return fmt.Errorf("no audio source for this item")
		}
		return d.downloadVideo(ctx, id, urls, ctype)
	}
	return d.downloadFile(ctx, id, url, ctype)
}

// rawExt picks the scratch container for a remote source: it follows the source
// container (mp3 stays mp3, m4s/fMP4 becomes m4a) because a "generate"-style URL
// has no extension at all.
func rawExt(srcURL, ctype string) string {
	switch strings.ToLower(filepath.Ext(urlPath(srcURL))) {
	case ".mp3":
		return ".mp3"
	case ".ogg", ".oga":
		return ".ogg"
	case ".flac":
		return ".flac"
	case ".aac":
		return ".aac"
	}
	if strings.Contains(strings.ToLower(ctype), "mpeg") {
		return ".mp3"
	}
	return ".m4a"
}

// downloadVideo fetches the remote media into a scratch file and extracts its
// audio track. Fetching first (chunked Range with retries) is deliberate: some
// CDNs (bilibili's mcdn edge nodes) answer small range probes instantly but stall
// or truncate a long sequential read, which makes ffmpeg give up mid-file.
func (d *Downloader) downloadVideo(ctx context.Context, id int64, urls []string, ctype string) error {
	media := filepath.Join(cacheRoot(), "tmp", itoa(id)+".src")
	if err := os.MkdirAll(filepath.Dir(media), 0o755); err != nil {
		return err
	}
	_ = os.Remove(media)
	var lastErr error
	for i, u := range urls {
		if i >= 4 {
			break
		}
		_ = os.Remove(media)
		if err := d.fetchMedia(ctx, u, media); err != nil {
			lastErr = err
			continue
		}
		if st, e := os.Stat(media); e == nil && st.Size() > 0 {
			lastErr = nil
			break
		}
	}
	if lastErr != nil {
		return lastErr
	}
	ext := rawExt(urls[0], ctype)
	raw := rawPathFor(id, ext)
	if !ffmpegAvailable() {
		_ = os.MkdirAll(filepath.Dir(raw), 0o755)
		if err := os.Rename(media, raw); err != nil {
			return err
		}
		if _, err := finishDownload(ctx, id, raw); err != nil {
			return err
		}
		return nil
	}
	if err := d.extractAudio(ctx, media, raw, ext); err != nil {
		return err
	}
	_ = os.Remove(media)
	if _, err := finishDownload(ctx, id, raw); err != nil {
		return err
	}
	return nil
}

// extractAudio copies the audio track of a local media file into the raw container.
func (d *Downloader) extractAudio(ctx context.Context, media, raw, ext string) error {
	muxer := downloadMuxer[strings.ToLower(ext)]
	if muxer == "" {
		muxer = "ipod"
	}
	if err := os.MkdirAll(filepath.Dir(raw), 0o755); err != nil {
		return err
	}
	tmp := raw + ".part"
	attempts := [][]string{{"-c:a", "copy"}}
	switch ext {
	case ".m4a", ".aac":
		attempts = append(attempts, []string{"-c:a", "aac", "-b:a", "96k"})
	case ".mp3":
		attempts = append(attempts, []string{"-c:a", "libmp3lame", "-b:a", "192k"})
	case ".ogg":
		attempts = append(attempts, []string{"-c:a", "libopus", "-b:a", "96k"})
	}
	var lastErr error
	for _, codec := range attempts {
		args := []string{"-hide_banner", "-loglevel", "error", "-y", "-i", media, "-vn"}
		args = append(args, codec...)
		args = append(args, "-f", muxer, tmp)
		if outb, err := exec.CommandContext(ctx, ffmpegPath(), args...).CombinedOutput(); err != nil {
			lastErr = fmt.Errorf("ffmpeg: %v: %s", err, strings.TrimSpace(string(outb)))
			continue
		}
		if st, err := os.Stat(tmp); err == nil && st.Size() > 0 {
			return os.Rename(tmp, raw)
		}
		lastErr = fmt.Errorf("ffmpeg wrote no audio")
	}
	_ = os.Remove(tmp)
	return lastErr
}

// fetchMedia downloads a remote file into dst using 4 MB Range requests, resuming
// from a previous .part and retrying each chunk. Partial bytes are never lost.
func (d *Downloader) fetchMedia(ctx context.Context, url, dst string) error {
	part := dst + ".part"
	var off int64
	if st, err := os.Stat(part); err == nil {
		off = st.Size()
	}
	f, err := os.OpenFile(part, os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err := f.Seek(off, io.SeekStart); err != nil {
		f.Close()
		return err
	}
	const chunk = 4 << 20
	total := int64(-1)
	for {
		if err := ctx.Err(); err != nil {
			f.Close()
			return err
		}
		n, tot, done, err := d.fetchRange(ctx, url, f, off, chunk)
		if tot > 0 {
			total = tot
		}
		off += n
		if err != nil {
			f.Close()
			return err
		}
		if done || n == 0 || (total > 0 && off >= total) {
			break
		}
	}
	if err := f.Close(); err != nil {
		return err
	}
	st, err := os.Stat(part)
	if err != nil || st.Size() == 0 {
		return fmt.Errorf("empty media download")
	}
	return os.Rename(part, dst)
}

// fetchRange appends one range chunk to f and reports progress and completion.
func (d *Downloader) fetchRange(ctx context.Context, url string, f *os.File, off, size int64) (n, total int64, done bool, err error) {
	for attempt := 0; attempt < 4; attempt++ {
		req, e := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if e != nil {
			return 0, 0, false, e
		}
		req.Header.Set("User-Agent", uaPodcastman)
		req.Header.Set("Referer", refererFor(url))
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", off, off+size-1))
		rctx, cancel := context.WithTimeout(ctx, 90*time.Second)
		resp, e := d.client.Do(req.WithContext(rctx))
		if e != nil {
			cancel()
			time.Sleep(time.Duration(attempt+1) * 400 * time.Millisecond)
			continue
		}
		if resp.StatusCode == http.StatusRequestedRangeNotSatisfiable {
			resp.Body.Close()
			cancel()
			return 0, 0, true, nil
		}
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
			resp.Body.Close()
			cancel()
			time.Sleep(time.Duration(attempt+1) * 400 * time.Millisecond)
			continue
		}
		if cr := resp.Header.Get("Content-Range"); cr != "" {
			if i := strings.LastIndexByte(cr, '/'); i >= 0 {
				if v, e := strconv.ParseInt(strings.TrimSpace(cr[i+1:]), 10, 64); e == nil {
					total = v
				}
			}
		}
		if resp.StatusCode == http.StatusOK && off > 0 {
			// the server ignored Range: start over cleanly
			if _, e := f.Seek(0, io.SeekStart); e != nil {
				resp.Body.Close()
				cancel()
				return 0, 0, false, e
			}
			if e := f.Truncate(0); e != nil {
				resp.Body.Close()
				cancel()
				return 0, 0, false, e
			}
			off = 0
		}
		n, cerr := io.Copy(f, resp.Body)
		resp.Body.Close()
		cancel()
		if n > 0 {
			return n, total, resp.StatusCode == http.StatusOK, nil
		}
		if cerr == nil {
			return 0, total, true, nil
		}
		time.Sleep(time.Duration(attempt+1) * 400 * time.Millisecond)
	}
	return 0, total, false, fmt.Errorf("range fetch failed at offset %d", off)
}

// downloadFile copies a plain audio enclosure into a raw scratch file with resume.
func (d *Downloader) downloadFile(ctx context.Context, id int64, url, ctype string) error {
	ext := extFor(ctype, url)
	raw := rawPathFor(id, ext)
	if err := os.MkdirAll(filepath.Dir(raw), 0o755); err != nil {
		return err
	}
	if st, err := os.Stat(raw); err == nil && st.Size() > 0 {
		if _, err := finishDownload(ctx, id, raw); err != nil {
			return err
		}
		return nil
	}
	part := raw + ".part"
	var size int64
	if st, err := os.Stat(part); err == nil {
		size = st.Size()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", uaPodcastman)
	req.Header.Set("Referer", refererFor(url))
	if size > 0 {
		req.Header.Set("Range", "bytes="+itoa(size)+"-")
	}
	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return &feedError{status: resp.StatusCode}
	}
	appending := resp.StatusCode == http.StatusPartialContent && size > 0
	flags := os.O_CREATE | os.O_WRONLY
	if appending {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
		size = 0
	}
	f, err := os.OpenFile(part, flags, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		return err
	}
	f.Close()
	if err := os.Rename(part, raw); err != nil {
		return err
	}
	if _, err := finishDownload(ctx, id, raw); err != nil {
		return err
	}
	return nil
}
