package main

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// Reads the progressive-playback fixture and reports how many requests the
// chunker produces from the ITEM description (summary-only feeds like sspai
// have no content:encoded, so the description is the only text available).
func TestFixtureChunkCount(t *testing.T) {
	b, err := os.ReadFile("fixtures/segfixture.xml")
	if err != nil {
		t.Skip("fixture missing")
	}
	s := strings.Split(string(b), "<item>")[1]
	i := strings.Index(s, "<description>")
	if i < 0 {
		t.Fatal("no item description in fixture")
	}
	desc := strings.Split(s[i+len("<description>"):], "</description>")[0]
	text := plainText(desc)
	fmt.Printf("fixture item: %d paragraphs, %d chars -> chunks(120)=%d chunks(3000)=%d\n",
		len(strings.Split(text, "\n")), len(text), len(chunks(text, 120)), len(chunks(text, 3000)))
	if len(chunks(text, 120)) < 2 {
		t.Errorf("expected a multi-chunk fixture, got %d", len(chunks(text, 120)))
	}
}
