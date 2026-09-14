package ui

import (
	"embed"
	"io/fs"
)

//go:embed all:web
var rawAssets embed.FS

// assets is the web/ subtree, rooted so paths are e.g. "index.html", "assets/app.js".
var assets fs.FS

func init() {
	sub, err := fs.Sub(rawAssets, "web")
	if err != nil {
		panic("parameters-core/ui: failed to mount web assets: " + err.Error())
	}
	assets = sub
}
