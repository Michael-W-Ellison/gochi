package main

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
)

// Static files (scripts, styles, images) are cacheable: the page refers to
// them with ?v= and a hash of their contents, and such requests may be
// cached for a year, by the browser or by a proxy or CDN in front of the
// server, since a change to any file changes the hash. Other requests are
// revalidated each time.

// staticMaxAge is how long versioned static files may be cached
const staticMaxAge = "public, max-age=31536000, immutable"

// staticVersion returns a short hash of every file under dir
func staticVersion(dir string) (string, error) {
	hash := sha256.New()
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(dir, path)
		io.WriteString(hash, filepath.ToSlash(rel)+"\x00")
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(hash, f)
		return err
	})
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil))[:12], nil
}

// staticHandler serves the static files in dir under /static/
func (s *Server) staticHandler(dir string) http.Handler {
	files := http.StripPrefix("/static/", http.FileServer(http.Dir(dir)))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if v := r.URL.Query().Get("v"); v != "" && v == s.staticVersion {
			w.Header().Set("Cache-Control", staticMaxAge)
		} else {
			w.Header().Set("Cache-Control", "no-cache") // Revalidate (by date) each time
		}
		files.ServeHTTP(w, r)
	})
}
