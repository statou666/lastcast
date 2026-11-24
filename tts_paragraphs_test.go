package main

import (
	"strings"
	"testing"
)

// Strict paragraph segmentation: paragraphs are never merged, even when short.
func TestParagraphsStaySeparate(t *testing.T) {
	text := "first short paragraph\n\nsecond short paragraph\n\nthird short paragraph"
	got := chunks(text, 200)
	if len(got) != 3 {
		t.Fatalf("got %d chunks, want 3: %#v", len(got), got)
	}
	for i, want := range []string{"first short paragraph", "second short paragraph", "third short paragraph"} {
		if got[i] != want {
			t.Fatalf("chunk %d = %q, want %q", i, got[i], want)
		}
	}
}

// A paragraph longer than max is split further, not dropped or merged.
func TestLongParagraphSplit(t *testing.T) {
	long := strings.Repeat("这是一个很长的段落。", 40) // 400 runes
	got := chunks(long, 200)
	if len(got) < 2 {
		t.Fatalf("expected long paragraph to split, got %d chunk(s)", len(got))
	}
	for i, p := range got {
		if len([]rune(p)) > 200 {
			t.Fatalf("chunk %d has %d runes (> 200)", i, len([]rune(p)))
		}
	}
	if strings.Join(got, "") != long {
		t.Fatalf("content changed after splitting")
	}
}
