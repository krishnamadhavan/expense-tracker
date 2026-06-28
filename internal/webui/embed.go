package webui

import "embed"

// Dist holds the built SPA (web/dist) for Profile A packaging.
//
//go:embed all:dist
var Dist embed.FS
