package render

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"sync"
	"text/template"

	"strings"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/console"
	"github.com/dop251/goja_nodejs/require"
	"github.com/dop251/goja_nodejs/url"
	"golang.org/x/net/html"
)

// Renderer is a renderer for svelte components. It is safe to use concurrently across threads.
type Renderer struct {
	fsys       *fs.FS
	renderfile renderfile
	infofile   infofile

	template *template.Template
	vmPool   sync.Pool
}

// New constructs a renderer from the given FS.
// The FS should be the "server" subdirectory of the build output from "npx golte".
// The second argument is the path where the JS, CSS, and other assets are expected to be served.
func New(fsys *fs.FS) *Renderer {
	// 列出 fsys 中的所有檔案，包含完整路徑
	fmt.Println("=== Listing all files in fsys ===")
	fs.WalkDir(*fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			fmt.Printf("File: %s\n", path)
		}
		return nil
	})
	fmt.Println("=== End of file listing ===")

	r := &Renderer{
		fsys:     fsys,
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
		// 轉換 []Entry 到 []*Entry
		entries := make([]*Entry, len(data.Entries))
		for i := range data.Entries {
			entries[i] = &data.Entries[i]
		}

		vm := r.vmPool.Get().(*goja.Runtime)
		result, err := r.renderfile.Render(entries, &data.SCData, data.ErrPage)
		r.vmPool.Put(vm)

		if err != nil {
			return err
		}

		if result.HasError {
			w.WriteHeader(http.StatusInternalServerError)
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Vary", "Golte")

		var buf bytes.Buffer
		err = r.template.Execute(&buf, result)
		if err != nil {
			return err
		}

		html := buf.String()

		// 解析資源路徑
		resources, err := extractResourcePaths(&html)
		if err != nil {
			return err
		}

		// 將資源資訊添加到結果中
		result.Resources = resources

		// 替換資源引用
		err = r.replaceResourcePaths(&html, resources)
		if err != nil {
			return err
		}

		// 在寫入 response 之前，加入除錯資訊
		fmt.Printf("SSR Mode: Resources found: %d\n", len(resources))
		for path := range resources {
			fmt.Printf("Resource: %s\n", path)
		}

		// 確保替換後的 HTML 不包含原始的外部資源引用
		if strings.Contains(html, "/golte_/assets/") || strings.Contains(html, "/golte_/entries/") {
			fmt.Println("Warning: External resources still present in HTML after replacement")
		}

		_, err = w.Write([]byte(html))
		return err
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

type result struct {
	Head      string
	Body      string
	HasError  bool
	Resources map[string]ResourceInfo
}

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

type ResourceInfo struct {
	TagName    string
	FullTag    string // 儲存完整的 HTML 標籤
	Attributes map[string]string
}

func extractResourcePaths(htmlContent *string) (map[string]ResourceInfo, error) {
	doc, err := html.Parse(strings.NewReader(*htmlContent))
	if err != nil {
		return nil, err
	}

	resources := make(map[string]ResourceInfo)

	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "script":
				for _, attr := range n.Attr {
					if attr.Key == "src" {
						resources[attr.Val] = ResourceInfo{
							TagName:    "script",
							FullTag:    renderNode(n),
							Attributes: extractAttributes(n.Attr),
						}
					}
				}
			case "link":
				for _, attr := range n.Attr {
					if attr.Key == "href" {
						resources[attr.Val] = ResourceInfo{
							TagName:    "link",
							FullTag:    renderNode(n),
							Attributes: extractAttributes(n.Attr),
						}
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}

	traverse(doc)
	return resources, nil
}

func extractAttributes(attrs []html.Attribute) map[string]string {
	attributes := make(map[string]string)
	for _, attr := range attrs {
		attributes[attr.Key] = attr.Val
	}
	return attributes
}

// 新增函數來渲染完整的 HTML 標籤
func renderNode(n *html.Node) string {
	var buf bytes.Buffer

	// 開始標
	buf.WriteString("<")
	buf.WriteString(n.Data)

	// 寫入屬性
	for _, attr := range n.Attr {
		buf.WriteString(" ")
		buf.WriteString(attr.Key)
		buf.WriteString("=\"")
		buf.WriteString(html.EscapeString(attr.Val))
		buf.WriteString("\"")
	}

	if n.FirstChild != nil {
		buf.WriteString(">")
		// 遞迴渲染子節點
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.TextNode {
				buf.WriteString(html.EscapeString(c.Data))
			} else {
				buf.WriteString(renderNode(c))
			}
		}
		buf.WriteString("</")
		buf.WriteString(n.Data)
		buf.WriteString(">")
	} else {
		// 自閉合標籤
		buf.WriteString("/>")
	}

	return buf.String()
}

// 修改輔助函數來搜尋檔案
func findFileInFS(fsys fs.FS, filename string) ([]byte, error) {
	fmt.Printf("Searching for file: %s\n", filename)

	// 嘗試在 server 和 client 的子目錄中搜尋
	searchPaths := []string{
		"server",         // server 目錄
		"client/assets",  // client 的 assets 目錄
		"client/entries", // client 的 entries 目錄
		"client/chunks",  // client 的 chunks 目錄
	}

	for _, basePath := range searchPaths {
		fullPath := basePath + "/" + filename
		fmt.Printf("Trying path: %s\n", fullPath)

		content, err := fs.ReadFile(fsys, fullPath)
		if err == nil {
			fmt.Printf("Found file at: %s\n", fullPath)
			return content, nil
		}
	}

	return nil, fmt.Errorf("file %s not found in server or client directories", filename)
}

// 修改 replaceResourcePaths 函數
func (r *Renderer) replaceResourcePaths(html *string, resources map[string]ResourceInfo) error {
	replacementCount := 0
	for path, resource := range resources {
		// 跳過外部資源（CDN）
		if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
			continue
		}

		// 只處理 golte_ 相關的資源
		if !strings.Contains(path, "/golte_/") {
			continue
		}

		var content []byte
		var err error

		fmt.Printf("Processing resource: %s\n", path)

		// 取得檔名
		parts := strings.Split(path, "/")
		filename := parts[len(parts)-1]

		// 在所有子目錄中搜尋檔案
		content, err = findFileInFS(*r.fsys, filename)
		if err != nil {
			fmt.Printf("Resource not found: %v\n", err)
			continue
		}

		var replacement string
		switch resource.TagName {
		case "script":
			replacement = "<script"
			for key, value := range resource.Attributes {
				if key != "src" {
					replacement += fmt.Sprintf(" %s=\"%s\"", key, value)
				}
			}
			replacement += ">\n" + string(content) + "\n</script>"

		case "link":
			if resource.Attributes["rel"] == "stylesheet" {
				replacement = "<style>\n" + string(content) + "\n</style>"
			} else {
				continue
			}
		}

		*html = strings.Replace(*html, resource.FullTag, replacement, 1)
		replacementCount++
	}
	fmt.Printf("Total replacements: %d\n", replacementCount)
	return nil
}
