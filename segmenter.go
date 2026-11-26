package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Video feeds are played as audio only.
//
// RSSHub video routes (e.g. /bilibili/user/video/<uid>) usually carry no
// <enclosure> at all — the link is an <iframe> with a bvid inside description.
// So we resolve the bvid to the platform's audio stream, then let ffmpeg's
// segment muxer write one playable file per chunk WHILE it is still reading the
// network input: segment 1 is playable within seconds instead of waiting for the
// whole video. Without an ffmpeg binary we fall back to downloading one file.
//
// Segments live in the same layout the TTS renderer uses:
// media/<slug>/<episode id>/<n>.m4a — so /api/segments/{id}, /media/{id}/{seg},
// the player's "segment n/N" display and progress.segment_idx work unchanged.

func jsonInt(v any) int64 {
	switch n := v.(type) {
	case float64:
		return int64(n)
	case int64:
		return n
	}
	return 0
}

func ffmpegPath() string {
	if v := store.getSetting("ffmpeg_path", env("FFMPEG_PATH", "ffmpeg")); v != "" {
		return v
	}
	return "ffmpeg"
}

func ffmpegAvailable() bool {
	_, err := exec.LookPath(ffmpegPath())
	return err == nil
}

func segmentSeconds() int {
	if n, err := strconv.Atoi(store.getSetting("segment_seconds", env("SEGMENT_SECONDS", "30"))); err == nil && n > 0 {
		return n
	}
	return 30
}

// segmentExt: the extension the player and /media/{id}/{seg} must use for an episode.
func segmentExt(id int64) string {
	var kind string
	if err := store.db.QueryRow(`SELECT kind FROM episodes WHERE id=?`, id).Scan(&kind); err != nil {
		return ".mp3"
	}
	if kind == "video" {
		return ".m4a"
	}
	return ".mp3"
}

// audioExts: containers an HTML5 <audio> element can play directly.
var audioExts = map[string]bool{
	".mp3": true, ".m4a": true, ".aac": true, ".ogg": true, ".opus": true, ".webm": true,
}

func urlPath(u string) string {
	if i := strings.IndexByte(u, '?'); i >= 0 {
		return u[:i]
	}
	return u
}

// biliHeaders is the header block ffmpeg needs too: the CDN rejects ffmpeg's
// default UA with 403 even though the signed URL is signed.
func biliHeaders() string {
	return "User-Agent: Mozilla/5.0 (X11; Linux x86_64) lastcast\r\nReferer: https://www.bilibili.com/"
}

// httpHeaders is the header block ffmpeg needs for an arbitrary media URL: a browser
// UA (CDNs reject ffmpeg's default UA) plus a Referer matching the URL's origin.
func httpHeaders(u string) string {
	ref := biliHeaders()
	if i := strings.Index(u, "://"); i > 0 {
		rest := u[i+3:]
		if j := strings.IndexAny(rest, "/?"); j > 0 {
			ref = "User-Agent: Mozilla/5.0 (X11; Linux x86_64) lastcast\r\nReferer: " + u[:i+3+j] + "/"
		}
	}
	return ref
}

func isAudio(u, ctype string) bool {
	if strings.HasPrefix(strings.ToLower(ctype), "audio/") {
		return true
	}
	return audioExts[strings.ToLower(filepath.Ext(urlPath(u)))]
}

// isMedia reports whether ffmpeg can read the URL: an audio type, a video type, or
// an extension from either table. PeerTube media:content URLs often have no
// extension at all (…/generate/<uuid>?videoFileIds=…), so the type matters too.
func isMedia(u, ctype string) bool {
	ct := strings.ToLower(ctype)
	if strings.HasPrefix(ct, "audio/") || strings.HasPrefix(ct, "video/") {
		return true
	}
	e := strings.ToLower(filepath.Ext(urlPath(u)))
	return audioExts[e] || videoExts[e]
}

// videoExts are containers ffmpeg can read but an HTML5 audio element cannot, so
// they only qualify when ffmpeg is available (ffmpeg takes the audio track out).
var videoExts = map[string]bool{
	".mp4": true, ".m4v": true, ".m4s": true, ".mkv": true, ".mov": true,
}

// biliGet calls the bilibili API — it answers 412 without a browser UA + Referer.
func biliGet(ctx context.Context, u string) (any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) lastcast")
	req.Header.Set("Referer", "https://www.bilibili.com/")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bilibili %s: HTTP %d", u, res.StatusCode)
	}
	var top map[string]any
	if err := json.NewDecoder(res.Body).Decode(&top); err != nil {
		return nil, err
	}
	if code, _ := top["code"].(float64); code != 0 {
		return nil, fmt.Errorf("bilibili API code %v", top["code"])
	}
	data, ok := top["data"]
	if !ok || data == nil {
		return nil, fmt.Errorf("bilibili API: no data")
	}
	return data, nil
}

