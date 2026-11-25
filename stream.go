package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Streaming playback: the browser reads the remote media through us with Range
// pass-through, so a video episode starts playing immediately, can be seeked at
// once, and costs zero disk. ffmpeg is only needed for downloads and for
// containers the browser cannot demux (those fall back to download-then-play).
const uaLastcast = "Mozilla/5.0 (X11; Linux x86_64) lastcast"

var streamClient = &http.Client{
	Timeout: 0, // long media: no overall timeout
	Transport: &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		ResponseHeaderTimeout: 30 * time.Second,
		ForceAttemptHTTP2:     true,
	},
}

// browserPlayable containers: an HTML5 <audio> element can demux these and play the
// audio track (an mp4's video track is simply ignored).
var browserExts = map[string]bool{
	".mp3": true, ".m4a": true, ".m4b": true, ".mp4": true, ".m4v": true, ".mov": true,
	".aac": true, ".ogg": true, ".oga": true, ".opus": true, ".webm": true, ".wav": true, ".flac": true,
}

// canPlay decides whether a URL can go straight into <audio>: known extension or
// audio/video content type, otherwise a tiny Range peek looking for a head-visible
// container signature (this is what makes bilibili's extension-less .m4s work).
func canPlay(ctx context.Context, u, ctype string) bool {
	if browserExts[strings.ToLower(filepath.Ext(urlPath(u)))] {
		return true
	}
	ct := strings.ToLower(ctype)
	if strings.HasPrefix(ct, "audio/") {
		return true
	}
	if strings.HasPrefix(ct, "video/") && (strings.Contains(ct, "mp4") || strings.Contains(ct, "webm") || strings.Contains(ct, "quicktime")) {
		return true
	}
	return headPlayable(ctx, u)
}

func headPlayable(ctx context.Context, u string) bool {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return false
	}
	req.Header.Set("User-Agent", uaLastcast)
	req.Header.Set("Referer", refererFor(u))
	req.Header.Set("Range", "bytes=0-4095")
	resp, err := streamClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return false
	}
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return looksPlayableBytes(b)
}

func looksPlayableBytes(b []byte) bool {
	if len(b) < 8 {
		return false
	}
	if bytes.Contains(b, []byte("ftyp")) && bytes.Contains(b, []byte("moov")) {
		return true // faststart mp4 / in-band-init fMP4 (bilibili DASH audio)
	}
	if bytes.HasPrefix(b, []byte("ID3")) || bytes.HasPrefix(b, []byte("OggS")) ||
		bytes.HasPrefix(b, []byte("fLaC")) || bytes.HasPrefix(b, []byte("RIFF")) {
		return true
	}
	if len(b) >= 4 && b[0] == 0x1A && b[1] == 0x45 && b[2] == 0xDF && b[3] == 0xA3 {
		return true // WebM / Matroska (EBML): yt-dlp's usual bestaudio container
	}
	if b[0] == 0xFF && (b[1]&0xE0) == 0xE0 { // mp3 frame sync
		return true
	}
	return false
}

// mediaTypeForExt maps an extractor's ext to a content type a browser accepts.
func mediaTypeForExt(ext string) string {
	switch strings.TrimPrefix(strings.ToLower(ext), ".") {
	case "m4a", "m4b", "mp4", "m4v":
		return "audio/mp4"
	case "mp3":
		return "audio/mpeg"
	case "aac":
		return "audio/aac"
	case "ogg", "oga", "opus":
		return "audio/ogg"
	case "webm":
		return "audio/webm"
	case "flac":
		return "audio/flac"
	}
	return ""
}

func refererFor(u string) string {
	lower := strings.ToLower(u)
	// bilibili CDNs (and their P2P edge aliases) reject anything but the site referer.
	if strings.Contains(lower, "bilivideo") || strings.Contains(lower, "hdslb.com") ||
		strings.Contains(lower, "bilibili.com") || strings.Contains(lower, "mountaintoys") {
		return "https://www.bilibili.com/"
	}
	if i := strings.Index(u, "://"); i > 0 {
		rest := u[i+3:]
		if j := strings.IndexAny(rest, "/?"); j > 0 {
			return u[:i+3+j] + "/"
		}
		return u + "/"
	}
	return "https://www.bilibili.com/"
}

