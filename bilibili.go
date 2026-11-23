package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Universal source resolution: bilibili (ranking/search/user feeds from RSSHub
// carry no enclosure — only a player page or a bvid), direct media links inside
// RSSHub HTML, and PeerTube watch pages.
var (
	bidRe = regexp.MustCompile(`BV[0-9A-Za-z]{6,16}`)
	aidRe = regexp.MustCompile(`(?:^|[^0-9A-Za-z])av(\d{3,})`)
	// direct media link inside an item body (<img>/<a>/<iframe>/raw text)
	mediaLinkRe = regexp.MustCompile(`(?i)https?://[^\s"'<>()]+?\.(?:mp4|m4v|webm|m3u8|mp3|m4a|aac|ogg|opus|flac)(?:\?[^\s"'<>]*)?`)
	// PeerTube watch page: https://host/w/<uuid>
	peerPageRe = regexp.MustCompile(`(?i)^https?://([^/]+)/w/([0-9a-f-]{16,})`)
)

type biliTrack struct {
	URL    string   // preferred (stable CDN) URL
	URLs   []string // ordered fallbacks: upos → other → mcdn (P2P edge)
	Dur    int64
	Pic    string
	Author string
	Title  string
}

var (
	biliMu    sync.Mutex
	biliCache = map[string]biliTrack{}
	biliAt    = map[string]time.Time{}
)

// biliRef finds a bilibili video reference (BV id or av id) in arbitrary item text.
func biliRef(s string) string {
	if m := bidRe.FindString(s); m != "" {
		return m
	}
	if m := aidRe.FindStringSubmatch(s); m != nil {
		return "av" + m[1]
	}
	return ""
}

// biliResolve turns a bvid/av id into a browser-playable audio URL in two API
// calls (view → playurl) and caches both the answer and its metadata. Bilibili's
// DASH audio file has its ftyp+moov prefix inline (SegmentBase.initialization
// 0-836), so the URL can be proxied straight into an <audio> element — no ffmpeg.
func biliResolve(ctx context.Context, ref string) (biliTrack, error) {
	biliMu.Lock()
	if t, ok := biliCache[ref]; ok && time.Since(biliAt[ref]) < 25*time.Minute {
		biliMu.Unlock()
		return t, nil
	}
	biliMu.Unlock()

	viewURL := "https://api.bilibili.com/x/web-interface/view?bvid=" + ref
	if strings.HasPrefix(ref, "av") {
		viewURL = "https://api.bilibili.com/x/web-interface/view?aid=" + ref[2:]
	}
	data, err := biliGet(ctx, viewURL)
	if err != nil {
		return biliTrack{}, err
	}
	m, _ := data.(map[string]any)
	if m == nil {
		return biliTrack{}, fmt.Errorf("bilibili: bad view payload")
	}
	bvid, _ := m["bvid"].(string)
	cid := jsonInt(m["cid"])
	if cid == 0 {
		if pages, ok := m["pages"].([]any); ok && len(pages) > 0 {
			if p, ok := pages[0].(map[string]any); ok {
				cid = jsonInt(p["cid"])
			}
		}
	}
	if bvid == "" || cid == 0 {
		return biliTrack{}, fmt.Errorf("bilibili: no cid for %s", ref)
	}
	t := biliTrack{Dur: jsonInt(m["duration"]), Title: str(m["title"])}
	if v, ok := m["pic"].(string); ok {
		t.Pic = v
	}
	if owner, ok := m["owner"].(map[string]any); ok {
		t.Author = str(owner["name"])
	}

	play, err := biliGet(ctx, fmt.Sprintf("https://api.bilibili.com/x/player/playurl?bvid=%s&cid=%d&fnval=16&fourline=1", bvid, cid))
	if err != nil {
		return biliTrack{}, err
	}
	pm, _ := play.(map[string]any)
	dash, _ := pm["dash"].(map[string]any)
	audio, _ := dash["audio"].([]any)
	var best map[string]any
	var bestBW int64 = -1
	for _, a := range audio {
		am, _ := a.(map[string]any)
		if am == nil {
			continue
		}
		if bw := jsonInt(am["bandwidth"]); bw > bestBW {
			best, bestBW = am, bw
		}
	}
	var cands []string
	if best != nil {
		cands = append(cands, str(best["baseUrl"]), str(best["base_url"]))
		cands = append(cands, stringSlice(best["backupUrl"])...)
		cands = append(cands, stringSlice(best["backup_url"])...)
	}
	if durl, ok := pm["durl"].([]any); ok && len(durl) > 0 {
		for _, d := range durl {
			if dm, ok := d.(map[string]any); ok {
				cands = append(cands, str(dm["url"]))
			}
		}
	}
	t.URLs = orderBiliURLs(cands)
	if len(t.URLs) > 0 {
		t.URLs = verifyFirstURL(ctx, t.URLs)
		t.URL = t.URLs[0]
	}
	if t.URL == "" {
		return biliTrack{}, fmt.Errorf("bilibili: no audio stream for %s", bvid)
	}
	biliMu.Lock()
	biliCache[ref] = t
	biliAt[ref] = time.Now()
	biliMu.Unlock()
	return t, nil
}

// extractMediaURL returns the first direct media link embedded in item HTML
// (RSSHub item bodies often contain <img>/<a>/<iframe> only).
func extractMediaURL(s string) string {
	if s == "" {
		return ""
	}
	s = strings.ReplaceAll(s, "&amp;", "&")
	return mediaLinkRe.FindString(s)
}

// peerTubeFile resolves a PeerTube watch page (/w/<uuid>) to a direct file URL.
func peerTubeFile(ctx context.Context, page string) (string, int64, error) {
	m := peerPageRe.FindStringSubmatch(page)
	if m == nil {
		return "", 0, fmt.Errorf("not a peertube page")
	}
	api := "https://" + m[1] + "/api/v1/videos/" + m[2]
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, api, nil)
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("User-Agent", uaPodcastman)
	req.Header.Set("Accept", "application/json")
	resp, err := streamClient.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("peertube api HTTP %d", resp.StatusCode)
	}
	var body struct {
		Duration int64 `json:"duration"`
		Files    []struct {
			FileURL    string `json:"fileUrl"`
			Resolution struct {
				ID int `json:"id"`
			} `json:"resolution"`
		} `json:"files"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", 0, err
	}
	best := ""
	bestID := 1 << 30
	for _, f := range body.Files {
		if f.FileURL == "" || f.Resolution.ID <= 0 {
			continue // id 0 is the audio-only file; its fileUrl is empty on some instances
		}
		if f.Resolution.ID < bestID {
			best, bestID = f.FileURL, f.Resolution.ID
		}
	}
	if best == "" && len(body.Files) > 0 {
		best = body.Files[0].FileURL
	}
	if best == "" {
		return "", 0, fmt.Errorf("peertube: no file")
	}
	return best, body.Duration, nil
}

func str(v any) string {
	s, _ := v.(string)
	return s
}

func stringSlice(v any) []string {
	arr, _ := v.([]any)
	var out []string
	for _, x := range arr {
		if s := str(x); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// orderBiliURLs prefers the regular upos CDN over mcdn P2P edge nodes (the mcdn
// edges answer tiny range probes instantly but stall on a full sequential read),
// keeping the P2P URL as the last fallback.
func orderBiliURLs(in []string) []string {
	score := func(u string) int {
		i := strings.Index(u, "://")
		if i < 0 {
			return 3
		}
		host := u[i+3:]
		if j := strings.IndexByte(host, '/'); j >= 0 {
			host = host[:j]
		}
		host = strings.ToLower(host)
		switch {
		case strings.Contains(host, "upos"):
			return 0
		case strings.Contains(host, "mcdn"):
			return 1
		default:
			return 2
		}
	}
	var out []string
	seen := map[string]bool{}
	for sc := 0; sc <= 3; sc++ {
		for _, u := range in {
			if u == "" || seen[u] || score(u) != sc {
				continue
			}
			seen[u] = true
			out = append(out, u)
		}
	}
	return out
}

// verifyFirstURL moves the first URL that actually returns bytes to the front:
// bilibili lists P2P/edge hosts that accept a request and then send nothing.
func verifyFirstURL(ctx context.Context, urls []string) []string {
	good := -1
	for i, u := range urls {
		if i >= 3 {
			break
		}
		if biliURLWorks(ctx, u) {
			good = i
			break
		}
	}
	if good <= 0 {
		return urls
	}
	out := []string{urls[good]}
	for i, u := range urls {
		if i != good {
			out = append(out, u)
		}
	}
	return out
}

func biliURLWorks(ctx context.Context, u string) bool {
	ctx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return false
	}
	req.Header.Set("User-Agent", uaPodcastman)
	req.Header.Set("Referer", "https://www.bilibili.com/")
	req.Header.Set("Range", "bytes=0-1023")
	resp, err := streamClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return false
	}
	buf := make([]byte, 256)
	n, _ := io.ReadFull(resp.Body, buf)
	return n >= 64
}
