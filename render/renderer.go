package render

import (
	"encoding/json"
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

	template *template.Template
	vm       *goja.Runtime
	mtx      sync.Mutex
}

// New constructs a renderer from the given FS.
// The FS should be the "server" subdirectory of the build output from "npx golte".
// The second argument is the path where the JS, CSS, and other assets are expected to be served.
func New(fsys *fs.FS) *Renderer {
	tmpl := template.Must(template.New("").ParseFS(*fsys, "template.html")).Lookup("template.html")

	vm := goja.New()
	vm.SetFieldNameMapper(fieldMapper{"json"})

	require.NewRegistryWithLoader(func(path string) ([]byte, error) {
		return fs.ReadFile(*fsys, path)
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
func (r *Renderer) Render(w http.ResponseWriter, data RenderData, csr bool) error {
	if !csr {
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

		return r.template.Execute(w, result)
	}

	var resp csrResponse
	for _, v := range data.Entries {
		comp := r.renderfile.Manifest[v.Comp]
		*resp.Entries = append(*resp.Entries, responseEntry{
			File:  comp.Client,
			Props: v.Props,
			CSS:   comp.CSS,
		})
	}

	resp.ErrPage = &responseEntry{
		File:  r.renderfile.Manifest[data.ErrPage].Client,
		CSS:   r.renderfile.Manifest[data.ErrPage].CSS,
		Props: map[string]any{},
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Vary", "Golte")

	return json.NewEncoder(w).Encode(resp)
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

type csrResponse struct {
	Entries *[]responseEntry
	ErrPage *responseEntry
}

type responseEntry struct {
	File  string
	Props map[string]any
	CSS   []string
}

type infofile struct {
	Assets string
}