// sniffType asks the origin for the content type of an extension-less media URL.
func sniffType(ctx context.Context, u string) string {
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, u, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", uaLastcast)
	req.Header.Set("Referer", refererFor(u))
	resp, err := streamClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	ct := resp.Header.Get("Content-Type")
	if i := strings.IndexByte(ct, ';'); i >= 0 {
		ct = ct[:i]
	}
	return strings.TrimSpace(ct)
}

type cachedSource struct {
	src   string
	ctype string
	at    time.Time
}

var srcCache sync.Map

func cacheSource(id int64, src, ctype string) {
	srcCache.Store(id, cachedSource{src, ctype, time.Now()})
}

// streamSource resolves the URL a browser can play directly for this episode,
// preferring an audio-only stream over the video container.
func streamSource(ctx context.Context, id int64) (string, string, error) {
	if v, ok := srcCache.Load(id); ok {
		c := v.(cachedSource)
		if time.Since(c.at) < 15*time.Minute {
			return c.src, c.ctype, nil
		}
	}
	var enc, ctype, guid, desc, kind, audioURL, videoURL string
	err := store.db.QueryRow(`SELECT COALESCE(e.enclosure_url,''), COALESCE(e.enclosure_type,''), e.guid,
		COALESCE(e.description,''), COALESCE(e.kind,''), COALESCE(e.audio_url,''), COALESCE(e.video_url,'')
		FROM episodes e WHERE e.id=?`, id).Scan(&enc, &ctype, &guid, &desc, &kind, &audioURL, &videoURL)
	if err != nil {
		return "", "", fmt.Errorf("episode not found")
	}
	if kind == "text" {
		return "", "", fmt.Errorf("text episode plays cached segments")
	}
	// bilibili first: RSSHub items only carry a player page, but the DASH audio
	// URL is a browser-playable fMP4 with its ftyp+moov prefix inline.
	if ref := biliRef(enc + "\n" + guid + "\n" + desc); ref != "" {
		if t, err := biliResolve(ctx, ref); err == nil && t.URL != "" {
			enrichFromBili(id, t)
			if canPlay(ctx, t.URL, "audio/mp4") {
				cacheSource(id, t.URL, "audio/mp4")
				return t.URL, "audio/mp4", nil
			}
		}
	}
	for _, u := range []string{audioURL, videoURL} {
		if u != "" && canPlay(ctx, u, "") {
			cacheSource(id, u, "")
			return u, "", nil
		}
	}
	if u := extractMediaURL(desc); u != "" && canPlay(ctx, u, "") {
		cacheSource(id, u, "")
		return u, "", nil
	}
	if u, dur, err := peerTubeFile(ctx, enc); err == nil && u != "" && canPlay(ctx, u, "") {
		if dur > 0 {
			_, _ = store.db.Exec(`UPDATE episodes SET duration_sec=? WHERE id=? AND (duration_sec IS NULL OR duration_sec=0)`, dur, id)
		}
		cacheSource(id, u, "")
		return u, "", nil
	}
	if ytdlpAvailable() {
		if page := pageURLFor(enc, guid); page != "" {
			if info, err := ytdlpResolve(ctx, page); err == nil {
				enrichFromYtdlp(id, info)
				ct := mediaTypeForExt(info.Ext)
				if ytdlpStreamable(info) && canPlay(ctx, info.URL, ct) {
					cacheSource(id, info.URL, ct)
					return info.URL, ct, nil
				}
			}
		}
	}
	if enc != "" {
		ct := ctype
		if ct == "" || !strings.Contains(ct, "/") {
			ct = sniffType(ctx, enc)
		}
		if canPlay(ctx, enc, ct) {
			cacheSource(id, enc, ct)
			return enc, ct, nil
		}
	}
	return "", "", fmt.Errorf("no browser-playable audio stream (needs ffmpeg)")
}

// enrichIfBili resolves a bilibili item when its metadata is still missing, so a
// download/retag gets the real cover and UP name even without a playback first.
func enrichIfBili(ctx context.Context, id int64) {
	var enc, guid, desc, author, cover string
	if err := store.db.QueryRow(`SELECT COALESCE(e.enclosure_url,''), e.guid,
		COALESCE(e.description,''), COALESCE(e.author,''), COALESCE(e.cover_url,'')
		FROM episodes e WHERE e.id=?`, id).Scan(&enc, &guid, &desc, &author, &cover); err != nil {
		return
	}
	if author != "" && cover != "" {
		return
	}
	if ref := biliRef(enc + "\n" + guid + "\n" + desc); ref != "" {
		if t, err := biliResolve(ctx, ref); err == nil {
			enrichFromBili(id, t)
		}
	}
}

