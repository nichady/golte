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
}

// New constructs a renderer from the given FS.
func New(serverDir *fs.FS, clientDir *fs.FS) *Renderer {
	r := &Renderer{
		mode:      "SSR",
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
			panic(err)
		}

		var infofile infofile
		if err := vm.ExportTo(require.Require(vm, "./info.js"), &infofile); err != nil {
			panic(err)
		}

		return vm
	}

	// Initialize the first VM instance
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
	if r.mode != "SSR" {
		return fmt.Errorf("invalid mode: %s", r.mode)
	}

	fmt.Println("Rendering using SSR Mode")
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
	resources, err := extractResourcePaths(&html)
	if err != nil {
		return err
	}

	result.Resources = resources

	err = r.replaceResourcePaths(&html, resources)
	if err != nil {
		return err
	}

	_, err = w.Write([]byte(html))
	return err
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

func (r *Renderer) replaceResourcePaths(html *string, resources []ResourceEntry) error {
	fileCache := make(map[string][]byte)

	for _, entry := range resources {
		path := entry.Path
		resource := entry.Resource
		if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
			continue
		}

		filename := path[strings.LastIndex(path, "/")+1:]
		content, cached := fileCache[filename]
		if !cached {
			var err error
			content, err = findFileInFS(*r.clientDir, filename)
			if err != nil {
				continue
			}
			fileCache[filename] = content
		}

		var replacement string
		switch resource.TagName {
		case "script":
			replacement = fmt.Sprintf("<script>%s</script>", string(content))
		case "link":
			if resource.Attributes["rel"] == "stylesheet" {
				replacement = fmt.Sprintf("<style>%s</style>", string(content))
			}
		}

		if replacement != "" {
			tagPattern := regexp.MustCompile(regexp.QuoteMeta(resource.FullTag))
			*html = tagPattern.ReplaceAllString(*html, replacement)
		}
	}

	return nil
}
