package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestStaticFilesAreCacheable(t *testing.T) {
	files := fstest.MapFS{"app.js": {Data: []byte("one")}}
	version, err := staticVersion(files)
	if err != nil || len(version) != 12 {
		t.Fatalf("Version %q, %v", version, err)
	}
	s := newTestServer(t)
	s.staticVersion = version

	for _, builtIn := range []bool{false, true} {
		handler := s.staticHandler(files, builtIn)
		get := func(path, etag string) *httptest.ResponseRecorder {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, path, nil)
			if etag != "" {
				req.Header.Set("If-None-Match", etag)
			}
			handler.ServeHTTP(rec, req)
			return rec
		}
		if rec := get("/static/app.js?v="+version, ""); rec.Code != http.StatusOK || rec.Header().Get("Cache-Control") != staticMaxAge ||
			rec.Body.String() != "one" {
			t.Errorf("Versioned file: %d %q %q", rec.Code, rec.Header().Get("Cache-Control"), rec.Body)
		}
		for _, path := range []string{"/static/app.js", "/static/app.js?v=old"} {
			if rec := get(path, ""); rec.Header().Get("Cache-Control") != "no-cache" {
				t.Errorf("%s should be revalidated: %q", path, rec.Header().Get("Cache-Control"))
			}
		}
		if rec := get("/static/", ""); rec.Code != http.StatusNotFound {
			t.Errorf("A folder listing: %d", rec.Code)
		}

		// Built-in files have no dates, so they're revalidated by version
		rec := get("/static/app.js", `"`+version+`"`)
		if builtIn && rec.Code != http.StatusNotModified {
			t.Errorf("A built-in file whose version is current: %d", rec.Code)
		}
		if !builtIn && (rec.Code != http.StatusOK || rec.Header().Get("ETag") != "") {
			t.Errorf("A file from a folder: %d %q", rec.Code, rec.Header().Get("ETag"))
		}
	}

	// Changing a file changes the version
	files["app.js"] = &fstest.MapFile{Data: []byte("two")}
	if changed, _ := staticVersion(files); changed == version {
		t.Error("The version should change with the files")
	}

	// The page asks for the current version
	rec := httptest.NewRecorder()
	s.serveDefaultPage(rec)
	if !strings.Contains(rec.Body.String(), "/static/js/ar.js?v="+version) || strings.Contains(rec.Body.String(), "{{static}}") {
		t.Error("The page should refer to versioned static files")
	}
}

// TestBuiltInStaticFiles checks that the game carries the page's files
func TestBuiltInStaticFiles(t *testing.T) {
	h := newHarness(t)
	for _, path := range []string{"/static/js/ar.js", "/static/css/ar.css"} {
		rec := h.do(http.MethodGet, path, "")
		if rec.Code != http.StatusOK || rec.Body.Len() == 0 {
			t.Errorf("%s: %d, %d bytes", path, rec.Code, rec.Body.Len())
		}
	}
}
