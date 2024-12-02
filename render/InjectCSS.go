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
	linkRegex      = regexp.MustCompile(`<link\s[^>]*rel=["']stylesheet["'][^>]*>`)
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
	linkRx := linkRegex
	fileCache := sync.Map{}
	var buf bytes.Buffer

	for _, entry := range *resources {
		if strings.HasPrefix(entry.Path, "http://") || strings.HasPrefix(entry.Path, "https://") {
			continue
		}

		filename := entry.Path[strings.LastIndex(entry.Path, "/")+1:]
		content, ok := fileCache.Load(filename)
		if !ok {
			data, err := findFileInFS(*r.clientDir, filename)
			if err != nil {
				continue
			}
			content = data
			go fileCache.Store(filename, data)
		}

		if entry.Resource.TagName == "link" && entry.Resource.Attributes["rel"] == "stylesheet" {
			buf.Reset()
			buf.WriteString("<style>")
			buf.Write(content.([]byte))
			buf.WriteString("</style>")

			*html = linkRx.ReplaceAllString(*html, buf.String())
		}
	}
	return nil
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
