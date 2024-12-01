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

// Assets returns the "assets" field that was used in the golte configuration file.
func (r *Renderer) Assets() string {
	return r.infofile.Assets
}

// Renderer is a renderer for svelte components. It is safe to use concurrently across threads.
type Renderer struct {
	mode       string
	serverDir  *fs.FS
	clientDir  *fs.FS
	renderfile renderfile
	infofile   infofile
	template   *template.Template
	vmPool     sync.Pool
	mutex      sync.Mutex
}

// New constructs a renderer from the given FS.
func New(serverDir *fs.FS, clientDir *fs.FS) *Renderer {
	mode := "SSR"

	r := &Renderer{
		mode:      mode,
		serverDir: serverDir,
		clientDir: clientDir,
		template:  template.Must(template.New("").ParseFS(*serverDir, "template.html")).Lookup("template.html"),
	}

	r.vmPool.New = func() interface{} {
		vm := goja.New()
		registry := require.NewRegistryWithLoader(func(path string) ([]byte, error) {
			return fs.ReadFile(*r.serverDir, path)
		})
		registry.Enable(vm)

		console.Enable(vm)
		url.Enable(vm)

		var renderfile renderfile
		if err := vm.ExportTo(require.Require(vm, "./render.js"), &renderfile); err != nil {
			panic(fmt.Sprintf("Failed to load render.js: %v", err))
		}

		var infofile infofile
		if err := vm.ExportTo(require.Require(vm, "./info.js"), &infofile); err != nil {
			panic(fmt.Sprintf("Failed to load info.js: %v", err))
		}

		return vm
	}

	// Initialize the first VM instance
	vm := r.vmPool.Get().(*goja.Runtime)
	defer r.vmPool.Put(vm)

	var renderfile renderfile
	if err := vm.ExportTo(require.Require(vm, "./render.js"), &renderfile); err != nil {
		panic(fmt.Sprintf("Failed to initialize render.js: %v", err))
	}
	r.renderfile = renderfile

	var infofile infofile
	if err := vm.ExportTo(require.Require(vm, "./info.js"), &infofile); err != nil {
		panic(fmt.Sprintf("Failed to initialize info.js: %v", err))
	}
	r.infofile = infofile

	return r
}

type RenderData struct {
	Entries []Entry
	ErrPage string
	SCData  SvelteContextData
}

func (r *Renderer) Render(w http.ResponseWriter, data *RenderData) error {
	if r.mode == "SSR" {
		fmt.Println("Rendering using SSR Mode")
	}

	// Copy entries to avoid modifying the original data
	entries := make([]*Entry, len(data.Entries))
	for i := range data.Entries {
		entries[i] = &data.Entries[i]
	}

	// Get a VM instance from the pool
	vm, ok := r.vmPool.Get().(*goja.Runtime)
	if !ok {
		http.Error(w, "Internal Server Error: VM Pool Error", http.StatusInternalServerError)
		fmt.Println("VM Pool returned invalid runtime")
		return fmt.Errorf("vm pool returned invalid runtime")
	}
	defer r.vmPool.Put(vm)

	// Execute the Render method
	var result *result
	var err error
	r.mutex.Lock() // Ensure thread safety during renderfile.Render execution
	result, err = r.renderfile.Render(entries, &data.SCData, data.ErrPage)
	r.mutex.Unlock()
	if err != nil {
		http.Error(w, "Internal Server Error: Rendering Failed", http.StatusInternalServerError)
		fmt.Printf("Render error: %v\n", err)
		return err
	}

	// Check if rendering resulted in an error
	if result.HasError {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return nil
	}

	// Set response headers
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Vary", "Golte")

	// Render HTML template
	var buf bytes.Buffer
	if err := r.template.Execute(&buf, result); err != nil {
		http.Error(w, "Internal Server Error: Template Execution Failed", http.StatusInternalServerError)
		fmt.Printf("Template execution error: %v\n", err)
		return err
	}

	html := buf.String()

	// Perform resource replacement (SSR mode only)
	if r.mode == "SSR" {
		resources, err := extractResourcePaths(&html)
		if err != nil {
			http.Error(w, "Internal Server Error: Resource Extraction Failed", http.StatusInternalServerError)
			fmt.Printf("Resource extraction error: %v\n", err)
			return err
		}

		err = r.replaceResourcePaths(&html, resources)
		if err != nil {
			http.Error(w, "Internal Server Error: Resource Replacement Failed", http.StatusInternalServerError)
			fmt.Printf("Resource replacement error: %v\n", err)
			return err
		}
	}

	// Write the response
	_, err = w.Write([]byte(html))
	if err != nil {
		fmt.Printf("Write response error: %v\n", err)
	}
	return err
}

