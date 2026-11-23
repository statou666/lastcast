package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// yt-dlp covers the sites that only expose a watch page (YouTube, Vimeo, ...):
// it resolves the audio stream URL for playback and can download the best audio
// track for the album library. Baked into the image (`apk add yt-dlp`).
type ytdlpInfo struct {
	URL       string `json:"url"`
	Ext       string `json:"ext"`
	Duration  int64  `json:"duration"`
	Title     string `json:"title"`
	Uploader  string `json:"uploader"`
	Thumbnail string `json:"thumbnail"`
	Headers   struct {
		UserAgent string `json:"User-Agent"`
		Referer   string `json:"Referer"`
	} `json:"http_headers"`
}

func ytdlpPath() string {
	if v := store.getSetting("ytdlp_path", env("YTDLP_PATH", "")); v != "" {
		return v
	}
	return "yt-dlp"
}

func ytdlpAvailable() bool {
	_, err := exec.LookPath(ytdlpPath())
	return err == nil
}

var (
	ytMu    sync.Mutex
	ytCache = map[string]ytdlpInfo{}
	ytAt    = map[string]time.Time{}
)

// pageURLFor picks the watch-page URL of an item (guid is usually the page).
func pageURLFor(enc, guid string) string {
	for _, u := range []string{guid, enc} {
		u = strings.TrimSpace(u)
		if strings.HasPrefix(u, "http") && !isMedia(u, "") {
			return u
		}
	}
	return ""
}

// ytdlpResolve returns the direct audio URL plus metadata, cached for 25 minutes.
func ytdlpResolve(ctx context.Context, page string) (ytdlpInfo, error) {
	ytMu.Lock()
	if info, ok := ytCache[page]; ok && time.Since(ytAt[page]) < 25*time.Minute {
		ytMu.Unlock()
		return info, nil
	}
	ytMu.Unlock()

	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	args := []string{"-j", "--no-playlist", "--no-warnings", "--no-progress", "-f", "bestaudio/best"}
	if p := store.getSetting("http_proxy", ""); p != "" {
		args = append(args, "--proxy", p)
	}
	args = append(args, page)
	cmd := exec.CommandContext(ctx, ytdlpPath(), args...)
	out, err := cmd.Output()
	if err != nil {
		return ytdlpInfo{}, fmt.Errorf("yt-dlp: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var info ytdlpInfo
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) == "" {
			continue
		}
		if json.Unmarshal([]byte(lines[i]), &info) == nil {
			break
		}
	}
	if info.URL == "" {
		return ytdlpInfo{}, fmt.Errorf("yt-dlp: no audio url")
	}
	ytMu.Lock()
	ytCache[page] = info
	ytAt[page] = time.Now()
	ytMu.Unlock()
	return info, nil
}

// ytdlpStreamable reports whether the extracted URL is something <audio> can play.
// HLS/DASH manifests need the download path instead.
func ytdlpStreamable(info ytdlpInfo) bool {
	ext := strings.ToLower(info.Ext)
	if ext == "m3u8" || ext == "mpd" {
		return false
	}
	return browserExts["."+ext] || streamPlayableURL(info.URL)
}

func streamPlayableURL(u string) bool {
	return browserExts[strings.ToLower(filepath.Ext(urlPath(u)))]
}

// ytdlpDownload fetches the best audio track into the scratch area and returns it.
func (d *Downloader) ytdlpDownload(ctx context.Context, id int64, page string) (string, error) {
	dir := filepath.Join(cacheRoot(), "tmp", fmt.Sprintf("%d.ytdlp", id))
	_ = os.RemoveAll(dir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()
	args := []string{
		"--no-playlist", "--no-warnings", "--no-progress", "--no-mtime",
		"-f", "bestaudio/best",
		"-o", filepath.Join(dir, "%(id)s.%(ext)s"),
	}
	if p := store.getSetting("http_proxy", ""); p != "" {
		args = append(args, "--proxy", p)
	}
	args = append(args, page)
	if out, err := exec.CommandContext(ctx, ytdlpPath(), args...).CombinedOutput(); err != nil {
		return "", fmt.Errorf("yt-dlp: %v: %s", err, tailText(string(out), 300))
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	best := ""
	var bestSize int64
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil || info.Size() == 0 {
			continue
		}
		if info.Size() > bestSize {
			best, bestSize = filepath.Join(dir, e.Name()), info.Size()
		}
	}
	if best == "" {
		return "", fmt.Errorf("yt-dlp produced no file")
	}
	return best, nil
}

// enrichFromYtdlp fills cover/author/duration from the extractor metadata.
func enrichFromYtdlp(id int64, info ytdlpInfo) {
	if info.Duration > 0 {
		_, _ = store.db.Exec(`UPDATE episodes SET duration_sec=? WHERE id=? AND (duration_sec IS NULL OR duration_sec=0)`, info.Duration, id)
	}
	if info.Thumbnail != "" {
		_, _ = store.db.Exec(`UPDATE episodes SET cover_url=? WHERE id=? AND COALESCE(cover_url,'')=''`, info.Thumbnail, id)
	}
	if info.Uploader != "" {
		_, _ = store.db.Exec(`UPDATE episodes SET author=? WHERE id=? AND COALESCE(author,'')=''`, info.Uploader, id)
	}
}

func tailText(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}
