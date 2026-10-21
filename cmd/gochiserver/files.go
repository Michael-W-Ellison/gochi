package main

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Settings files that can't be read (cut short, corrupted, or replaced by
// something else) are set aside, renamed, so the game can start without
// them and nothing is lost; temporary files left by a crash are removed.

// loadOrSetAside loads a settings file with load. If it can't be read, it
// is renamed to <name>.damaged-<time> and load is tried again, which
// starts afresh.
func loadOrSetAside[T any](what, path string, load func() (T, error)) T {
	value, err := load()
	if err == nil {
		return value
	}
	aside := fmt.Sprintf("%s.damaged-%s", path, time.Now().Format("20060102-150405"))
	if renameErr := os.Rename(path, aside); renameErr != nil {
		log.Fatalf("Could not read %s: %v", what, err)
	}
	log.Printf("Warning: could not read %s (%v); starting afresh. The old file is %s", what, err, aside)
	if value, err = load(); err != nil {
		log.Fatalf("Could not read %s: %v", what, err)
	}
	return value
}

// tempFileAge is how old a temporary file must be to be removed at
// startup, so files another running copy of the game is writing are left
const tempFileAge = time.Hour

// removeTempFiles removes temporary files (*.tmp and *.tmp-*) left in dir
// and its folders by a crash or a full disk
func removeTempFiles(dir string) {
	_ = filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".tmp") && !strings.Contains(name, ".tmp-") {
			return nil
		}
		if info, err := entry.Info(); err == nil && time.Since(info.ModTime()) > tempFileAge {
			if os.Remove(path) == nil {
				log.Printf("Removed %s, left by an earlier crash", path)
			}
		}
		return nil
	})
}
