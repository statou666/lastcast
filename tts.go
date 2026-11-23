package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// TTS talks to any OpenAI-compatible /audio/speech endpoint you configure in
// Settings (base URL + key), so a self-hosted vLLM/LocalAI/Ollama-TTS gateway
// works the same way as api.openai.com.
type ttsCfg struct {
	Provider string
	Base     string
	Key      string
	Model    string
	Voice    string
	Speed    float64
	MaxChars int
}

var ttsClient = &http.Client{Timeout: 120 * time.Second}

func ttsConfig() ttsCfg {
	d := settingsMap()
	speed, err := strconv.ParseFloat(store.getSetting("tts_speed", d["tts_speed"]), 64)
	if err != nil || speed <= 0 {
		speed = 1
	}
	maxChars, err := strconv.Atoi(store.getSetting("tts_max_chars", d["tts_max_chars"]))
	if err != nil || maxChars < 200 {
		maxChars = 2000
	}
	return ttsCfg{
		Provider: store.getSetting("tts_provider", d["tts_provider"]),
		Base:     strings.TrimRight(store.getSetting("tts_base_url", d["tts_base_url"]), "/"),
		Key:      store.getSetting("tts_key", d["tts_key"]),
		Model:    store.getSetting("tts_model", d["tts_model"]),
		Voice:    store.getSetting("tts_voice", d["tts_voice"]),
		Speed:    speed,
		MaxChars: maxChars,
	}
}

var tagRe = regexp.MustCompile(`<[^>]*>`)

// plainText turns an HTML show-notes body into speakable text.
func plainText(s string) string {
	s = tagRe.ReplaceAllString(s, "\n")
	s = html.UnescapeString(s)
	lines := strings.Split(s, "\n")
	var out []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			out = append(out, l)
		}
	}
	return strings.Join(out, "\n")
}

// chunkSource is the text a text item is spoken from: content:encoded when the
// feed provides it, otherwise description (summary-only feeds such as sspai).
func chunkSource(body, desc string) string {
	if strings.TrimSpace(body) != "" {
		return body
	}
	return desc
}

// chunks splits text into request-sized pieces. Short paragraphs are merged up
// to max so a feed with many one-line paragraphs does not become hundreds of
// requests; paragraph breaks are still preferred over cutting mid-sentence.
func chunks(text string, max int) []string {
	if max < 200 {
		max = 200
	}
	var out []string
	var cur strings.Builder
	for _, para := range strings.Split(text, "\n") {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}
		if cur.Len() == 0 {
			for len(para) > max {
				// hard cut at max (Chinese text often has no spaces to break on)
				cut := max
				if i := strings.LastIndex(para[:max], " "); i > max/2 {
					cut = i
				}
				out = append(out, strings.TrimSpace(para[:cut]))
				para = strings.TrimSpace(para[cut:])
			}
			if para == "" {
				continue
			}
			cur.WriteString(para)
			continue
		}
		if cur.Len()+len(para)+1 <= max {
			cur.WriteString("\n")
			cur.WriteString(para)
			continue
		}
		out = append(out, cur.String())
		cur.Reset()
		for len(para) > max {
			// hard cut at max (Chinese text often has no spaces to break on)
			cut := max
			if i := strings.LastIndex(para[:max], " "); i > max/2 {
				cut = i
			}
			out = append(out, strings.TrimSpace(para[:cut]))
			para = strings.TrimSpace(para[cut:])
		}
		if para != "" {
			cur.WriteString(para)
		}
	}
	if cur.Len() > 0 {
		out = append(out, cur.String())
	}
	return out
}

