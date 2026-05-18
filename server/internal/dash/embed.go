package dash

import "embed"

// FS holds the compiled Dash PWA assets embedded at build time.
// The embed path resolves relative to the package directory.
//
//go:embed all:dist
var FS embed.FS
