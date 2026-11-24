package main

import (
	"bytes"
	"embed"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"time"
)

// The web UI is embedded in the binary as a safety net: a host bind-mount may be
// unreadable by the container user (uid 65532), which used to leave users on a
// bare directory listing because Go's FileServer could not open index.html.
// Files on disk win; embedded copies are used when a file cannot be opened.
//
//go:embed web
var embeddedWeb embed.FS

type diskThenEmbed struct{ dir string }

func (d diskThenEmbed) Open(name string) (fs.File, error) {
	if f, err := os.Open(filepath.Join(d.dir, filepath.FromSlash(name))); err == nil {
		return f, nil
	}
	return embeddedWeb.Open(path.Join("web", name))
}

func (d diskThenEmbed) Stat(name string) (fs.FileInfo, error) {
	if st, err := os.Stat(filepath.Join(d.dir, filepath.FromSlash(name))); err == nil {
		return st, nil
	}
	return fs.Stat(embeddedWeb, path.Join("web", name))
}

// handleIndex serves the SPA shell from disk, falling back to the embedded copy.
func handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-cache")
	if f, err := os.Open(filepath.Join(cfg.WebDir, "index.html")); err == nil {
		defer f.Close()
		if st, err := f.Stat(); err == nil {
			http.ServeContent(w, r, "index.html", st.ModTime(), f)
			return
		}
	}
	data, err := embeddedWeb.ReadFile("web/index.html")
	if err != nil {
		http.Error(w, "web UI missing and no embedded copy", http.StatusInternalServerError)
		return
	}
	http.ServeContent(w, r, "index.html", time.Time{}, bytes.NewReader(data))
}
