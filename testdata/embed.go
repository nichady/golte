package testdata

import (
	"embed"
	"io/fs"
)

//go:embed dist
var app embed.FS

var App fs.FS

func init() {
	fsys, err := fs.Sub(app, "dist")
	if err != nil {
		panic(err)
	}

	App = fsys
}
