// Package web embeds the buildless browser UI served by simserver at /ui/.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:static
var assets embed.FS

// Assets returns the UI's file tree, rooted at static/ (so paths like
// "index.html" resolve directly, without a "static/" prefix).
func Assets() fs.FS {
	sub, err := fs.Sub(assets, "static")
	if err != nil {
		panic(err)
	}
	return sub
}