// enrichFromBili fills per-episode metadata the RSS item did not carry.
func enrichFromBili(id int64, t biliTrack) {
	pic := t.Pic
	if strings.HasPrefix(pic, "http://") {
		pic = "https://" + strings.TrimPrefix(pic, "http://")
	}
	if t.Dur > 0 {
		_, _ = store.db.Exec(`UPDATE episodes SET duration_sec=? WHERE id=? AND (duration_sec IS NULL OR duration_sec=0)`, t.Dur, id)
	}
	if pic != "" {
		_, _ = store.db.Exec(`UPDATE episodes SET cover_url=? WHERE id=? AND COALESCE(cover_url,'')=''`, pic, id)
	}
	if t.Author != "" {
		_, _ = store.db.Exec(`UPDATE episodes SET author=? WHERE id=? AND COALESCE(author,'')=''`, t.Author, id)
	}
}

// handleStream serves one episode for playback: the finished file when present,
// otherwise a Range pass-through of the remote media.
func handleStream(w http.ResponseWriter, r *http.Request) {
	id := atoi(r.PathValue("id"))
	if id == 0 {
		writeJSON(w, map[string]any{"error": "bad id"})
		return
	}
	uid := userID(r)
	var file, kind string
	var dur int64
	if err := store.db.QueryRow(`SELECT COALESCE(e.file_path,''), COALESCE(e.kind,''), COALESCE(e.duration_sec,0)
		FROM episodes e JOIN feeds f ON f.id=e.feed_id WHERE e.id=? AND f.user_id=?`, id, uid).Scan(&file, &kind, &dur); err != nil {
		writeJSONStatus(w, http.StatusNotFound, map[string]any{"error": "episode not found"})
		return
	}
	if file != "" {
		if st, err := os.Stat(file); err == nil && st.Size() > 0 {
			f, err := os.Open(file)
			if err != nil {
				writeJSON(w, map[string]any{"error": "file missing"})
				return
			}
			defer f.Close()
			http.ServeContent(w, r, filepath.Base(file), st.ModTime(), f)
			return
		}
	}
	if kind == "text" {
		writeJSON(w, map[string]any{"error": "text episode plays from cached segments", "fallback": "segment"})
		return
	}
	// "segment" mode means: keep a local copy (download first), never proxy.
	if store.getSetting("play_mode", "stream") == "segment" {
		writeJSON(w, map[string]any{"error": "play_mode=segment: download first", "fallback": "segment"})
		return
	}
	src, ctype, err := streamSource(r.Context(), int64(id))
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error(), "fallback": "segment"})
		return
	}
	proxyMedia(w, r, src, ctype)
	if dur == 0 {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if d := probeDuration(ctx, src); d > 0 {
				_, _ = store.db.Exec(`UPDATE episodes SET duration_sec=? WHERE id=? AND (duration_sec IS NULL OR duration_sec=0)`, d, id)
			}
		}()
	}
}

// guessedAudioType maps a generic/absent upstream content type to something a
// browser will actually decode (bilibili CDNs answer application/octet-stream).
func guessedAudioType(u string) string {
	switch strings.ToLower(filepath.Ext(urlPath(u))) {
	case ".mp3", "":
		if strings.EqualFold(filepath.Ext(urlPath(u)), ".mp3") {
			return "audio/mpeg"
		}
		return ""
	case ".m4a", ".m4b", ".mp4", ".m4v", ".mov", ".m4s":
		return "audio/mp4"
	case ".aac":
		return "audio/aac"
	case ".ogg", ".oga", ".opus":
		return "audio/ogg"
	case ".webm":
		return "audio/webm"
	case ".flac":
		return "audio/flac"
	}
	return ""
}

func genericContentType(ct string) bool {
	ct = strings.ToLower(strings.TrimSpace(ct))
	return ct == "" || strings.HasPrefix(ct, "application/octet-stream") ||
		strings.HasPrefix(ct, "binary/octet-stream") || strings.HasPrefix(ct, "application/binary")
}

