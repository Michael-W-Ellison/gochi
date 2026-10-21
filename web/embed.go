// Package web holds the game page's static files (scripts, styles and
// images), built into the game so it runs as a single file.
package web

import (
	"embed"
	"io/fs"
)

//go:embed static
var files embed.FS

// Static returns the static files, as served under /static/
func Static() fs.FS {
	static, err := fs.Sub(files, "static")
	if err != nil {
		panic(err) // The folder is embedded above
	}
	return static
}
