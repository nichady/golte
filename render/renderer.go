package render

import (
	"bytes"
	"fmt"
	"io/fs"
	"net/http"
	"regexp"
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
	mode       string
	serverDir  *fs.FS
	clientDir  *fs.FS
	renderfile renderfile
	infofile   infofile
	template   *template.Template
	vmPool     sync.Pool
}

// New constructs a renderer from the given FS.
// The FS should be the "server" subdirectory of the build output from "npx golte".
// The second argument is the path where the JS, CSS, and other assets are expected to be served.
func New(serverDir *fs.FS, clientDir *fs.FS) *Renderer {
	// 讀取並印出 template.html 的內容
	// templateContent, err := fs.ReadFile(*serverDir, "template.html")
	// if err == nil {
	// 	fmt.Println("=== template.html content ===")
	// 	fmt.Println(string(templateContent))
	// 	fmt.Println("=== End of template.html ===")
	// }

	// 列出 fsys 中的所有檔案，包含完整路徑
	fmt.Println("=== Listing all files in fsys ===")
	fs.WalkDir(*serverDir, ".", func(path string, d fs.DirEntry, err error) error {
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
		mode:      "SSR",
		serverDir: serverDir,
		clientDir: clientDir,
		template:  template.Must(template.New("").ParseFS(*serverDir, "template.html")).Lookup("template.html"),
	}

	r.vmPool.New = func() interface{} {
		vm := goja.New()
		vm.SetFieldNameMapper(NewFieldMapper("json"))

		registry := require.NewRegistryWithLoader(func(path string) ([]byte, error) {
			return fs.ReadFile(*r.serverDir, path)
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
func (r *Renderer) Render(w http.ResponseWriter, data *RenderData) error {
	switch r.mode {
	case "SSR":
		fmt.Println("rendering using SSR Mode")
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
	case "CSR":
		// fmt.Println("rendering using CSR Mode")
		// resp := &csrResponse{
		// 	Entries: make([]*responseEntry, 0, len(data.Entries)),
		// }

		// for _, v := range data.Entries {
		// 	comp := r.renderfile.Manifest[v.Comp]
		// 	resp.Entries = append(resp.Entries, &responseEntry{
		// 		File:  comp.Client,
		// 		Props: v.Props,
		// 		CSS:   comp.CSS,
		// 	})
		// }

		// resp.ErrPage = &responseEntry{
		// 	File: r.renderfile.Manifest[data.ErrPage].Client,
		// 	CSS:  r.renderfile.Manifest[data.ErrPage].CSS,
		// }

		// w.Header().Set("Content-Type", "application/json; charset=utf-8")
		// w.Header().Set("Vary", "Golte")

		// return json.NewEncoder(w).Encode(resp)
	}

	return fmt.Errorf("invalid mode: %s", r.mode)
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
			var resourcePath string
			var originalTag string

			// 保存原始的 HTML 標籤字串
			originalTag = renderNode(n)

			switch n.Data {
			case "script":
				for _, attr := range n.Attr {
					if attr.Key == "src" {
						resourcePath = attr.Val
						break
					}
				}
			case "link":
				// 檢查是否為 stylesheet
				isStylesheet := false
				var href string
				for _, attr := range n.Attr {
					if attr.Key == "rel" && attr.Val == "stylesheet" {
						isStylesheet = true
					}
					if attr.Key == "href" {
						href = attr.Val
					}
				}
				if isStylesheet {
					resourcePath = href
				}
			}

			if resourcePath != "" {
				fmt.Printf("Found %s: %s with original tag: %s\n", n.Data, resourcePath, originalTag)
				resources[resourcePath] = ResourceInfo{
					TagName:    n.Data,
					FullTag:    originalTag,
					Attributes: extractAttributes(n.Attr),
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

// 修改 renderNode 函數
func renderNode(n *html.Node) string {
	var buf bytes.Buffer

	// 開始標籤
	buf.WriteString("<")
	buf.WriteString(n.Data)

	// 先找到 href/src 屬性
	var mainAttr *html.Attribute
	var otherAttrs []html.Attribute

	for _, attr := range n.Attr {
		if attr.Key == "href" || attr.Key == "src" {
			mainAttr = &attr
		} else {
			otherAttrs = append(otherAttrs, attr)
		}
	}

	// 先寫入 href/src 屬性（如果存在）
	if mainAttr != nil {
		buf.WriteString(" ")
		buf.WriteString(mainAttr.Key)
		buf.WriteString("=\"")
		buf.WriteString(html.EscapeString(mainAttr.Val))
		buf.WriteString("\"")
	}

	// 再寫入其他屬性
	for _, attr := range otherAttrs {
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
func findFileInFS(clientDir fs.FS, filename string) ([]byte, error) {
	fmt.Printf("Searching for file: %s\n", filename)

	// 直接嘗試在 assets 目錄中查找 CSS 檔案
	if strings.HasSuffix(filename, ".css") {
		fullPath := "assets/" + filename
		content, err := fs.ReadFile(clientDir, fullPath)
		if err == nil {
			fmt.Printf("Found CSS file at: %s\n", fullPath)
			return content, nil
		}
	}

	// 對於 JS 檔案，嘗試在 entries 和 chunks 目錄中查找
	if strings.HasSuffix(filename, ".js") {
		paths := []string{"entries/" + filename, "chunks/" + filename}
		for _, path := range paths {
			content, err := fs.ReadFile(clientDir, path)
			if err == nil {
				fmt.Printf("Found JS file at: %s\n", path)
				return content, nil
			}
		}
	}

	return nil, fmt.Errorf("file %s not found in any directory", filename)
}

// 修改 replaceResourcePaths 函數
func (r *Renderer) replaceResourcePaths(html *string, resources map[string]ResourceInfo) error {
	replacementCount := 0
	for path, resource := range resources {
		// 跳過外部資源（CDN）
		if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
			fmt.Printf("Skipping external resource: %s\n", path)
			continue
		}

		// 只處理 golte_ 相關的資源
		if !strings.Contains(path, "golte_/") {
			fmt.Printf("Skipping non-golte resource: %s\n", path)
			continue
		}

		fmt.Printf("Processing resource: %s\n", path)

		// 取得完整路徑而不僅僅是檔名
		filename := path[strings.LastIndex(path, "/")+1:]

		// 在所有子目錄中搜尋檔案
		content, err := findFileInFS(*r.clientDir, filename)
		if err != nil {
			fmt.Printf("Resource not found: %v\n", err)
			continue
		}

		var replacement string
		switch resource.TagName {
		case "script":
			// 所有 script 都內聯
			replacement = "<script"
			for key, value := range resource.Attributes {
				if key != "src" {
					replacement += fmt.Sprintf(" %s=\"%s\"", key, value)
				}
			}
			replacement += ">\n" + string(content) + "\n</script>"
			fmt.Printf("Replacing script: %s\n", resource.FullTag)

		case "link":
			if resource.Attributes["rel"] == "stylesheet" {
				// CSS 內聯
				replacement = "<style data-source=\"" + filename + "\">\n" + string(content) + "\n</style>"
				fmt.Printf("Replacing CSS: %s\n", resource.FullTag)
			}
		}

		// 更改替換邏輯以更精確地匹配和替換完整的 HTML 標籤
		if replacement != "" {
			// 使用正則表達式匹配標籤名稱和屬性，並且忽略屬性的順序
			tagPattern := regexp.MustCompile(fmt.Sprintf(`<%s[^>]*>`, resource.TagName))
			matches := tagPattern.FindAllString(*html, -1)
			for _, match := range matches {
				if strings.Contains(match, resource.Attributes["rel"]) {
					*html = strings.Replace(*html, match, replacement, 1)
					replacementCount++
					fmt.Printf("Successfully replaced %s\n", resource.FullTag)
					break
				}
			}
		}
	}

	fmt.Printf("Total replacements: %d\n", replacementCount)
	return nil
}

// 注意事項：
// 1. 使用 `regexp.QuoteMeta` 對資源標籤進行編碼，避免正則表達式中出現特殊字元導致替換失敗。
// 2. 使用正則表達式來更精確地匹配完整的 HTML 標籤，並忽略屬性順序，以防止出現部分替換或錯誤替換的情況。
