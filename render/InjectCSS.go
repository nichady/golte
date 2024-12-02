package render

import (
	"bufio"
	"bytes"
	"fmt"
	"io/fs"
	"regexp"
	"strings"
	"sync"
)

var (
	hrefRegex      = regexp.MustCompile(`href=["']([^"']+)["']`)
	attrRegex      = regexp.MustCompile(`(\w+)=["']([^"']+)["']`)
	scannerBufPool = sync.Pool{
		New: func() interface{} {
			b := make([]byte, 64*1024)
			return &b
		},
	}
	pathBuilderPool = sync.Pool{
		New: func() interface{} {
			return &strings.Builder{}
		},
	}
)

// ResourceInfo contains information about a resource tag
// ===
// ResourceInfo 包含資源標籤的信息
type ResourceInfo struct {
	TagName    string            // HTML tag name / HTML 標籤名
	FullTag    string            // Complete tag string / 完整標籤字符串
	Attributes map[string]string // Tag attributes / 標籤屬性
}

// ResourceEntry represents a resource and its HTML tag information
// ===
// ResourceEntry 表示資源及其 HTML 標籤信息
type ResourceEntry struct {
	Path     string       // Resource path / 資源路徑
	Resource ResourceInfo // Resource tag info / 資源標籤信息
}

func (r *Renderer) replaceResourcePaths(html *string, resources *[]ResourceEntry) error {
	// 使用切片保持順序
	type StyleEntry struct {
		pattern string // 原始的 link 標籤
		style   string // 轉換後的 style 標籤
	}
	styles := make([]StyleEntry, 0, len(*resources))

	fileCache := sync.Map{}
	var buf bytes.Buffer

	// 1. 收集所有樣式，保持順序
	for _, entry := range *resources {
		if strings.HasPrefix(entry.Path, "http://") || strings.HasPrefix(entry.Path, "https://") {
			continue
		}

		if entry.Resource.TagName != "link" || entry.Resource.Attributes["rel"] != "stylesheet" {
			continue
		}

		// 構建這個 link 標籤的匹配模式
		pattern := fmt.Sprintf(`<link[^>]+href=["']%s["'][^>]*>`, regexp.QuoteMeta(entry.Path))

		content, _ := r.getContent(&fileCache, entry.Path)
		if content != nil {
			buf.Reset()
			buf.WriteString("<style>")
			buf.Write(content)
			buf.WriteString("</style>")

			styles = append(styles, StyleEntry{
				pattern: pattern,
				style:   buf.String(),
			})
		}
	}

	// 2. 按順序替換
	result := *html
	for _, style := range styles {
		re := regexp.MustCompile(style.pattern)
		result = re.ReplaceAllString(result, style.style)
	}
	*html = result

	return nil
}

// 抽取獲取內容的邏輯
func (r *Renderer) getContent(cache *sync.Map, path string) ([]byte, error) {
	filename := path[strings.LastIndex(path, "/")+1:]
	if content, ok := cache.Load(filename); ok {
		return content.([]byte), nil
	}

	data, err := findFileInFS(*r.clientDir, filename)
	if err != nil {
		return nil, err
	}

	go cache.Store(filename, data)
	return data, nil
}

func extractResourcePaths(htmlContent *string) (*[]ResourceEntry, error) {
	href := hrefRegex
	attr := attrRegex

	resources := make([]ResourceEntry, 0, 8)

	scanBuf := *(scannerBufPool.Get().(*[]byte))
	defer scannerBufPool.Put(&scanBuf)

	scanner := bufio.NewScanner(strings.NewReader(*htmlContent))
	scanner.Buffer(scanBuf, len(scanBuf))

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, "<link") || !strings.Contains(line, "stylesheet") {
			continue
		}

		if matches := href.FindStringSubmatch(line); len(matches) > 1 {
			resources = append(resources, ResourceEntry{
				Path: matches[1],
				Resource: ResourceInfo{
					TagName:    "link",
					Attributes: extractAttributesFromLine(line, attr),
				},
			})
		}
	}

	return &resources, scanner.Err()
}

func findFileInFS(clientDir fs.FS, filename string) ([]byte, error) {
	pathBuilder := pathBuilderPool.Get().(*strings.Builder)
	defer pathBuilderPool.Put(pathBuilder)

	prefixes := []string{"assets/", "entries/", "chunks/"}
	maxLen := len(filename) + 7

	for _, prefix := range prefixes {
		pathBuilder.Reset()
		pathBuilder.Grow(maxLen)
		pathBuilder.WriteString(prefix)
		pathBuilder.WriteString(filename)

		if content, err := fs.ReadFile(clientDir, pathBuilder.String()); err == nil {
			return content, nil
		}
	}
	return nil, fmt.Errorf("file %s not found", filename)
}

// 從單行提取屬性
func extractAttributesFromLine(line string, re *regexp.Regexp) map[string]string {
	attrs := make(map[string]string)
	matches := re.FindAllStringSubmatch(line, -1)
	for _, match := range matches {
		if len(match) > 2 {
			attrs[match[1]] = match[2]
		}
	}
	return attrs
}