// speak sends one chunk and returns raw audio bytes.
func (c ttsCfg) speak(ctx context.Context, text string) ([]byte, error) {
	if c.Provider == "off" || c.Provider == "" {
		return nil, fmt.Errorf("tts provider is off")
	}
	if c.Key == "" {
		return nil, fmt.Errorf("tts key is empty")
	}
	body, err := json.Marshal(map[string]any{"model": c.Model, "input": text, "voice": c.Voice, "speed": c.Speed})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Base+"/audio/speech", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Key)
	resp, err := ttsClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 || strings.Contains(resp.Header.Get("Content-Type"), "json") {
		var e struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(data, &e) == nil && e.Error.Message != "" {
			return nil, fmt.Errorf("tts %d: %s", resp.StatusCode, e.Error.Message)
		}
		return nil, fmt.Errorf("tts %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	return data, nil
}

// synthesize renders a text episode into one audio file (the same file path the
// player and the audiobook packer use).
func (d *Downloader) synthesize(ctx context.Context, id int64) error {
	cfg := ttsConfig()
	var feedTitle, guid, title, body, desc string
	if err := store.db.QueryRow(`SELECT f.title, e.guid, e.title, e.body, COALESCE(e.description,'') FROM episodes e
		JOIN feeds f ON f.id = e.feed_id WHERE e.id = ?`, id).Scan(&feedTitle, &guid, &title, &body, &desc); err != nil {
		return err
	}
	text := plainText(body)
	if text == "" {
		text = plainText(desc)
	}
	if text == "" {
		text = plainText(title)
	}
	if text == "" {
		return fmt.Errorf("no text to speak")
	}
	if d.overLimit() {
		return fmt.Errorf("storage limit reached")
	}
	// TTS chunks are disposable cache; the joined mp3 is a download.
	segDir := d.segDir(id)
	if err := os.MkdirAll(segDir, 0o755); err != nil {
		return err
	}
	segs := chunks(text, cfg.MaxChars)
	_, _ = store.db.Exec(`UPDATE episodes SET segments=?, status='working' WHERE id=?`, len(segs), id)

	// Segments are written one by one so the player can start on the first one
	// instead of waiting for the whole episode; already-rendered segments are
	// never re-requested, so a retry only pays for what is missing.
	for i := range segs {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		sp := filepath.Join(segDir, fmt.Sprintf("%d.mp3", i+1))
		if st, err := os.Stat(sp); err != nil || st.Size() == 0 {
			data, err := cfg.speak(ctx, segs[i])
			if err != nil {
				_, _ = store.db.Exec(`UPDATE episodes SET status='error' WHERE id=?`, id)
				return fmt.Errorf("chunk %d: %w", i+1, err)
			}
			if err := os.WriteFile(sp, data, 0o644); err != nil {
				return err
			}
		}
		_, _ = store.db.Exec(`UPDATE episodes SET segments_done=? WHERE id=?`, i+1, id)
	}

	raw := rawPathFor(id, ".mp3")
	if err := os.MkdirAll(filepath.Dir(raw), 0o755); err != nil {
		return err
	}
	f, err := os.Create(raw)
	if err != nil {
		return err
	}
	for i := range segs {
		b, err := os.ReadFile(filepath.Join(segDir, itoa(int64(i+1))+".mp3"))
		if err != nil {
			f.Close()
			return err
		}
		if _, err := f.Write(b); err != nil {
			f.Close()
			return err
		}
	}
	f.Close()
	if _, err := finishDownload(ctx, id, raw); err != nil {
		return err
	}
	store.logEvent("episode", guid, "tts")
	return nil
}

// testTTS sends one short line so the Settings page can verify a configured
// endpoint without downloading a whole episode.
func testTTS(ctx context.Context) (int, error) {
	cfg := ttsConfig()
	data, err := cfg.speak(ctx, "Hello, this is a test.")
	if err != nil {
		return 0, err
	}
	return len(data), nil
}

// handleTTS: POST /api/tts {episode_id} renders one text item;
// POST /api/tts/test checks the configured endpoint with a short line.
func handleTTS(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/api/tts/test" {
		n, err := testTTS(r.Context())
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		writeJSON(w, map[string]any{"ok": true, "bytes": n})
		return
	}
	var body struct {
		EpisodeID int64 `json:"episode_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, map[string]any{"error": "invalid json"})
		return
	}
	var exists int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM episodes e JOIN feeds f ON f.id=e.feed_id WHERE e.id=? AND f.user_id=?`,
		body.EpisodeID, userID(r)).Scan(&exists); err != nil || exists == 0 {
		writeJSON(w, map[string]any{"error": "episode not found"})
		return
	}
	// Queued, not awaited: the caller can start playing segment 1 right away.
	dl.Enqueue(body.EpisodeID)
	writeJSON(w, map[string]any{"ok": true, "queued": true})
}

