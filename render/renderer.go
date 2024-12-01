package render

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"text/template"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/console"
	"github.com/dop251/goja_nodejs/require"
	"github.com/dop251/goja_nodejs/url"
)

// 將 result 類型定義移到前面
type result struct {
	Head     string
	Body     string
	HasError bool
}

// Renderer is a renderer for svelte components. It is safe to use concurrently across threads.
type Renderer struct {
	renderfile renderfile
	infofile   infofile

	template  *template.Template
	vmPool    sync.Pool
	inlineJS  string
	inlineCSS string
}

// New constructs a renderer from the given FS.
// The FS should be the "server" subdirectory of the build output from "npx golte".
// The second argument is the path where the JS, CSS, and other assets are expected to be served.
func New(fsys *fs.FS) *Renderer {
	r := &Renderer{
		template: template.Must(template.New("").ParseFS(*fsys, "template.html")).Lookup("template.html"),
	}

	r.vmPool.New = func() interface{} {
		vm := goja.New()
		vm.SetFieldNameMapper(NewFieldMapper("json"))

		registry := require.NewRegistryWithLoader(func(path string) ([]byte, error) {
			return fs.ReadFile(*fsys, path)
		})
		registry.Enable(vm)

		console.Enable(vm)
		url.Enable(vm)

		var renderfile renderfile
		if err := vm.ExportTo(require.Require(vm, "./render.js"), &renderfile); err != nil {
			panic(err)
		}

		var infofile infofile
		if err := vm.ExportTo(require.Require(vm, "./info.js"), &infofile); err != nil {
			panic(err)
		}

		return vm
	}

	// 初始化第一個 VM 實例
	vm := r.vmPool.Get().(*goja.Runtime)
	var renderfile renderfile
	vm.ExportTo(require.Require(vm, "./render.js"), &renderfile)
	r.renderfile = renderfile

	var infofile infofile
	vm.ExportTo(require.Require(vm, "./info.js"), &infofile)
	r.infofile = infofile

	r.vmPool.Put(vm)

	return r
}

type RenderData struct {
	Entries []Entry
	ErrPage string
	SCData  SvelteContextData
}

// Render renders a slice of entries into the writer.
func (r *Renderer) Render(w http.ResponseWriter, data *RenderData, csr bool) error {
	if !csr {
		entries := make([]*Entry, len(data.Entries))
		for i := range data.Entries {
			entries[i] = &data.Entries[i]
		}

		vm := r.vmPool.Get().(*goja.Runtime)
		origResult, err := r.renderfile.Render(entries, &data.SCData, data.ErrPage)
		r.vmPool.Put(vm)

		if err != nil {
			return err
		}

		if origResult.HasError {
			w.WriteHeader(http.StatusInternalServerError)
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Vary", "Golte")

		// 修改 HTML 內容，注入內聯資源
		head := origResult.Head
		body := origResult.Body

		// 移除 <head> 內的 <style> 標籤及內容，使用正則表達式
		head = regexp.MustCompile(`<style[^>]*>.*?</style>`).ReplaceAllString(head, "")

		// 移除 <body> 內的 <script> 標籤及內容，使用正則表達式
		body = regexp.MustCompile(`<script[^>]*>.*?</script>`).ReplaceAllString(body, "")

		// 在 </head> 前注入 CSS
		if r.inlineCSS != "" {
			head = strings.Replace(head, "</head>", "<style>"+r.inlineCSS+"</style></head>", 1)
		}

		// 在 </body> 前注入 JS
		if r.inlineJS != "" {
			body = strings.Replace(body, "</body>", "<script>"+r.inlineJS+"</script></body>", 1)
		}

		modifiedResult := &result{
			Head:     head,
			Body:     body,
			HasError: origResult.HasError,
		}

		return r.template.Execute(w, modifiedResult)
	}

	resp := &csrResponse{
		Entries: make([]*responseEntry, 0, len(data.Entries)),
	}

	for _, v := range data.Entries {
		comp := r.renderfile.Manifest[v.Comp]
		resp.Entries = append(resp.Entries, &responseEntry{
			File:  comp.Client,
			Props: v.Props,
			CSS:   comp.CSS,
		})
	}

	resp.ErrPage = &responseEntry{
		File: r.renderfile.Manifest[data.ErrPage].Client,
		CSS:  r.renderfile.Manifest[data.ErrPage].CSS,
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Vary", "Golte")

	return json.NewEncoder(w).Encode(resp)
}

// Assets returns the "assets" field that was used in the golte configuration file.
func (r *Renderer) Assets() string {
	return r.infofile.Assets
}

// 移除後面的重複定義
type renderfile struct {
	Manifest map[string]*struct {
		Client string
		CSS    []string
	}
	Render func([]*Entry, *SvelteContextData, string) (*result, error)
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
	Entries []*responseEntry
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

func (r *Renderer) SetInlineAssets(js, css string) {
	r.inlineJS = js
	r.inlineCSS = css
}
