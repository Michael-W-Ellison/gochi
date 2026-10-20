package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStaticFilesAreCacheable(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "app.js"), []byte("one"), 0o644); err != nil {
		t.Fatal(err)
	}
	version, err := staticVersion(dir)
	if err != nil || len(version) != 12 {
		t.Fatalf("Version %q, %v", version, err)
	}
	s := newTestServer(t)
	s.staticVersion = version
	handler := s.staticHandler(dir)

	get := func(path string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		return rec
	}
	if rec := get("/static/app.js?v=" + version); rec.Code != http.StatusOK || rec.Header().Get("Cache-Control") != staticMaxAge ||
		rec.Body.String() != "one" {
		t.Errorf("Versioned file: %d %q %q", rec.Code, rec.Header().Get("Cache-Control"), rec.Body)
	}
	for _, path := range []string{"/static/app.js", "/static/app.js?v=old"} {
		if rec := get(path); rec.Header().Get("Cache-Control") != "no-cache" {
			t.Errorf("%s should be revalidated: %q", path, rec.Header().Get("Cache-Control"))
		}
	}

	// Changing a file changes the version
	os.WriteFile(filepath.Join(dir, "app.js"), []byte("two"), 0o644)
	if changed, _ := staticVersion(dir); changed == version {
		t.Error("The version should change with the files")
	}

	// The page asks for the current version
	rec := httptest.NewRecorder()
	s.serveDefaultPage(rec)
	if !strings.Contains(rec.Body.String(), "/static/js/ar.js?v="+version) || strings.Contains(rec.Body.String(), "{{static}}") {
		t.Error("The page should refer to versioned static files")
	}
}
