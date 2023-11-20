package render

import (
	"io"
	"io/fs"
	"sync"
	"text/template"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
)

// Renderer is a renderer for svelte components.
type Renderer struct {
	template   *template.Template
	vm         *goja.Runtime
	assetsPath string
	render     func(assetsPath string, entries []Entry) (result, error)
	mtx        sync.Mutex
}

// New constructs a new renderer from the given filesystem.
// The filesystem should be the "server" subdirectory of the build
// output from "npx golte". assetsPath should be the absolute path
// from which asset files are expected to be served.
func New(fsys fs.FS, assetsPath string) *Renderer {
	tmpl := template.Must(template.New("").ParseFS(fsys, "template.html")).Lookup("template.html")

	vm := goja.New()
	vm.SetFieldNameMapper(goja.UncapFieldNameMapper())

	require.NewRegistryWithLoader(func(path string) ([]byte, error) {
		return fs.ReadFile(fsys, path)
	}).Enable(vm)

	var m renderfile
	vm.ExportTo(require.Require(vm, "./renderfile.cjs"), &m)

	return &Renderer{
		template:   tmpl,
		vm:         vm,
		render:     m.Render,
		assetsPath: assetsPath,
	}
}

// Render renders a slice of entries into the writer
func (g *Renderer) Render(w io.Writer, components []Entry) error {
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

type renderfile struct {
	Render func(assetsPath string, entries []Entry) (result, error)
}

// Entry represents a component to be rendered, along with its props.
type Entry struct {
	Comp  string
	Props map[string]any
}