// resolveAudioCandidates maps any item to an ordered list of playable sources.
// Bilibili returns several CDN URLs (upos first, mcdn P2P edge last).
func resolveAudioCandidates(ctx context.Context, enc, ctype, guid, desc string) ([]string, int64, error) {
	if enc != "" && isAudio(enc, ctype) {
		return []string{enc}, 0, nil
	}
	hay := enc + "\n" + guid + "\n" + desc
	if ref := biliRef(hay); ref != "" {
		t, err := biliResolve(ctx, ref)
		if err != nil {
			return nil, 0, err
		}
		if len(t.URLs) > 0 {
			return t.URLs, t.Dur, nil
		}
	}
	if u := extractMediaURL(desc); u != "" {
		return []string{u}, 0, nil
	}
	if u, dur, err := peerTubeFile(ctx, enc); err == nil && u != "" {
		return []string{u}, dur, nil
	}
	if u, dur, err := peerTubeFile(ctx, guid); err == nil && u != "" {
		return []string{u}, dur, nil
	}
	if enc != "" && isMedia(enc, ctype) {
		return []string{enc}, 0, nil
	}
	if ytdlpAvailable() {
		if page := pageURLFor(enc, guid); page != "" {
			if info, err := ytdlpResolve(ctx, page); err == nil && ytdlpStreamable(info) {
				return []string{info.URL}, info.Duration, nil
			}
		}
	}
	return nil, 0, fmt.Errorf("no playable audio stream")
}

func resolveAudioURL(ctx context.Context, enc, ctype, guid, desc string) (string, int64, error) {
	urls, dur, err := resolveAudioCandidates(ctx, enc, ctype, guid, desc)
	if err != nil {
		return "", 0, err
	}
	if len(urls) == 0 {
		return "", 0, fmt.Errorf("no playable audio stream")
	}
	return urls[0], dur, nil
}

// segDir is the per-episode segment folder inside the disposable cache root
// (same layout the TTS renderer uses).
func (d *Downloader) segDir(id int64) string {
	return segmentDirFor(id)
}

// countSegments counts ready segment files (numeric names) in dir.
func countSegments(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	n := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if _, err := strconv.Atoi(strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))); err == nil {
			n++
		}
	}
	return n
}

// segmentsViaFFmpeg runs ffmpeg's segment muxer and records readiness as each file
// appears, so the player can start on segment 1 while ffmpeg is still reading.
func (d *Downloader) segmentsViaFFmpeg(ctx context.Context, id int64, url string) error {
	dir := d.segDir(id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	out := filepath.Join(dir, "%d.m4a")
	args := append([]string{"-hide_banner", "-loglevel", "error", "-nostats"}, ffmpegProxyArgs()...)
	args = append(args, "-headers", httpHeaders(url), "-i", url,
		"-vn", "-c:a", "copy", "-f", "segment",
		"-segment_time", strconv.Itoa(segmentSeconds()), "-segment_start_number", "1",
		"-reset_timestamps", "1", out)
	cmd := exec.CommandContext(ctx, ffmpegPath(), args...)
	if err := cmd.Start(); err != nil {
		return err
	}
	var stderr strings.Builder
	cmd.Stderr = &stderr
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case err := <-done:
			n := countSegments(dir)
			if n == 0 {
				if err != nil {
					return err
				}
				return fmt.Errorf("ffmpeg wrote no segments")
			}
			// Segments are a disposable playback cache, NOT a download: record
			// readiness only, never touch the episode's download status. A played
			// (streamed/segmented) episode must keep its real status so the history
			// and list views do not show "downloaded" for something never downloaded.
			_, _ = d.store.db.Exec(`UPDATE episodes SET segments=?, segments_done=? WHERE id=?`, n, n, id)
			// ffmpeg prints the input duration while it opens the stream: free backfill.
			if dur := parseFFmpegDuration(stderr.String()); dur > 0 {
				_, _ = d.store.db.Exec(`UPDATE episodes SET duration_sec=? WHERE id=? AND (duration_sec IS NULL OR duration_sec=0)`, dur, id)
			}
			return nil
		case <-ticker.C:
			if n := countSegments(dir); n > 0 {
				_, _ = d.store.db.Exec(`UPDATE episodes SET segments=?, segments_done=? WHERE id=?`, n, n, id)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
