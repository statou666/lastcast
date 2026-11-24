package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

// Generic full-text extraction: no site-specific rules. It picks the longest
// <article>/<main> (falling back to <body>), drops chrome, converts blocks to
// text and collects <img>/og:image URLs. Good enough for most blogs and RSS
// items that only ship a summary; the feed's own summary remains the fallback.

var (
	reArticle    = regexp.MustCompile(`(?is)<article\b[^>]*>(.*?)</article>`)
	reMain       = regexp.MustCompile(`(?is)<main\b[^>]*>(.*?)</main>`)
	reBody       = regexp.MustCompile(`(?is)<body\b[^>]*>(.*?)</body>`)
	reDropBlocks = func() []*regexp.Regexp {
		tags := []string{"script", "style", "noscript", "nav", "header", "footer", "aside", "form", "svg", "iframe", "button", "select", "textarea"}
		out := make([]*regexp.Regexp, 0, len(tags))
		for _, t := range tags {
			out = append(out, regexp.MustCompile(`(?is)<`+t+`\b[^>]*>.*?</`+t+`>`))
		}
		return out
	}()
	reImgSrc  = regexp.MustCompile(`(?is)<img\b[^>]*?(?:src|data-src|data-original)\s*=\s*["']([^"']+)["']`)
	reMetaImg = regexp.MustCompile(`(?is)<meta[^>]+(?:property|name)\s*=\s*["'](?:og:image|twitter:image)["'][^>]*>`)
	reContent = regexp.MustCompile(`(?is)content\s*=\s*["']([^"']+)["']`)
	reBr      = regexp.MustCompile(`(?i)<br\s*/?>`)
	reBlock   = regexp.MustCompile(`(?i)</(p|div|section|article|h[1-6]|li|tr|blockquote|pre|figure)>`)
	reTag     = regexp.MustCompile(`(?s)<[^>]+>`)
	reSpace   = regexp.MustCompile(`[ \t\f\v]+`)
)

var fullTextClient = &http.Client{Timeout: 35 * time.Second, Transport: newTransport()}

func absoluteURL(base, ref string) string {
	ref = strings.TrimSpace(html.UnescapeString(ref))
	if ref == "" || strings.HasPrefix(ref, "data:") {
		return ""
	}
	u, err := url.Parse(ref)
	if err != nil {
		return ""
	}
	if u.IsAbs() {
		if u.Scheme == "http" || u.Scheme == "https" {
			return u.String()
		}
		return ""
	}
	if base == "" {
		return ""
	}
	b, err := url.Parse(base)
	if err != nil {
		return ""
	}
	return b.ResolveReference(u).String()
}

func dedupeImages(in []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, s := range in {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
		if len(out) >= 16 {
			break
		}
	}
	return out
}

// extractImages finds images in an HTML fragment (and meta images in a full doc).
func extractImages(fragment, base, doc string) []string {
	var out []string
	for _, m := range reImgSrc.FindAllStringSubmatch(fragment, -1) {
		if u := absoluteURL(base, m[1]); u != "" {
			out = append(out, u)
		}
	}
	for _, m := range reMetaImg.FindAllString(doc, -1) {
		if cm := reContent.FindStringSubmatch(m); cm != nil {
			if u := absoluteURL(base, cm[1]); u != "" {
				out = append(out, u)
			}
		}
	}
	return dedupeImages(out)
}

func htmlToText(fragment string) string {
	for _, re := range reDropBlocks {
		fragment = re.ReplaceAllString(fragment, " ")
	}
	fragment = reBr.ReplaceAllString(fragment, "\n")
	fragment = reBlock.ReplaceAllString(fragment, "\n")
	fragment = reTag.ReplaceAllString(fragment, " ")
	fragment = html.UnescapeString(fragment)
	var b strings.Builder
	for _, line := range strings.Split(fragment, "\n") {
		line = strings.TrimSpace(reSpace.ReplaceAllString(line, " "))
		if line == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString(line)
	}
	return strings.TrimSpace(b.String())
}

