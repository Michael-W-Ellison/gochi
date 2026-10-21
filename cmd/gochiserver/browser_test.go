package main

import (
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestBrowserSmoke plays the page in a headless browser (test/e2e/smoke.js)
// against a game with a fresh data folder. It needs node and Playwright, so
// it runs only with GOCHI_E2E=1 (make e2e); NODE_PATH tells node where
// Playwright is.
func TestBrowserSmoke(t *testing.T) {
	if os.Getenv("GOCHI_E2E") == "" {
		t.Skip("set GOCHI_E2E=1 to play the page in a browser")
	}
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatal("node isn't installed")
	}
	h := newHarness(t)
	server := httptest.NewServer(h.handler)
	defer server.Close()
	go h.server.updateLoop() // Time passes as in play
	script, _ := filepath.Abs(filepath.Join("..", "..", "test", "e2e", "smoke.js"))
	out, err := exec.Command(node, script, server.URL).CombinedOutput()
	t.Logf("%s", out)
	if err != nil {
		t.Fatalf("The page had problems: %v", err)
	}
}