func proxyMedia(w http.ResponseWriter, r *http.Request, src, ctype string) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeJSON(w, map[string]any{"error": "method not allowed"})
		return
	}
	req, err := http.NewRequestWithContext(r.Context(), r.Method, src, nil)
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	req.Header.Set("User-Agent", uaLastcast)
	req.Header.Set("Referer", refererFor(src))
	if rg := r.Header.Get("Range"); rg != "" {
		req.Header.Set("Range", rg)
	}
	resp, err := streamClient.Do(req)
	if err != nil {
		writeJSON(w, map[string]any{"error": "upstream: " + err.Error()})
		return
	}
	defer resp.Body.Close()
	for _, h := range []string{"Content-Type", "Content-Length", "Content-Range", "Accept-Ranges", "Last-Modified", "ETag"} {
		if v := resp.Header.Get(h); v != "" {
			w.Header().Set(h, v)
		}
	}
	ct := w.Header().Get("Content-Type")
	if ctype != "" && !strings.HasPrefix(strings.ToLower(ct), "audio/") {
		// e.g. googlevideo answers video/webm for an audio-only stream
		w.Header().Set("Content-Type", ctype)
	} else if genericContentType(ct) {
		if guess := guessedAudioType(src); guess != "" {
			w.Header().Set("Content-Type", guess)
		}
	}
	if w.Header().Get("Accept-Ranges") == "" {
		w.Header().Set("Accept-Ranges", "bytes")
	}
	w.Header().Set("Content-Disposition", "inline")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(resp.StatusCode)
	if r.Method != http.MethodHead {
		_, _ = copyFlushing(w, resp.Body)
	}
}

// ffprobePath locates ffprobe: the setting wins, otherwise the sibling of ffmpeg.
func ffprobePath() string {
	if v := store.getSetting("ffprobe_path", env("FFPROBE_PATH", "")); v != "" {
		return v
	}
	f := ffmpegPath()
	base := filepath.Base(f)
	if !strings.Contains(base, "ffmpeg") {
		return "ffprobe"
	}
	return filepath.Join(filepath.Dir(f), strings.Replace(base, "ffmpeg", "ffprobe", 1))
}

var durRe = regexp.MustCompile(`Duration:\s*(\d+):(\d+):(\d+(?:\.\d+)?)`)

// parseFFmpegDuration reads the "Duration: HH:MM:SS.ss" line ffmpeg prints.
func parseFFmpegDuration(s string) int64 {
	m := durRe.FindStringSubmatch(s)
	if m == nil {
		return 0
	}
	h, _ := strconv.Atoi(m[1])
	mi, _ := strconv.Atoi(m[2])
	sec, _ := strconv.ParseFloat(m[3], 64)
	return int64(h*3600+mi*60) + int64(sec+0.5)
}

// ffmpegProxyArgs passes the configured proxy to ffmpeg/ffprobe URL inputs.
func ffmpegProxyArgs() []string {
	if p := store.getSetting("http_proxy", ""); p != "" {
		return []string{"-http_proxy", p}
	}
	return nil
}

// probeDuration asks ffprobe for the media duration, falling back to ffmpeg's
// header line (ffmpeg stops after opening the input, so no full download).
func probeDuration(ctx context.Context, u string) int64 {
	if p, err := exec.LookPath(ffprobePath()); err == nil {
		args := append([]string{"-v", "error"}, ffmpegProxyArgs()...)
		args = append(args, "-headers", httpHeaders(u), "-show_entries", "format=duration", "-of", "default=nw=1:nk=1", u)
		cmd := exec.CommandContext(ctx, p, args...)
		if out, err := cmd.Output(); err == nil {
			if f, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64); err == nil && f > 0 {
				return int64(f + 0.5)
			}
		}
	}
	if _, err := exec.LookPath(ffmpegPath()); err == nil {
		args := append([]string{"-hide_banner"}, ffmpegProxyArgs()...)
		args = append(args, "-headers", httpHeaders(u), "-i", u)
		cmd := exec.CommandContext(ctx, ffmpegPath(), args...)
		if out, _ := cmd.CombinedOutput(); len(out) > 0 {
			return parseFFmpegDuration(string(out))
		}
	}
	return 0
}

// serveFileRange is the shared ServeContent helper for cache segments.
func serveFileRange(w http.ResponseWriter, r *http.Request, path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || st.IsDir() || st.Size() == 0 {
		return false
	}
	http.ServeContent(w, r, filepath.Base(path), st.ModTime(), f)
	return true
}
