package render

import (
	"io"
	"io/fs"
	"sync"
	"text/template"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
)

type Renderer struct {
	template   *template.Template
	vm         *goja.Runtime
	assetsPath string
	render     func(assetsPath string, args []string) (result, error)
	mtx        sync.Mutex
}

func New(fsys fs.FS, assetsPath string) *Renderer {
	tmpl := template.Must(template.New("").ParseFS(fsys, "template.html")).Lookup("template.html")

	vm := goja.New()
	vm.SetFieldNameMapper(goja.UncapFieldNameMapper())

	require.NewRegistryWithLoader(func(path string) ([]byte, error) {
		return fs.ReadFile(fsys, path)
	}).Enable(vm)

	var m Manifest
	vm.ExportTo(require.Require(vm, "./server/renderfile.cjs"), &m)

	return &Renderer{
		template:   tmpl,
		vm:         vm,
		render:     m.Render,
		assetsPath: assetsPath,
	}
}

func (g *Renderer) Render(w io.Writer, components ...string) error {
	g.mtx.Lock()
	result, err := g.render(g.assetsPath, components)
	g.mtx.Unlock()

	if err != nil {
		return err
	}

	return g.template.Execute(w, result)
}

type result struct {
	Head string
	Body string
}

type Manifest struct {
	Render func(assetsPath string, args []string) (result, error)
}
