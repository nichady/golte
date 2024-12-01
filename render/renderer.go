package render

// import (
// 	"bytes"
// 	"encoding/json"
// 	"fmt"
// 	"io/fs"
// 	"net/http"
// 	"reflect"
// 	"regexp"
// 	"strings"
// 	"sync"
// 	"text/template"

// 	"github.com/dop251/goja"
// 	"github.com/dop251/goja_nodejs/console"
// 	"github.com/dop251/goja_nodejs/require"
// 	"github.com/dop251/goja_nodejs/url"
// 	"golang.org/x/net/html"
// )

// type Renderer struct {
// 	serverDir  *fs.FS
// 	clientDir  *fs.FS
// 	renderfile renderfile
// 	infofile   infofile
// 	template   *template.Template
// 	vmPool     sync.Pool
// 	mutex      sync.Mutex
// }

// type RenderData struct {
// 	Entries []Entry
// 	ErrPage string
// 	SCData  SvelteContextData
// }

// func New(serverDir *fs.FS, clientDir *fs.FS) *Renderer {
// 	r := &Renderer{
// 		serverDir: serverDir,
// 		clientDir: clientDir,
// 		template:  template.Must(template.New("").ParseFS(*serverDir, "template.html")).Lookup("template.html"),
// 	}

// 	r.vmPool.New = func() interface{} {
// 		vm := goja.New()
// 		registry := require.NewRegistryWithLoader(func(path string) ([]byte, error) {
// 			fmt.Printf("Loading file: %s\n", path)
// 			return fs.ReadFile(*serverDir, path)
// 		})
// 		registry.Enable(vm)

// 		console.Enable(vm)
// 		url.Enable(vm)

// 		var renderfile renderfile
// 		if err := vm.ExportTo(require.Require(vm, "./render.js"), &renderfile); err != nil {
// 			fmt.Printf("Error loading render.js: %v\n", err)
// 			return nil
// 		}

// 		if renderfile.RenderJSON == nil {
// 			fmt.Println("Error: RenderJSON method is not initialized in render.js")
// 			return nil
// 		}

// 		var infofile infofile
// 		if err := vm.ExportTo(require.Require(vm, "./info.js"), &infofile); err != nil {
// 			fmt.Printf("Error loading info.js: %v\n", err)
// 			return nil
// 		}

// 		return vm
// 	}

// 	vm := r.vmPool.Get().(*goja.Runtime)
// 	if vm == nil {
// 		panic("Failed to initialize VM during Renderer creation")
// 	}
// 	defer r.vmPool.Put(vm)

// 	if err := vm.ExportTo(require.Require(vm, "./render.js"), &r.renderfile); err != nil {
// 		panic(fmt.Sprintf("Failed to initialize render.js: %v", err))
// 	}
// 	if r.renderfile.RenderJSON == nil {
// 		panic("RenderJSON method in render.js is not initialized")
// 	}

// 	if err := vm.ExportTo(require.Require(vm, "./info.js"), &r.infofile); err != nil {
// 		panic(fmt.Sprintf("Failed to initialize info.js: %v", err))
// 	}

// 	return r
// }

// func (r *Renderer) Render(w http.ResponseWriter, data *RenderData) error {
// 	if r.renderfile.RenderJSON == nil {
// 		http.Error(w, "Internal Server Error: RenderJSON not initialized", http.StatusInternalServerError)
// 		return fmt.Errorf("renderfile.RenderJSON is nil")
// 	}

// 	dataJSON, err := json.Marshal(data)
// 	if err != nil {
// 		http.Error(w, "Internal Server Error: Data Serialization Failed", http.StatusInternalServerError)
// 		return fmt.Errorf("data serialization error: %w", err)
// 	}

// 	vm, ok := r.vmPool.Get().(*goja.Runtime)
// 	if !ok {
// 		http.Error(w, "Internal Server Error: VM Pool Error", http.StatusInternalServerError)
// 		return fmt.Errorf("vm pool returned invalid runtime")
// 	}
// 	defer r.vmPool.Put(vm)

// 	var result *result
// 	r.mutex.Lock()
// 	result, err = r.renderfile.RenderJSON(string(dataJSON))
// 	r.mutex.Unlock()
// 	if err != nil {
// 		http.Error(w, "Internal Server Error: Rendering Failed", http.StatusInternalServerError)
// 		return fmt.Errorf("render error: %w", err)
// 	}

// 	if result.HasError {
// 		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
// 		return nil
// 	}

// 	w.Header().Set("Content-Type", "text/html; charset=utf-8")
// 	var buf bytes.Buffer
// 	if err := r.template.Execute(&buf, result); err != nil {
// 		http.Error(w, "Internal Server Error: Template Execution Failed", http.StatusInternalServerError)
// 		return fmt.Errorf("template execution error: %w", err)
// 	}

// 	html := buf.String()
// 	resources, err := extractResourcePaths(&html)
// 	if err != nil {
// 		http.Error(w, "Internal Server Error: Resource Extraction Failed", http.StatusInternalServerError)
// 		return fmt.Errorf("resource extraction error: %w", err)
// 	}

// 	err = r.replaceResourcePaths(&html, resources)
// 	if err != nil {
// 		http.Error(w, "Internal Server Error: Resource Replacement Failed", http.StatusInternalServerError)
// 		return fmt.Errorf("resource replacement error: %w", err)
// 	}

// 	_, err = w.Write([]byte(html))
// 	return err
// }

