package main

import (
	"encoding/xml"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type FeedMeta struct {
	Kind        string // audio | video | text (majority of items)
	Title       string
	Description string
	CoverURL    string
	Author      string
}

type FeedItem struct {
	Kind          string // audio | video | text
	Title         string
	Description   string
	Guid          string
	Link          string
	EnclosureURL  string
	EnclosureType string
	VideoURL      string
	Body          string
	Author        string
	CoverURL      string
	PublishedAt   int64
	DurationSec   int64
}

// readText consumes tokens until text content is found (handles nested markup).
func readText(dec *xml.Decoder) string {
	for {
		t, err := dec.Token()
		if err != nil {
			return ""
		}
		switch x := t.(type) {
		case xml.CharData:
			return strings.TrimSpace(string(x))
		case xml.StartElement:
			// gather chars inside this element until its end
			var out string
			for {
				t2, err := dec.Token()
				if err != nil {
					return out
				}
				if cd, ok := t2.(xml.CharData); ok {
					out += string(cd)
				}
				if e, ok := t2.(xml.EndElement); ok && e.Name.Local == x.Name.Local {
					return strings.TrimSpace(out)
				}
			}
		}
	}
}

func attr(a []xml.Attr, name string) string {
	for _, x := range a {
		if x.Name.Local == name {
			return x.Value
		}
	}
	return ""
}

// parseFeed handles RSS 2.0 and Atom with a token-level walk (namespaces agnostic).
func parseFeed(r io.Reader) (FeedMeta, []FeedItem, error) {
	dec := xml.NewDecoder(r)
	var meta FeedMeta
	var items []FeedItem
	var cur FeedItem
	tally := map[string]int{}
	inChannel, inItem, inImage := false, false, false

	for {
		t, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return meta, items, err
		}
		switch se := t.(type) {
		case xml.StartElement:
			name := se.Name.Local
			switch {
			case name == "channel":
				inChannel = true
			case name == "item" || name == "entry":
				inItem = true
				cur = FeedItem{}
			case inImage && name == "url":
				if meta.CoverURL == "" {
					meta.CoverURL = readText(dec)
				}
			case inChannel && !inItem && (name == "creator" || name == "author"):
				if meta.Author == "" {
					meta.Author = readText(dec)
				}
			case inItem && (name == "creator" || name == "author"):
				if cur.Author == "" {
					cur.Author = readText(dec)
				}
			case inItem && name == "thumbnail":
				if u := attr(se.Attr, "url"); u != "" && cur.CoverURL == "" {
					cur.CoverURL = u
				}
			case inChannel && !inItem && (name == "title" || name == "description"):
				v := readText(dec)
				if name == "title" && meta.Title == "" {
					meta.Title = v
				} else if name == "description" && meta.Description == "" {
					meta.Description = v
				}
			case name == "image" && inChannel:
				if href := attr(se.Attr, "href"); href != "" && meta.CoverURL == "" {
					meta.CoverURL = href
				}
				inImage = true
			case name == "link" && inItem:
				rel := attr(se.Attr, "rel")
				href := attr(se.Attr, "href")
				switch {
				case rel == "enclosure":
					cur.EnclosureURL, cur.EnclosureType = href, attr(se.Attr, "type")
				case href != "":
					if cur.Link == "" {
						cur.Link = href
					}
				default:
					if v := readText(dec); cur.Link == "" && v != "" {
						cur.Link = v
					}
				}
			case name == "enclosure" && inItem:
				// RSS enclosure uses the `url` attribute; Atom link rel=enclosure uses href.
				cur.EnclosureURL = attr(se.Attr, "url")
				if cur.EnclosureURL == "" {
					cur.EnclosureURL = attr(se.Attr, "href")
				}
				cur.EnclosureType = attr(se.Attr, "type")
			case inItem && name == "content":
				// media:content carries a url attribute; content:encoded / Atom content carries text.
				// Match on the local name so the namespace prefix does not matter.
				if u := attr(se.Attr, "url"); u != "" {
					if d := attr(se.Attr, "duration"); d != "" {
						if n, err := strconv.ParseInt(d, 10, 64); err == nil && n > 0 {
							cur.DurationSec = n
						}
					}
					typ, medium := strings.ToLower(attr(se.Attr, "type")), strings.ToLower(attr(se.Attr, "medium"))
					if strings.Contains(typ, "video") || medium == "video" {
						if cur.VideoURL == "" {
							cur.VideoURL = u
						}
					} else if cur.EnclosureURL == "" && (typ == "" || strings.Contains(typ, "audio")) {
						cur.EnclosureURL, cur.EnclosureType = u, typ
					}
				} else if v := readText(dec); len(v) > len(cur.Body) {
					cur.Body = v
				}
			case inItem && name == "encoded":
				// content:encoded — the local name is "encoded" under the content namespace.
				if v := readText(dec); len(v) > len(cur.Body) {
					cur.Body = v
				}
			case inItem && (name == "title" || name == "description" || name == "summary" || name == "content" ||
				name == "guid" || name == "id" || name == "pubDate" || name == "published" ||
				(name == "duration" && se.Name.Space != "")):
				v := readText(dec)
				switch name {
				case "title":
					if cur.Title == "" {
						cur.Title = v
					}
				case "description", "summary":
					if cur.Description == "" {
						cur.Description = v
					}
				case "guid", "id":
					cur.Guid = v
				case "pubDate", "published":
					if ts, ok := parseTime(v); ok {
						cur.PublishedAt = ts
					}
				case "duration":
					cur.DurationSec = parseDuration(v)
				case "content":
					if len(v) > len(cur.Body) {
						cur.Body = v
					}
				}
			}
		case xml.EndElement:
			name := se.Name.Local
			if name == "channel" {
				inChannel = false
			}
			if name == "item" || name == "entry" {
				if inItem {
					if cur.Guid == "" {
						cur.Guid = cur.EnclosureURL
					}
					if cur.CoverURL == "" {
						cur.CoverURL = firstImgSrc(cur.Body + " " + cur.Description)
					}
					cur.Kind = classifyItem(cur)
					if cur.Title != "" && (cur.EnclosureURL != "" || cur.Body != "" || cur.VideoURL != "" || cur.Description != "") {
						items = append(items, cur)
						tally[cur.Kind]++
					}
					inItem = false
				}
			}
			if name == "image" {
				inImage = false
			}
		}
	}
	meta.Kind = topKind(tally)
	return meta, items, nil
}

