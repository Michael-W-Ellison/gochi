//go:build tools

// Package tools pins the versions of the development tools the Makefile
// runs (lint and check). They're in a module of their own, so the game's
// go.mod lists only what the game uses; make tools builds them into
// bin/tools at the versions in tools/go.mod.
package tools

import (
	_ "github.com/golangci/golangci-lint/v2/cmd/golangci-lint"
	_ "golang.org/x/tools/cmd/deadcode"
	_ "golang.org/x/tools/cmd/goimports"
	_ "honnef.co/go/tools/cmd/staticcheck"
)
