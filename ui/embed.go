package ui

import (
	"embed"
	"io/fs"
)

//go:embed all:build/client
var distFS embed.FS

// Dist returns an fs.FS rooted at "build/client" containing the embedded SPA assets.
func Dist() (fs.FS, error) {
	return fs.Sub(distFS, "build/client")
}