// videoPageRe spots the video platforms/containers that RSSHub-style feeds only
// link to (a player page, an iframe, or a direct media URL in the item body).
var videoPageRe = regexp.MustCompile(`(?i)(bilibili\.com/(video|bangumi)/|player\.bilibili\.com|blackboard/newplayer|youtube\.com/(watch|embed|shorts)|youtu\.be/|youtube-nocookie\.com/|vimeo\.com/(\d|video/|player)|dailymotion\.com/video|\.mp4\b|\.m3u8\b|\.webm\b|\.mkv\b)`)

var imgSrcRe = regexp.MustCompile(`(?i)<img[^>]+src=["']([^"']+)["']`)

func firstImgSrc(s string) string {
	if m := imgSrcRe.FindStringSubmatch(s); m != nil {
		return strings.ReplaceAll(m[1], "&amp;", "&")
	}
	return ""
}

func looksLikeVideo(s string) bool { return s != "" && videoPageRe.MatchString(s) }

func classifyItem(it FeedItem) string {
	t := strings.ToLower(it.EnclosureType)
	// The enclosure may not be the media itself: RSSHub bilibili links an HTML
	// player page (type text/html) and the real stream is resolved from the bvid.
	if strings.Contains(t, "html") || strings.Contains(t, "xml") {
		if looksLikeVideo(it.Guid+" "+it.Link+" "+it.EnclosureURL) || biliRef(it.Guid+" "+it.EnclosureURL) != "" {
			return "video"
		}
	}
	switch {
	case strings.Contains(t, "video"), it.EnclosureURL == "" && it.VideoURL != "":
		return "video"
	case looksLikeVideo(it.Guid + " " + it.Link + " " + it.EnclosureURL), biliRef(it.Guid+" "+it.Description+" "+it.Body) != "":
		return "video"
	case it.EnclosureURL == "" && (it.Body != "" || it.Description != ""):
		return "text"
	default:
		return "audio"
	}
}

func topKind(tally map[string]int) string {
	best, n := "audio", 0
	for k, v := range tally {
		if v > n {
			best, n = k, v
		}
	}
	return best
}

func parseTime(v string) (int64, bool) {
	for _, layout := range []string{time.RFC3339, time.RFC1123Z, time.RFC1123} {
		if t, err := time.Parse(layout, v); err == nil {
			return t.Unix(), true
		}
	}
	return 0, false
}

func parseDuration(v string) int64 {
	v = strings.TrimSpace(v)
	if n, err := strconv.ParseInt(v, 10, 64); err == nil {
		return n
	}
	var parts []int64
	for _, p := range strings.Split(v, ":") {
		n, err := strconv.ParseInt(p, 10, 64)
		if err != nil {
			return 0
		}
		parts = append(parts, n)
	}
	switch len(parts) {
	case 3:
		return parts[0]*3600 + parts[1]*60 + parts[2]
	case 2:
		return parts[0]*60 + parts[1]
	case 1:
		return parts[0]
	}
	return 0
}

// fetchFeed performs a conditional GET and returns parsed content.
func fetchFeed(client *http.Client, url, etag string) (meta FeedMeta, items []FeedItem, newEtag string, notModified bool, err error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return meta, items, "", false, err
	}
	req.Header.Set("User-Agent", "podcastman/1.0 (self-hosted podcast manager)")
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	resp, err := client.Do(req)
	if err != nil {
		return meta, items, "", false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotModified {
		return meta, items, etag, true, nil
	}
	if resp.StatusCode != http.StatusOK {
		return meta, items, "", false, &feedError{status: resp.StatusCode}
	}
	newEtag = resp.Header.Get("ETag")
	meta, items, err = parseFeed(resp.Body)
	return meta, items, newEtag, false, err
}

type feedError struct{ status int }

func (e *feedError) Error() string { return "feed fetch failed: HTTP " + strconv.Itoa(e.status) }

var imgURLRe = regexp.MustCompile(`(?i)https?://[^\s"'<>]+?\.(?:jpg|jpeg|png|webp|gif)(?:\?[^\s"'<>]*)?`)

// coverFromText finds the first usable image URL in an item body.
func coverFromText(s string) string {
	if u := firstImgSrc(s); u != "" {
		return u
	}
	return imgURLRe.FindString(s)
}
