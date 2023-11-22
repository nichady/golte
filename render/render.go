package render

import (
	"io"
	"io/fs"
	"sync"
	"text/template"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/console"
	"github.com/dop251/goja_nodejs/require"
)

// Renderer is a renderer for svelte components. It uses a *goja.Runtime underneath the hood
// to run javascript.
type Renderer struct {
	template      *template.Template
	vm            *goja.Runtime
	assetsPath    string
	render        func(assetsPath string, entries []Entry) (result, error)
	isRenderError func(goja.Value) bool
	mtx           sync.Mutex
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

	console.Enable(vm)

	var renderfile renderfile
	vm.ExportTo(require.Require(vm, "./renderfile.cjs"), &renderfile)

	var exports exports
	vm.ExportTo(require.Require(vm, "./exports.cjs"), &exports)

	return &Renderer{
		template:      tmpl,
		vm:            vm,
		render:        renderfile.Render,
		isRenderError: exports.IsRenderError,
		assetsPath:    assetsPath,
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

func (g *Renderer) ToRenderError(err error) (RenderError, bool) {
	ex, ok := err.(*goja.Exception)
	if !ok {
		return RenderError{}, false
	}

	g.mtx.Lock()
	defer g.mtx.Unlock()

	if !g.isRenderError(ex.Value()) {
		return RenderError{}, false
	}

	var rerr RenderError
	if g.vm.ExportTo(ex.Value(), &rerr) != nil {
		return RenderError{}, false
	}

	return rerr, true
}

type exports struct {
	IsRenderError func(goja.Value) bool
}

type RenderError struct {
	Cause goja.Value
	Index int
}
