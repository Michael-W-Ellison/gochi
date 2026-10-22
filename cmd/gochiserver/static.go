package main

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"net/http"
	"strings"
)

// Static files (scripts, styles, images) are cacheable: the page refers to
// them with ?v= and a hash of their contents, and such requests may be
// cached for a year, by the browser or by a proxy or CDN in front of the
// server, since a change to any file changes the hash. Other requests are
// revalidated each time.
//
// The files are built into the game (see package web), or read from a
// folder with -web-dir while working on the page.

// staticMaxAge is how long versioned static files may be cached
const staticMaxAge = "public, max-age=31536000, immutable"

// staticVersion returns a short hash of every file in files
func staticVersion(files fs.FS) (string, error) {
	hash := sha256.New()
	err := fs.WalkDir(files, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := fs.ReadFile(files, path)
		if err != nil {
			return err
		}
		hash.Write([]byte(path + "\x00")) // A hash's Write never fails
		hash.Write(data)
		return nil
	})
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil))[:12], nil
}

// staticHandler serves files under /static/. Built-in files have no
// dates to revalidate by, so they're revalidated by version instead
// (builtIn); files read from a folder may change while the game runs.
func (s *Server) staticHandler(files fs.FS, builtIn bool) http.Handler {
	server := http.StripPrefix("/static/", http.FileServer(http.FS(files)))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/") {
			http.NotFound(w, r) // No folder listings
			return
		}
		if v := r.URL.Query().Get("v"); v != "" && v == s.staticVersion {
			w.Header().Set("Cache-Control", staticMaxAge)
		} else {
			w.Header().Set("Cache-Control", "no-cache") // Revalidate each time
		}
		if builtIn {
			w.Header().Set("ETag", `"`+s.staticVersion+`"`)
		}
		server.ServeHTTP(w, r)
	})
}
