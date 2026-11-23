package main

import (
	"os"
	"testing"
)

func TestKindDetection(t *testing.T) {
	for _, f := range []string{"fixtures/text.rss.xml", "fixtures/video.rss.xml"} {
		r, err := os.Open(f)
		if err != nil {
			t.Fatal(err)
		}
		meta, items, err := parseFeed(r)
		r.Close()
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("feed %s meta.Kind=%q items=%d", f, meta.Kind, len(items))
		for _, it := range items {
			t.Logf("  item kind=%q body=%d video=%q", it.Kind, len(it.Body), it.VideoURL)
		}
	}
}

func TestClassifyItem(t *testing.T) {
	cases := []struct {
		it   FeedItem
		want string
	}{
		{FeedItem{Body: "hello"}, "text"},
		{FeedItem{EnclosureURL: "u", EnclosureType: "audio/mpeg"}, "audio"},
		{FeedItem{EnclosureURL: "u", EnclosureType: "video/mp4"}, "video"},
		{FeedItem{VideoURL: "u"}, "video"},
	}
	for _, c := range cases {
		if got := classifyItem(c.it); got != c.want {
			t.Errorf("classifyItem(%+v) = %q, want %q", c.it, got, c.want)
		}
	}
}