func (r *Renderer) replaceResourcePaths(html *string, resources []ResourceEntry) error {
	fileCache := sync.Map{}
	replacementCount := 0

	for _, entry := range resources {
		path := entry.Path
		resource := entry.Resource

		// 跳過外部資源
		if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
			continue
		}

		// 提取檔名
		filename := path[strings.LastIndex(path, "/")+1:]
		content, ok := fileCache.Load(filename)
		if !ok {
			// 嘗試從文件系統讀取檔案
			data, err := findFileInFS(*r.clientDir, filename)
			if err != nil {
				fmt.Printf("Failed to find file: %s, error: %v\n", filename, err)
				continue
			}
			content = data
			fileCache.Store(filename, data)
		}

		// 構建替換內容
		var replacement string
		if resource.TagName == "link" && resource.Attributes["rel"] == "stylesheet" {
			replacement = fmt.Sprintf("<style>%s</style>", string(content.([]byte)))
		}

		if replacement != "" {
			// 使用正則表達式替換標籤
			attrPattern := ""
			for key, value := range resource.Attributes {
				attrPattern += fmt.Sprintf(`\s%s=["']%s["']`, key, regexp.QuoteMeta(value))
			}

			tagPattern := fmt.Sprintf(`<%s%s[^>]*>`, resource.TagName, attrPattern)
			re := regexp.MustCompile(tagPattern)

			if re.MatchString(*html) {
				*html = re.ReplaceAllString(*html, replacement)
				replacementCount++
			} else {
				fmt.Printf("No match found for tag: %s\n", tagPattern)
			}
		}
	}

	fmt.Printf("Total replacements made: %d\n", replacementCount)
	return nil
}

type renderfile struct {
	Manifest map[string]*struct {
		Client string
		CSS    []string
	}
	Render func([]*Entry, *SvelteContextData, string) (*result, error)
}

type result struct {
	Head      string
	Body      string
	HasError  bool
	Resources []ResourceEntry
}

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

type ResourceInfo struct {
	TagName    string
	FullTag    string
	Attributes map[string]string
}

type ResourceEntry struct {
	Path     string
	Resource ResourceInfo
}

func extractResourcePaths(htmlContent *string) ([]ResourceEntry, error) {
	doc, err := html.Parse(strings.NewReader(*htmlContent))
	if err != nil {
		return nil, err
	}

	var resources []ResourceEntry
	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode {
			var resourcePath string
			var originalTag = renderNode(n)
			switch n.Data {
			case "script":
				for _, attr := range n.Attr {
					if attr.Key == "src" {
						resourcePath = attr.Val
						break
					}
				}
			case "link":
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
				resources = append(resources, ResourceEntry{
					Path: resourcePath,
					Resource: ResourceInfo{
						TagName:    n.Data,
						FullTag:    originalTag,
						Attributes: extractAttributes(n.Attr),
					},
				})
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

func renderNode(n *html.Node) string {
	var buf bytes.Buffer
	buf.WriteString("<")
	buf.WriteString(n.Data)

	for _, attr := range n.Attr {
		buf.WriteString(" ")
		buf.WriteString(attr.Key)
		buf.WriteString("=\"")
		buf.WriteString(html.EscapeString(attr.Val))
		buf.WriteString("\"")
	}

	if n.FirstChild != nil {
		buf.WriteString(">")
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
		buf.WriteString("/>")
	}

	return buf.String()
}

func findFileInFS(clientDir fs.FS, filename string) ([]byte, error) {
	paths := []string{"assets/" + filename, "entries/" + filename, "chunks/" + filename}
	for _, path := range paths {
		content, err := fs.ReadFile(clientDir, path)
		if err == nil {
			return content, nil
		}
	}
	return nil, fmt.Errorf("file %s not found", filename)
}

// func (r *Renderer) replaceResourcePaths(html *string, resources []ResourceEntry) error {
// 	fileCache := make(map[string][]byte)
// 	replacementCount := 0

// 	for _, entry := range resources {
// 		path := entry.Path
// 		resource := entry.Resource

// 		// 跳過外部資源
// 		if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
// 			continue
// 		}

// 		// 提取檔名
// 		filename := path[strings.LastIndex(path, "/")+1:]
// 		content, cached := fileCache[filename]
// 		if !cached {
// 			var err error
// 			content, err = findFileInFS(*r.clientDir, filename)
// 			if err != nil {
// 				fmt.Printf("Failed to find file: %s\n", filename)
// 				continue
// 			}
// 			fileCache[filename] = content
// 		}

// 		// 構建替換內容
// 		var replacement string
// 		switch resource.TagName {
// 		// case "script":
// 		// 	replacement = fmt.Sprintf("<script>%s</script>", string(content))
// 		case "link":
// 			if resource.Attributes["rel"] == "stylesheet" {
// 				replacement = fmt.Sprintf("<style>%s</style>", string(content))
// 			}
// 		}

// 		if replacement != "" {
// 			// 建立靈活的正則表達式匹配標籤
// 			attrPattern := ""
// 			for key, value := range resource.Attributes {
// 				attrPattern += fmt.Sprintf(`\s%s=["']%s["']`, key, regexp.QuoteMeta(value))
// 			}

// 			tagPattern := fmt.Sprintf(`<%s%s[^>]*>`, resource.TagName, attrPattern)
// 			re := regexp.MustCompile(tagPattern)

// 			if re.MatchString(*html) {
// 				*html = re.ReplaceAllString(*html, replacement)
// 				replacementCount++
// 			} else {
// 				fmt.Printf("No match found for tag: %s\n", tagPattern)
// 			}
// 		}
// 	}

// 	fmt.Printf("Total replacements made: %d\n", replacementCount)
// 	return nil
// }