// handleSpeak streams a text episode as audio while it is being synthesized, so a
// podcast client can start playing within a second instead of waiting for the
// whole episode. Once finished the tagged file is stored like a normal download.
func handleSpeak(w http.ResponseWriter, r *http.Request) {
	id := int64(atoi(r.PathValue("id")))
	if id == 0 || !episodeOwned(userID(r), id) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	var kind, file, body, desc, title string
	if err := store.db.QueryRow(`SELECT COALESCE(kind,''),COALESCE(file_path,''),COALESCE(body,''),
		COALESCE(description,''),COALESCE(title,'') FROM episodes WHERE id=?`, id).
		Scan(&kind, &file, &body, &desc, &title); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if file != "" {
		if f, err := os.Open(file); err == nil {
			defer f.Close()
			if st, err := f.Stat(); err == nil && st.Size() > 0 {
				http.ServeContent(w, r, filepath.Base(file), st.ModTime(), f)
				return
			}
		}
	}
	if kind != "text" {
		http.Error(w, "not a text episode", http.StatusNotFound)
		return
	}
	cfg := ttsConfig()
	if cfg.Provider == "off" {
		http.Error(w, "tts is off", http.StatusServiceUnavailable)
		return
	}
	text := plainText(body)
	if text == "" {
		text = plainText(desc)
	}
	if text == "" {
		text = plainText(title)
	}
	if text == "" {
		http.Error(w, "no text", http.StatusNotFound)
		return
	}
	segs := chunks(text, cfg.MaxChars)
	segDir := segmentDirFor(id)
	_ = os.MkdirAll(segDir, 0o755)

	// fetch a chunk, reusing the cache the downloader/TTS renderer already built
	getChunk := func(i int) ([]byte, error) {
		sp := filepath.Join(segDir, itoa(int64(i+1))+".mp3")
		if b, err := os.ReadFile(sp); err == nil && len(b) > 0 {
			return b, nil
		}
		b, err := cfg.speak(r.Context(), segs[i])
		if err != nil {
			return nil, err
		}
		_ = os.WriteFile(sp, b, 0o644)
		_, _ = store.db.Exec(`UPDATE episodes SET segments=?, segments_done=MAX(segments_done,?) WHERE id=?`, len(segs), i+1, id)
		return b, nil
	}
	if r.Method == http.MethodHead {
		w.Header().Set("Content-Type", "audio/mpeg")
		return
	}
	first, err := getChunk(0)
	if err != nil {
		http.Error(w, "tts: "+err.Error(), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Disposition", `inline; filename="`+safeName(title)+`.mp3"`)
	w.WriteHeader(http.StatusOK)
	flusher, _ := w.(http.Flusher)
	n, _ := w.Write(first)
	if flusher != nil {
		flusher.Flush()
	}
	raw := rawPathFor(id, ".mp3")
	_ = os.MkdirAll(filepath.Dir(raw), 0o755)
	f, err := os.Create(raw)
	if err == nil {
		_, _ = f.Write(first)
	}
	for i := 1; i < len(segs); i++ {
		if r.Context().Err() != nil {
			if f != nil {
				f.Close()
				_ = os.Remove(raw)
			}
			return
		}
		b, err := getChunk(i)
		if err != nil {
			// headers are already sent: stop the stream, keep what we have
			if f != nil {
				f.Close()
				_ = os.Remove(raw)
			}
			return
		}
		if _, err := w.Write(b); err != nil {
			if f != nil {
				f.Close()
				_ = os.Remove(raw)
			}
			return
		}
		n += len(b)
		if f != nil {
			_, _ = f.Write(b)
		}
		if flusher != nil {
			flusher.Flush()
		}
	}
	if f != nil {
		f.Close()
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			if _, err := finishDownload(ctx, id, raw); err != nil {
				// the client already received the audio; do not turn the episode red
				store.logEvent("episode", itoa(id), "speak_finish_error: "+err.Error())
			}
		}()
	}
	_ = n
}
