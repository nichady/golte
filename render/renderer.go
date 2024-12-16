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
// ===
// Renderer 是用於渲染 Svelte 組件的渲染器。它可以安全地在多個線程中並發使用。
type Renderer struct {
	renderfile *renderfile
	infofile   infofile
	clientDir  *fs.FS
	template   *template.Template
	vm         *goja.Runtime
	mtx        sync.Mutex
}

// New constructs a renderer from the given FS.
// The FS should be the "server" subdirectory of the build output from "npx golte".
// The second argument is the path where the JS, CSS, and other assets are expected to be served.
// ===
// New 從給定的文件系統構建一個渲染器。
// 文件系統應該是 "npx golte" 構建輸出的 "server" 子目錄。
// 第二個參數是預期提供 JS、CSS 和其他資源的路徑。
func New(serverFS *fs.FS, clientFS *fs.FS) *Renderer {
	tmpl := template.Must(template.New("").ParseFS(*serverFS, "template.html")).Lookup("template.html")

	vm := goja.New()
	vm.SetFieldNameMapper(NewFieldMapper("json"))

	require.NewRegistryWithLoader(func(path string) ([]byte, error) {
		return fs.ReadFile(*serverFS, path)
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
		clientDir:  clientFS,
		template:   tmpl,
		vm:         vm,
		renderfile: &renderfile,
		infofile:   infofile,
	}
}

// RenderData contains all necessary data for rendering components
// ===
// RenderData 包含渲染組件所需的所有數據
type RenderData struct {
	Entries *[]Entry          // Components to render / 要渲染的組件
	ErrPage string            // Error page component / 錯誤頁面組件
	SCData  SvelteContextData // Svelte context data / Svelte 上下文數據
}

// Render renders a slice of entries into the writer.
// It handles resource path extraction and replacement, and sets appropriate headers.
// ===
// Render 將一系列條目渲染到寫入器中。
// 它處理資源路徑的提取和替換，並設置適當的標頭。
func (r *Renderer) Render(w http.ResponseWriter, data *RenderData) error {
	r.mtx.Lock()
	result, err := r.renderfile.Render(*data.Entries, data.SCData, data.ErrPage)
	if err != nil {
		return fmt.Errorf("render error: %w", err)
	}

	resources, err := extractResourcePaths(&result.Head)
	if err != nil {
		return fmt.Errorf("resource extraction error: %w", err)
	}
	err = r.replaceResourcePaths(&result.Head, resources)
	if err != nil {
		return fmt.Errorf("resource replacement error: %w", err)
	}

	r.mtx.Unlock()

	if result.HasError {
		w.WriteHeader(http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Vary", "Golte")

	return r.template.Execute(w, result)
}

// Assets returns the "assets" field that was used in the golte configuration file.
// ===
// Assets 返回在 golte 配置文件中使用的 "assets" 字段。
func (r *Renderer) Assets() string {
	return r.infofile.Assets
}

// result represents the output of component rendering
// ===
// result 表示組件渲染的輸出結果
type result struct {
	Head     string // HTML head content / HTML 頭部內容
	Body     string // Rendered component HTML / 渲染後的組件 HTML
	HasError bool   // Whether an error occurred / 是否發生錯誤
}

// renderfile contains component manifest and render function
// ===
// renderfile 包含組件清單和渲染函數
type renderfile struct {
	Manifest map[string]struct {
		Client string   // Client-side JS path / 客戶端 JS 路徑
		CSS    []string // Component CSS paths / 組件 CSS 路徑
	}
	Render func([]Entry, SvelteContextData, string) (result, error)
}

// Entry represents a component to be rendered, along with its props.
// ===
// Entry 表示要渲染的組件及其屬性。
type Entry struct {
	Comp  string         // Component name / 組件名稱
	Props map[string]any // Component props / 組件屬性
}

// SvelteContextData contains context data for Svelte components
// ===
// SvelteContextData 包含 Svelte 組件的上下文數據
type SvelteContextData struct {
	URL string // Current URL / 當前 URL
}

// infofile contains build configuration information
// ===
// infofile 包含構建配置信息
type infofile struct {
	Assets string // Asset serving path / 資源服務路徑
}
