package render

import (
	"io/fs"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
)

type Renderer struct {
	vm     *goja.Runtime
	render func(args []string) (Result, error)
}

func (g *Renderer) Render(components ...string) (Result, error) {
	return g.render(components)
}

func New(fsys fs.FS) *Renderer {
	vm := goja.New()
	vm.SetFieldNameMapper(goja.UncapFieldNameMapper())

	require.NewRegistryWithLoader(func(path string) ([]byte, error) {
		return fs.ReadFile(fsys, path)
	}).Enable(vm)

	var m Manifest
	vm.ExportTo(require.Require(vm, "./server/manifest.js"), &m)

	return &Renderer{
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
