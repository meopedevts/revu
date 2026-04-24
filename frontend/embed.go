// Package frontend exposes the built React assets (frontend/dist) as an
// embed.FS so Wails can serve them without shelling out to vite at runtime.
// In `wails dev` the dev server is used instead (configured in wails.json).
package frontend

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

// AssetsFS returns the embedded frontend rooted at dist/ — Wails'
// assetserver expects the root to contain index.html.
func AssetsFS() fs.FS {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		// fs.Sub only fails on invalid paths; "dist" is static and valid.
		panic(err)
	}
	return sub
}