// func (r *Renderer) replaceResourcePaths(html *string, resources []ResourceEntry) error {
// 	fileCache := sync.Map{}
// 	for _, entry := range resources {
// 		if strings.HasPrefix(entry.Path, "http://") || strings.HasPrefix(entry.Path, "https://") {
// 			continue
// 		}
// 		filename := entry.Path[strings.LastIndex(entry.Path, "/")+1:]
// 		content, ok := fileCache.Load(filename)
// 		if !ok {
// 			data, err := findFileInFS(*r.clientDir, filename)
// 			if err != nil {
// 				continue
// 			}
// 			content = data
// 			fileCache.Store(filename, data)
// 		}
// 		if entry.Resource.TagName == "link" && entry.Resource.Attributes["rel"] == "stylesheet" {
// 			replacement := fmt.Sprintf("<style>%s</style>", string(content.([]byte)))
// 			attrPattern := ""
// 			for key, value := range entry.Resource.Attributes {
// 				attrPattern += fmt.Sprintf(`\s%s=["']%s["']`, key, regexp.QuoteMeta(value))
// 			}
// 			tagPattern := fmt.Sprintf(`<%s%s[^>]*>`, entry.Resource.TagName, attrPattern)
// 			re := regexp.MustCompile(tagPattern)
// 			*html = re.ReplaceAllString(*html, replacement)
// 		}
// 	}
// 	return nil
// }

// type renderfile struct {
// 	Manifest map[string]*struct {
// 		Client string
// 		CSS    []string
// 	}
// 	RenderJSON func(string) (*result, error) // 修改為支援 JSON 字串
// }

// type result struct {
// 	Head     string
// 	Body     string
// 	HasError bool
// }

// type Entry struct {
// 	Comp  string
// 	Props map[string]any
// }

// type SvelteContextData struct {
// 	URL string
// }

// type infofile struct {
// 	Assets string
// }

// type ResourceInfo struct {
// 	TagName    string
// 	FullTag    string
// 	Attributes map[string]string
// }

// type ResourceEntry struct {
// 	Path     string
// 	Resource ResourceInfo
// }

// func extractResourcePaths(htmlContent *string) ([]ResourceEntry, error) {
// 	doc, err := html.Parse(strings.NewReader(*htmlContent))
// 	if err != nil {
// 		return nil, err
// 	}

// 	var resources []ResourceEntry
// 	var traverse func(*html.Node)
// 	traverse = func(n *html.Node) {
// 		if n.Type == html.ElementNode {
// 			var resourcePath string
// 			switch n.Data {
// 			case "link":
// 				isStylesheet := false
// 				var href string
// 				for _, attr := range n.Attr {
// 					if attr.Key == "rel" && attr.Val == "stylesheet" {
// 						isStylesheet = true
// 					}
// 					if attr.Key == "href" {
// 						href = attr.Val
// 					}
// 				}
// 				if isStylesheet {
// 					resourcePath = href
// 				}
// 			}

// 			if resourcePath != "" {
// 				resources = append(resources, ResourceEntry{
// 					Path: resourcePath,
// 					Resource: ResourceInfo{
// 						TagName:    n.Data,
// 						Attributes: extractAttributes(n.Attr),
// 					},
// 				})
// 			}
// 		}
// 		for c := n.FirstChild; c != nil; c = c.NextSibling {
// 			traverse(c)
// 		}
// 	}

// 	traverse(doc)
// 	return resources, nil
// }

// func extractAttributes(attrs []html.Attribute) map[string]string {
// 	attributes := make(map[string]string)
// 	for _, attr := range attrs {
// 		attributes[attr.Key] = attr.Val
// 	}
// 	return attributes
// }

// func findFileInFS(clientDir fs.FS, filename string) ([]byte, error) {
// 	paths := []string{"assets/" + filename, "entries/" + filename, "chunks/" + filename}
// 	for _, path := range paths {
// 		content, err := fs.ReadFile(clientDir, path)
// 		if err == nil {
// 			return content, nil
// 		}
// 	}
// 	return nil, fmt.Errorf("file %s not found", filename)
// }

// // ConvertStructsToJSON 遞迴將 map[string]any 和切片中的結構體轉換為 JSON 兼容的格式
// func ConvertStructsToJSON(data map[string]any) (map[string]any, error) {
// 	converted := make(map[string]any)
// 	for key, value := range data {
// 		switch reflect.TypeOf(value).Kind() {
// 		case reflect.Map:
// 			// 遞迴處理 map
// 			if v, ok := value.(map[string]any); ok {
// 				innerMap, err := ConvertStructsToJSON(v)
// 				if err != nil {
// 					return nil, err
// 				}
// 				converted[key] = innerMap
// 			} else {
// 				converted[key] = value
// 			}
// 		case reflect.Slice:
// 			// 處理切片
// 			v := reflect.ValueOf(value)
// 			convertedSlice := make([]any, v.Len())
// 			for i := 0; i < v.Len(); i++ {
// 				item := v.Index(i).Interface()
// 				if reflect.TypeOf(item).Kind() == reflect.Struct {
// 					// 將結構體轉換為 JSON 兼容格式
// 					jsonData, err := json.Marshal(item)
// 					if err != nil {
// 						return nil, err
// 					}
// 					var jsonMap map[string]any
// 					if err := json.Unmarshal(jsonData, &jsonMap); err != nil {
// 						return nil, err
// 					}
// 					convertedSlice[i] = jsonMap
// 				} else {
// 					convertedSlice[i] = item
// 				}
// 			}
// 			converted[key] = convertedSlice
// 		default:
// 			// 其他類型直接保留
// 			converted[key] = value
// 		}
// 	}
// 	return converted, nil
// }

// // Assets returns the "assets" field that was used in the Golte configuration file.
// func (r *Renderer) Assets() string {
// 	if r.infofile.Assets == "" {
// 		fmt.Println("Warning: Assets field in infofile is empty")
// 	}
// 	return r.infofile.Assets
// }
