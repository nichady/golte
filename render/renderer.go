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
func (r *Renderer) Render(w io.Writer, components []Entry) error {
	r.mtx.Lock()
	result, err := r.render(r.assetsPath, components)
	r.mtx.Unlock()

	if err != nil {
		return r.tryConvToRenderError(err)
	}

	return r.template.Execute(w, result)
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

type exports struct {
	IsRenderError func(goja.Value) bool
}
