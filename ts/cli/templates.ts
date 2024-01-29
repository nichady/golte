export function embed(pkg: string) {
	return `
package ${pkg}

import (
	"embed"

	"github.com/nichady/golte"
)

//go:embed */**
var fsys embed.FS

var Golte = golte.New(fsys)
	`;
}
