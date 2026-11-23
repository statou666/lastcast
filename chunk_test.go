package main

import (
	"strings"
	"testing"
)

// chunks() clamps max to a 200-byte minimum; the test accounts for that.
func TestChunks(t *testing.T) {
	long := "一二三四五六七八九零"
	for i := 0; i < 40; i++ {
		long += "一二三四五六七八九零"
	}
	short := "para one is about sixty characters long and should merge"

	cases := []struct {
		name string
		text string
		max  int
		want int
	}{
		{"single long chinese paragraph", long, 120, 7},
		{"short paragraphs merge", strings.Join([]string{short, short}, "\n"), 200, 1},
		{"four short paragraphs", strings.Join([]string{short, short, short, short}, "\n"), 200, 2},
	}
	for _, c := range cases {
		effMax := c.max
		if effMax < 200 {
			effMax = 200
		}
		got := chunks(c.text, c.max)
		if len(got) != c.want {
			t.Fatalf("%s: got %d chunks, want %d", c.name, len(got), c.want)
		}
		for i, part := range got {
			if part == "" {
				t.Fatalf("%s: empty chunk at %d", c.name, i)
			}
			if len(part) > effMax {
				t.Fatalf("%s: chunk %d is %d bytes, exceeds effective max %d", c.name, i, len(part), effMax)
			}
		}
		want := 0
		for _, p := range strings.Split(c.text, "\n") {
			if p = strings.TrimSpace(p); p != "" {
				want += len(p)
			}
		}
		gotLen := 0
		for _, part := range got {
			gotLen += len(strings.ReplaceAll(part, "\n", ""))
		}
		if gotLen != want {
			t.Fatalf("%s: content lost (%d vs %d)", c.name, gotLen, want)
		}
	}
}