// fetchArticle returns the main text and image URLs of a page.
func fetchArticle(ctx context.Context, page string) (string, []string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, page, nil)
	if err != nil {
		return "", nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("Referer", refererFor(page))
	resp, err := fullTextClient.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return "", nil, err
	}
	doc := string(raw)
	// Longest article wins; then main; then body; then the whole document.
	container := ""
	for _, m := range reArticle.FindAllStringSubmatch(doc, -1) {
		if len(m[1]) > len(container) {
			container = m[1]
		}
	}
	if len(container) < 200 {
		if m := reMain.FindStringSubmatch(doc); m != nil && len(m[1]) > len(container) {
			container = m[1]
		}
	}
	if len(container) < 200 {
		if m := reBody.FindStringSubmatch(doc); m != nil && len(m[1]) > len(container) {
			container = m[1]
		}
	}
	if container == "" {
		container = doc
	}
	images := extractImages(container, page, doc)
	text := htmlToText(container)
	if len([]rune(text)) < 200 {
		return "", images, fmt.Errorf("article text too short (%d chars)", len([]rune(text)))
	}
	return text, images, nil
}

// fullTextImages parses the stored images JSON.
func decodeImages(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	var out []string
	if json.Unmarshal([]byte(s), &out) != nil {
		return nil
	}
	return out
}

func encodeImages(v []string) string {
	if len(v) == 0 {
		return ""
	}
	b, _ := json.Marshal(v)
	return string(b)
}

// storeEpisodeFullText fetches and saves the full article for one episode.
func storeEpisodeFullText(ctx context.Context, id int64, force bool) error {
	var link, guid, title, desc, body, cover, imagesJSON, fullText string
	if err := store.db.QueryRow(`SELECT COALESCE(link,''), guid, COALESCE(title,''), COALESCE(description,''),
		COALESCE(body,''), COALESCE(cover_url,''), COALESCE(images,''), COALESCE(full_text,'') FROM episodes WHERE id=?`, id).
		Scan(&link, &guid, &title, &desc, &body, &cover, &imagesJSON, &fullText); err != nil {
		return err
	}
	if strings.TrimSpace(fullText) != "" && !force {
		return nil
	}
	page := ""
	for _, cand := range []string{link, guid} {
		if strings.HasPrefix(cand, "http://") || strings.HasPrefix(cand, "https://") {
			page = cand
			break
		}
	}
	if page == "" {
		return fmt.Errorf("no article url")
	}
	text, imgs, err := fetchArticle(ctx, page)
	if err != nil {
		return err
	}
	all := dedupeImages(append(imgs, decodeImages(imagesJSON)...))
	if cover == "" && len(all) > 0 {
		cover = all[0]
	}
	_, err = store.db.Exec(`UPDATE episodes SET full_text=?, images=?, cover_url=CASE WHEN COALESCE(cover_url,'')='' THEN ? ELSE cover_url END,
		segments=0, segments_done=0 WHERE id=?`, text, encodeImages(all), cover, id)
	if err == nil {
		_ = os.RemoveAll(segmentDirFor(id))
	}
	if err == nil {
		store.logEvent("episode", itoa(id), "fulltext")
	}
	return err
}

// Background queue: enabling "fetch full text" per feed must not slow the feed
// fetch itself, and must never hammer a site with unbounded parallelism.
var ftQueue = make(chan int64, 512)

func enqueueFullText(id int64) {
	select {
	case ftQueue <- id:
	default:
	}
}

func fullTextWorkers(ctx context.Context) {
	for i := 0; i < 2; i++ {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case id := <-ftQueue:
					ectx, cancel := context.WithTimeout(ctx, 60*time.Second)
					err := storeEpisodeFullText(ectx, id, false)
					cancel()
					if err != nil {
						store.logEvent("episode", itoa(id), "fulltext_error: "+err.Error())
					}
				}
			}
		}()
	}
}

// handleEpisodeFullText: POST /api/episodes/{id}/fulltext (explicit user action).
func handleEpisodeFullText(w http.ResponseWriter, r *http.Request) {
	id := int64(atoi(r.PathValue("id")))
	if id == 0 || !episodeOwned(userID(r), id) {
		writeJSON(w, map[string]any{"error": "episode not found"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()
	if err := storeEpisodeFullText(ctx, id, true); err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}
