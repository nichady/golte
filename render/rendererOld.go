package render

import (
	"fmt"
	"io/fs"
	"net/http"
	"sync"
	"text/template"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/console"
	"github.com/dop251/goja_nodejs/require"
	"github.com/dop251/goja_nodejs/url"
)

// Renderer is a renderer for svelte components. It is safe to use concurrently across threads.
type Renderer struct {
	renderfile *renderfile
	infofile   *infofile
	clientDir  *fs.FS
	template   *template.Template
	vm         *goja.Runtime
	mtx        sync.Mutex
}

// New constructs a renderer from the given FS.
// The FS should be the "server" subdirectory of the build output from "npx golte".
// The second argument is the path where the JS, CSS, and other assets are expected to be served.
func New(ServerDir *fs.FS, ClientDir *fs.FS) *Renderer {
	tmpl := template.Must(template.New("").ParseFS(*ServerDir, "template.html")).Lookup("template.html")

	vm := goja.New()
	vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))

	require.NewRegistryWithLoader(func(path string) ([]byte, error) {
		return fs.ReadFile(*ServerDir, path)
	}).Enable(vm)

	console.Enable(vm)
	url.Enable(vm)

	var renderfile renderfile
	err := vm.ExportTo(require.Require(vm, "./render.js"), &renderfile)
	if err != nil {
		panic(err)
	}

	var infofile infofile
	err = vm.ExportTo(require.Require(vm, "./info.js"), &infofile)
	if err != nil {
		panic(err)
	}

	return &Renderer{
		template:   tmpl,
		clientDir:  ClientDir,
		vm:         vm,
		renderfile: &renderfile,
		infofile:   &infofile,
	}
}

type RenderData struct {
	Entries []Entry
	ErrPage string
	SCData  SvelteContextData
}

// Render renders a slice of entries into the writer.
func (r *Renderer) Render(w http.ResponseWriter, data RenderData) error {
	r.mtx.Lock()
	result, err := r.renderfile.Render(data.Entries, data.SCData, data.ErrPage)
	r.mtx.Unlock()

	if err != nil {
		return err
	}

	if result.HasError {
		w.WriteHeader(http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Vary", "Golte")

	resources, err := extractResourcePaths(&result.Head)
	if err != nil {
		http.Error(w, "Internal Server Error: Resource Extraction Failed", http.StatusInternalServerError)
		return fmt.Errorf("resource extraction error: %w", err)
	}

	err = r.replaceResourcePaths(&result.Head, resources)
	if err != nil {
		http.Error(w, "Internal Server Error: Resource Replacement Failed", http.StatusInternalServerError)
		return fmt.Errorf("resource replacement error: %w", err)
	}

	return r.template.Execute(w, result)
}

// Assets returns the "assets" field that was used in the golte configuration file.
func (r *Renderer) Assets() string {
	return r.infofile.Assets
}

type result struct {
	Head     string
	Body     string
	HasError bool
}

type renderfile struct {
	Manifest map[string]struct {
		Client string
		CSS    []string
	}
	Render func([]Entry, SvelteContextData, string) (result, error)
}

// Entry represents a component to be rendered, along with its props.
type Entry struct {
	Comp  string
	Props map[string]any
}

type SvelteContextData struct {
	URL string
}

type infofile struct {
	Assets string
}
