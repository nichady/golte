package render

import (
	"io"
	"io/fs"
	"text/template"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
)

type Renderer struct {
	template *template.Template
	vm       *goja.Runtime
	render   func(args []string) (result, error)
}

func New(fsys fs.FS) *Renderer {
	tmpl := template.Must(template.New("").ParseFS(fsys, "template.html")).Lookup("template.html")

	vm := goja.New()
	vm.SetFieldNameMapper(goja.UncapFieldNameMapper())

	require.NewRegistryWithLoader(func(path string) ([]byte, error) {
		return fs.ReadFile(fsys, path)
	}).Enable(vm)

	var m Manifest
	vm.ExportTo(require.Require(vm, "./server/manifest.js"), &m)

	return &Renderer{
		template: tmpl,
		vm:       vm,
		render:   m.Render,
	}
}

func (g *Renderer) Render(w io.Writer, components ...string) error {
	r, err := g.render(components)
	if err != nil {
		return err
	}

	return g.template.Execute(w, data{
		Head: r.Head,
		Body: r.Html,
	})
}

type result struct {
	Html string
	Css  any
	Head string
}

type data struct {
	Head string
	Body string
}

type Manifest struct {
	Render func(args []string) (result, error)
}
