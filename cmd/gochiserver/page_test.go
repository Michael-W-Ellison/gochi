package main

import (
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// The page's scripts share one global scope and call the game by path, so
// mistakes there don't show until a player opens the page: a name
// declared in two scripts stops the second from loading (one once broke
// the collection panel), and a misspelt route or element ID fails only
// when used. These tests read the page as the game serves it, and its
// scripts in the order it loads them.

// pageScript is a script the page runs
type pageScript struct {
	name   string // The file, or "inline"
	source string
}

// pageAndScripts returns the page's HTML and its scripts in load order
func pageAndScripts(t *testing.T) (string, []pageScript) {
	t.Helper()
	h := newHarness(t)
	rec := h.do(http.MethodGet, "/", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("The page: %d", rec.Code)
	}
	html := rec.Body.String()
	tag := regexp.MustCompile(`(?s)<script(?: src="([^"?]+)[^"]*")?>(.*?)</script>`)
	var scripts []pageScript
	for _, m := range tag.FindAllStringSubmatch(html, -1) {
		if m[1] == "" {
			scripts = append(scripts, pageScript{"inline", m[2]})
			continue
		}
		path := strings.TrimPrefix(m[1], "/static/")
		data, err := fs.ReadFile(h.server.static, path)
		if err != nil {
			t.Fatalf("The page loads %s, which can't be read: %v", m[1], err)
		}
		scripts = append(scripts, pageScript{filepath.Base(path), string(data)})
	}
	if len(scripts) < 5 {
		t.Fatalf("Only %d scripts found", len(scripts))
	}
	return html, scripts
}

// topLevelNames returns the names a script declares at its top level: at
// its smallest indentation
func topLevelNames(source string) []string {
	decl := regexp.MustCompile(`^(\s*)(?:async\s+)?(?:function\*?|const|let|var|class)\s+([A-Za-z_$][\w$]*)`)
	indent := -1
	for _, line := range strings.Split(source, "\n") {
		if m := decl.FindStringSubmatch(line); m != nil && (indent < 0 || len(m[1]) < indent) {
			indent = len(m[1])
		}
	}
	var names []string
	for _, line := range strings.Split(source, "\n") {
		if m := decl.FindStringSubmatch(line); m != nil && len(m[1]) == indent {
			names = append(names, m[2])
		}
	}
	return names
}

func TestPageScriptsDontShareNames(t *testing.T) {
	_, scripts := pageAndScripts(t)
	declared := map[string]string{}
	for _, s := range scripts {
		seen := map[string]bool{}
		for _, name := range topLevelNames(s.source) {
			if seen[name] {
				t.Errorf("%s declares %s twice", s.name, name)
			}
			seen[name] = true
			if other, ok := declared[name]; ok {
				t.Errorf("%s and %s both declare %s: the page stops loading the second", other, s.name, name)
			}
			declared[name] = s.name
		}
	}
}

// pageNames returns every top-level name the page's scripts declare
func pageNames(scripts []pageScript) map[string]bool {
	names := map[string]bool{}
	for _, s := range scripts {
		for _, n := range topLevelNames(s.source) {
			names[n] = true
		}
	}
	return names
}

func TestPageHandlersExist(t *testing.T) {
	html, scripts := pageAndScripts(t)
	names := pageNames(scripts)
	for _, m := range regexp.MustCompile(`on(?:click|change|input|submit)="([A-Za-z_$][\w$]*)\(`).FindAllStringSubmatch(html, -1) {
		if !names[m[1]] {
			t.Errorf("The page calls %s(), which no script declares", m[1])
		}
	}

	// Functions help.js adds to must be declared before it loads
	before := map[string]bool{}
	for _, s := range scripts {
		if s.name == "help.js" {
			for _, m := range regexp.MustCompile(`after\('([\w$]+)'`).FindAllStringSubmatch(s.source, -1) {
				if !before[m[1]] {
					t.Errorf("help.js adds to %s, which isn't declared before help.js loads", m[1])
				}
			}
		}
		for _, n := range topLevelNames(s.source) {
			before[n] = true
		}
	}
}

func TestPageUsesOnlyGameRoutes(t *testing.T) {
	html, scripts := pageAndScripts(t)
	h := newHarness(t)
	routes := map[string]bool{}
	for _, r := range h.server.routes() {
		routes[r.pattern] = true
	}
	all := html
	for _, s := range scripts {
		all += "\n" + s.source
	}
	used := map[string]bool{}
	for _, m := range regexp.MustCompile("['\"`](/api/[a-z0-9_/]+)").FindAllStringSubmatch(all, -1) {
		used[strings.TrimSuffix(m[1], "/")] = true
	}
	if len(used) < 40 {
		t.Fatalf("Only %d routes found in the page", len(used))
	}
	for path := range used {
		if !routes[path] && !strings.HasPrefix(path, "/api/v1") {
			t.Errorf("The page calls %s, which isn't a route", path)
		}
	}
}

func TestPageElementsExist(t *testing.T) {
	html, scripts := pageAndScripts(t)
	ids := map[string]bool{}
	for _, m := range regexp.MustCompile(`\sid="([\w-]+)"`).FindAllStringSubmatch(html, -1) {
		ids[m[1]] = true
	}
	all := ""
	for _, s := range scripts {
		all += "\n" + s.source
	}
	made := regexp.MustCompile(`\.id\s*=\s*'([\w-]+)'|\sid=["']([\w-]+)["']`) // By a script, or in HTML it writes
	for _, m := range made.FindAllStringSubmatch(all, -1) {
		ids[m[1]+m[2]] = true
	}
	var missing []string
	for _, m := range regexp.MustCompile(`getElementById\('([\w-]+)'\)`).FindAllStringSubmatch(all, -1) {
		if !ids[m[1]] && !contains(missing, m[1]) {
			missing = append(missing, m[1])
		}
	}
	sort.Strings(missing)
	for _, id := range missing {
		t.Errorf("A script looks up #%s, which the page doesn't have", id)
	}
}

func TestPageScriptsParse(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node isn't installed")
	}
	_, scripts := pageAndScripts(t)
	dir := t.TempDir()
	for i, s := range scripts {
		path := filepath.Join(dir, strings.TrimSuffix(s.name, ".js")+"_"+string(rune('a'+i))+".js")
		if err := os.WriteFile(path, []byte(s.source), 0o600); err != nil {
			t.Fatal(err)
		}
		if out, err := exec.Command(node, "--check", path).CombinedOutput(); err != nil {
			t.Errorf("%s doesn't parse: %s", s.name, out)
		}
	}
}
