package golte

import (
	"io/fs"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
)

type Golte struct {
	vm     *goja.Runtime
	render func(args []string) (Result, error)
}

func (g *Golte) Render(components ...string) (Result, error) {
	return g.render(components)
}

func New(fsys fs.FS) *Golte {
	vm := goja.New()
	vm.SetFieldNameMapper(goja.UncapFieldNameMapper())

	require.NewRegistryWithLoader(func(path string) ([]byte, error) {
		return fs.ReadFile(fsys, path)
	}).Enable(vm)

	var m Manifest
	vm.ExportTo(require.Require(vm, "./server/manifest.js"), &m)

	return &Golte{
		vm:     vm,
		render: m.Render,
	}
}

type Result struct {
	Html string
	Css  any
	Head string
}

type Manifest struct {
	Render func(args []string) (Result, error)
}
