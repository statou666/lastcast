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
	"sync"
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
var blockRe = regexp.MustCompile(`(?i)</(p|div|section|article|h[1-6]|li|tr|blockquote|pre|figure)>|<br\s*/?>`)

// plainText turns an HTML show-notes body into speakable text, keeping block
// boundaries as blank lines so the segmenter can split strictly by paragraph.
func plainText(s string) string {
	s = blockRe.ReplaceAllString(s, "\n\n")
	s = tagRe.ReplaceAllString(s, " ")
	s = html.UnescapeString(s)
	lines := strings.Split(s, "\n")
	var out []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" {
			if len(out) > 0 && out[len(out)-1] != "" {
				out = append(out, "")
			}
			continue
		}
		out = append(out, l)
	}
	for len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
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

// chunks splits text strictly by paragraph: one segment per paragraph, and a
// paragraph longer than max is further split at sentence boundaries (or hard-cut
// by rune count when there is no punctuation). Short paragraphs are never merged.
func chunks(text string, max int) []string {
	if max < 200 {
		max = 200
	}
	var out []string
	for _, p := range paragraphs(text) {
		if len([]rune(p)) <= max {
			out = append(out, p)
			continue
		}
		out = append(out, splitLongParagraph(p, max)...)
	}
	return out
}

var blankLineRe = regexp.MustCompile(`\n\s*\n`)

func paragraphs(text string) []string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	var out []string
	for _, block := range blankLineRe.Split(text, -1) {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		// A block without blank lines may still be several lines (feeds often put
		// one paragraph per line); keep those as separate paragraphs too.
		for _, line := range strings.Split(block, "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				out = append(out, line)
			}
		}
	}
	return out
}

func sentenceEnd(r rune) bool {
	switch r {
	case '。', '！', '？', '.', '!', '?', ';', '；', '…':
		return true
	}
	return false
}

