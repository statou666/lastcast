package main

import (
	"os"
	"testing"
)

func TestParseFeedSample(t *testing.T) {
	f, err := os.Open("testfeed.xml")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	meta, items, err := parseFeed(f)
	t.Logf("title=%q desc=%q cover=%q items=%d err=%v", meta.Title, meta.Description, meta.CoverURL, len(items), err)
	for i, it := range items {
		if i < 3 {
			t.Logf("item %d: title=%q guid=%q enclosure=%q type=%q dur=%d", i, it.Title, it.Guid, it.EnclosureURL, it.EnclosureType, it.DurationSec)
		}
	}
	if len(items) == 0 {
		t.Fail()
	}
}
