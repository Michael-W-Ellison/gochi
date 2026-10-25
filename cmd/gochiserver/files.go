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

// reservedFileName reports whether name (without its extension) is one
// Windows keeps for a device, such as CON or COM1: a file of that name
// opens the device instead, which can hang the game waiting for input
func reservedFileName(name string) bool {
	base := strings.ToUpper(strings.SplitN(name, ".", 2)[0])
	switch base {
	case "CON", "PRN", "AUX", "NUL", "CONIN$", "CONOUT$":
		return true
	}
	return len(base) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) &&
		base[3] >= '0' && base[3] <= '9'
}

// loadOrSetAside loads a settings file with load. If it can't be read, it
// is renamed to <name>.damaged-<time> and load is tried again, which
// starts afresh.
func loadOrSetAside[T any](what, path string, load func() (T, error)) T {
	value, err := setAsideOnError(what, path, load)
	if err != nil {
		log.Fatal(err)
	}
	return value
}

// setAsideOnError is loadOrSetAside, returning an error instead of
// stopping the game
func setAsideOnError[T any](what, path string, load func() (T, error)) (T, error) {
	value, err := load()
	if err == nil {
		return value, nil
	}
	aside := fmt.Sprintf("%s.damaged-%s", path, time.Now().Format("20060102-150405"))
	if renameErr := os.Rename(path, aside); renameErr != nil {
		return value, fmt.Errorf("could not read %s: %w", what, err)
	}
	log.Printf("Warning: could not read %s (%v); starting afresh. The old file is %s", what, err, aside)
	if value, err = load(); err != nil {
		return value, fmt.Errorf("could not read %s: %w", what, err)
	}
	return value, nil
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