func splitLongParagraph(p string, max int) []string {
	runes := []rune(strings.TrimSpace(p))
	var out []string
	for len(runes) > max {
		cut := max
		for i := max; i > max*2/3; i-- {
			if sentenceEnd(runes[i-1]) {
				cut = i
				break
			}
		}
		piece := strings.TrimSpace(string(runes[:cut]))
		if piece != "" {
			out = append(out, piece)
		}
		runes = []rune(strings.TrimSpace(string(runes[cut:])))
	}
	if len(runes) > 0 {
		out = append(out, string(runes))
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
	segs, segDir, _, guid, err := d.prepareSegments(ctx, id)
	if err != nil {
		return err
	}
	if d.overLimit() {
		return fmt.Errorf("storage limit reached")
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

// prepareSegments renders (or reuses) the per-paragraph cache files only. It does
// NOT join them into a download: playing a text episode must stay disposable
// cache until the user explicitly hits download.
func episodeTextForTTS(id int64) (guid, text string, err error) {
	var title, body, desc, full string
	if err := store.db.QueryRow(`SELECT e.guid, e.title, COALESCE(e.full_text,''), e.body, COALESCE(e.description,'') FROM episodes e WHERE e.id = ?`, id).
		Scan(&guid, &title, &full, &body, &desc); err != nil {
		return "", "", err
	}
	text = plainText(full)
	if text == "" {
		text = plainText(body)
	}
	if text == "" {
		text = plainText(desc)
	}
	if text == "" {
		text = plainText(title)
	}
	if text == "" {
		return "", "", fmt.Errorf("no text to speak")
	}
	return guid, text, nil
}

func (d *Downloader) prepareSegments(ctx context.Context, id int64) ([]string, string, ttsCfg, string, error) {
	cfg := ttsConfig()
	guid, text, err := episodeTextForTTS(id)
	if err != nil {
		return nil, "", cfg, "", err
	}
	segDir := d.segDir(id)
	if err := os.MkdirAll(segDir, 0o755); err != nil {
		return nil, "", cfg, "", err
	}
	segs := chunks(text, cfg.MaxChars)
	_, _ = store.db.Exec(`UPDATE episodes SET segments=? WHERE id=?`, len(segs), id)
	for i := range segs {
		if ctx.Err() != nil {
			return nil, "", cfg, "", ctx.Err()
		}
		sp := filepath.Join(segDir, fmt.Sprintf("%d.mp3", i+1))
		if st, err := os.Stat(sp); err != nil || st.Size() == 0 {
			data, err := cfg.speak(ctx, segs[i])
			if err != nil {
				store.logEvent("episode", guid, "tts_error: "+err.Error())
				return nil, "", cfg, "", fmt.Errorf("chunk %d: %w", i+1, err)
			}
			if err := os.WriteFile(sp, data, 0o644); err != nil {
				return nil, "", cfg, "", err
			}
		}
		_, _ = store.db.Exec(`UPDATE episodes SET segments_done=? WHERE id=?`, i+1, id)
	}
	return segs, segDir, cfg, guid, nil
}

// renderOneSegment renders just one paragraph (the player asks for the next one
// only when it needs it, so switching episodes never queues a whole article).
func (d *Downloader) renderOneSegment(ctx context.Context, id int64, n int) error {
	cfg := ttsConfig()
	guid, text, err := episodeTextForTTS(id)
	if err != nil {
		return err
	}
	segs := chunks(text, cfg.MaxChars)
	if n < 1 || n > len(segs) {
		return fmt.Errorf("segment %d out of range (total %d)", n, len(segs))
	}
	segDir := d.segDir(id)
	if err := os.MkdirAll(segDir, 0o755); err != nil {
		return err
	}
	sp := filepath.Join(segDir, fmt.Sprintf("%d.mp3", n))
	if st, err := os.Stat(sp); err == nil && st.Size() > 0 {
		_, _ = store.db.Exec(`UPDATE episodes SET segments=? WHERE id=?`, len(segs), id)
		return nil
	}
	data, err := cfg.speak(ctx, segs[n-1])
	if err != nil {
		store.logEvent("episode", guid, "tts_error: "+err.Error())
		return fmt.Errorf("chunk %d: %w", n, err)
	}
	if err := os.WriteFile(sp, data, 0o644); err != nil {
		return err
	}
	_, _ = store.db.Exec(`UPDATE episodes SET segments=?, segments_done=? WHERE id=?`, len(segs), countSegments(segDir), id)
	return nil
}

// renderSegmentsOnly is the playback path: build the segment cache, never a download.
func (d *Downloader) renderSegmentsOnly(ctx context.Context, id int64) error {
	_, _, _, _, err := d.prepareSegments(ctx, id)
	return err
}

// handleEpisodeSegments: POST /api/episodes/{id}/segments — start rendering the
// paragraph cache for instant playback (no permanent file).
func handleEpisodeSegments(w http.ResponseWriter, r *http.Request) {
	id := int64(atoi(r.PathValue("id")))
	if id == 0 || !episodeOwned(userID(r), id) {
		writeJSON(w, map[string]any{"error": "episode not found"})
		return
	}
	var body struct {
		Segment int `json:"segment"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body.Segment <= 0 {
		body.Segment = 1
	}
	startSegmentRender(id, body.Segment)
	writeJSON(w, map[string]any{"ok": true, "queued": true, "segment": body.Segment})
}

var segRendering = map[string]bool{}
var segRenderMu sync.Mutex

func startSegmentRender(id int64, n int) {
	key := fmt.Sprintf("%d:%d", id, n)
	segRenderMu.Lock()
	if segRendering[key] {
		segRenderMu.Unlock()
		return
	}
	segRendering[key] = true
	segRenderMu.Unlock()
	go func() {
		defer func() {
			segRenderMu.Lock()
			delete(segRendering, key)
			segRenderMu.Unlock()
		}()
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		if err := dl.renderOneSegment(ctx, id, n); err != nil {
			store.logEvent("episode", itoa(id), "segments_error: "+err.Error())
		}
	}()
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
	var kind, file, body, desc, title, full string
	if err := store.db.QueryRow(`SELECT COALESCE(kind,''),COALESCE(file_path,''),COALESCE(body,''),
		COALESCE(description,''),COALESCE(title,''),COALESCE(full_text,'') FROM episodes WHERE id=?`, id).
		Scan(&kind, &file, &body, &desc, &title, &full); err != nil {
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
	text := plainText(full)
	if text == "" {
		text = plainText(body)
	}
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
	_, _ = w.Write(first)
	if flusher != nil {
		flusher.Flush()
	}
	for i := 1; i < len(segs); i++ {
		if r.Context().Err() != nil {
			return
		}
		b, err := getChunk(i)
		if err != nil {
			// headers are already sent: stop the stream, keep the cached chunks
			return
		}
		if _, err := w.Write(b); err != nil {
			return
		}
		if flusher != nil {
			flusher.Flush()
		}
	}
}

// ttsVoice is one voice id returned by an OpenAI-compatible TTS server.
type ttsVoice struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

func defaultTTSVoices() []ttsVoice {
	ids := []string{"alloy", "ash", "ballad", "coral", "echo", "fable", "nova", "onyx", "sage", "shimmer", "verse",
		"zh-CN-XiaoxiaoNeural", "zh-CN-XiaoyiNeural", "zh-CN-YunxiNeural", "zh-CN-YunjianNeural", "zh-CN-YunyangNeural",
		"en-US-AriaNeural", "en-US-GuyNeural", "en-US-JennyNeural"}
	out := make([]ttsVoice, 0, len(ids))
	for _, id := range ids {
		out = append(out, ttsVoice{ID: id})
	}
	return out
}

// parseVoiceList accepts the handful of shapes OpenAI-compatible TTS servers use:
// ["alloy"], {"voices":[{"id":..,"name":..}]}, {"data":[{"id":..}]}, flat objects.
func parseVoiceList(data []byte) []ttsVoice {
	var root any
	if json.Unmarshal(data, &root) != nil {
		return nil
	}
	var items []any
	switch v := root.(type) {
	case []any:
		items = v
	case map[string]any:
		for _, key := range []string{"voices", "data", "items", "models"} {
			if arr, ok := v[key].([]any); ok {
				items = arr
				break
			}
		}
	}
	out := []ttsVoice{}
	seen := map[string]bool{}
	for _, it := range items {
		switch x := it.(type) {
		case string:
			if x != "" && !seen[x] {
				out = append(out, ttsVoice{ID: x})
				seen[x] = true
			}
		case map[string]any:
			id, _ := x["id"].(string)
			name, _ := x["name"].(string)
			if id == "" {
				id, _ = x["voice"].(string)
			}
			if id == "" {
				id = name
			}
			if id == "" || seen[id] {
				continue
			}
			if name == "" || name == id {
				name = ""
			}
			out = append(out, ttsVoice{ID: id, Name: name})
			seen[id] = true
		}
	}
	return out
}

// fetchTTSVoices asks the configured provider for its voice list.
func fetchTTSVoices(ctx context.Context) ([]ttsVoice, error) {
	cfg := ttsConfig()
	if cfg.Provider == "off" {
		return nil, fmt.Errorf("tts is off")
	}
	client := &http.Client{Timeout: 15 * time.Second}
	paths := []string{"/audio/voices", "/voices", "/audio/speech/voices"}
	var lastErr error
	for _, p := range paths {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.Base+p, nil)
		if err != nil {
			continue
		}
		if cfg.Key != "" {
			req.Header.Set("Authorization", "Bearer "+cfg.Key)
		}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		resp.Body.Close()
		if resp.StatusCode >= 400 {
			lastErr = fmt.Errorf("%s: HTTP %d", p, resp.StatusCode)
			continue
		}
		if vs := parseVoiceList(body); len(vs) > 0 {
			return vs, nil
		}
		lastErr = fmt.Errorf("%s: no voices in response", p)
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no voice list endpoint found")
	}
	return nil, lastErr
}

// handleTTSVoices lists the provider's voices so the UI is not limited to a
// hardcoded list. Admins only (it uses the stored key).
func handleTTSVoices(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		writeJSONStatus(w, http.StatusForbidden, map[string]any{"error": "admin only"})
		return
	}
	vs, err := fetchTTSVoices(r.Context())
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": err.Error(), "voices": defaultTTSVoices()})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "voices": vs, "count": len(vs)})
}
