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

// Global variables for regex patterns and resource pools
// ===
// 全局變量，用於正則表達式模式和資源池
var (
	// Regex for extracting href attributes
	// 用於提取 href 屬性的正則表達式
	hrefRegex = regexp.MustCompile(`href=["']([^"']+)["']`)

	// Regex for extracting HTML attributes
	// 用於提取 HTML 屬性的正則表達式
	attrRegex = regexp.MustCompile(`(\w+)=["']([^"']+)["']`)

	// Pool for style buffer reuse
	// 用於重用樣式緩衝區的池
	styleBufferPool = sync.Pool{
		New: func() interface{} {
			return &bytes.Buffer{}
		},
	}

	// Pool for scanner buffer reuse
	// 用於重用掃描器緩衝區的池
	scannerBufPool = sync.Pool{
		New: func() interface{} {
			b := make([]byte, 64*1024)
			return &b
		},
	}
)

// StyleEntry represents a style tag and its replacement
// ===
// StyleEntry 表示樣式標籤及其替換內容
type StyleEntry struct {
	pattern string // Original link tag / 原始的 link 標籤
	style   string // Replacement style tag / 替換的 style 標籤
}

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

// replaceResourcePaths replaces external CSS links with inline styles
// It maintains the original order of stylesheets and caches file contents
// ===
// replaceResourcePaths 將外部 CSS 鏈接替換為內聯樣式
// 它保持樣式表的原始順序並緩存文件內容
func (r *Renderer) replaceResourcePaths(html *string, resources *[]ResourceEntry) error {
	styles := make([]StyleEntry, 0, len(*resources))
	fileCache := sync.Map{}
	styleBuffer := styleBufferPool.Get().(*bytes.Buffer)
	defer styleBufferPool.Put(styleBuffer)

	// 1. 收集所有樣式
	for _, entry := range *resources {
		if !isStylesheet(entry) {
			continue
		}

		// 構建特定於此 entry 的匹配模式
		pattern := fmt.Sprintf(`<link[^>]+href=["']%s["'][^>]*>`, regexp.QuoteMeta(entry.Path))
		linkRe := regexp.MustCompile(pattern)

		if matches := linkRe.FindString(*html); matches != "" {
			content, _ := r.getContent(&fileCache, entry.Path)
			if content != nil {
				styleBuffer.Reset()
				styleBuffer.WriteString("<style>")
				styleBuffer.Write(content)
				styleBuffer.WriteString("</style>")

				styles = append(styles, StyleEntry{
					pattern: matches,
					style:   styleBuffer.String(),
				})
			}
		}
	}

	// 2. 批量替換
	var result strings.Builder
	result.Grow(len(*html))
	lastIndex := 0

	for _, style := range styles {
		index := strings.Index((*html)[lastIndex:], style.pattern)
		if index == -1 {
			continue
		}
		index += lastIndex

		result.WriteString((*html)[lastIndex:index])
		result.WriteString(style.style)

		lastIndex = index + len(style.pattern)
	}
	result.WriteString((*html)[lastIndex:])
	*html = result.String()

	return nil
}

// isStylesheet checks if the entry is a CSS stylesheet link
// Returns false for external URLs
// ===
// isStylesheet 檢查條目是否為 CSS 樣式表鏈接
// 對於外部 URL 返回 false
func isStylesheet(entry ResourceEntry) bool {
	return entry.Resource.TagName == "link" &&
		entry.Resource.Attributes["rel"] == "stylesheet" &&
		!strings.HasPrefix(entry.Path, "http://") &&
		!strings.HasPrefix(entry.Path, "https://")
}

// getContent retrieves file content from cache or filesystem
// File content is cached asynchronously if not found in cache
// ===
// getContent 從緩存或文件系統獲取文件內容
// 如果緩存中未找到，則異步緩存文件內容
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

// extractResourcePaths scans HTML content for stylesheet links
// Uses efficient line-by-line scanning with reusable buffers
// ===
// extractResourcePaths 掃描 HTML 內容以查找樣式表鏈接
// 使用可重用緩衝區進行高效的逐行掃描
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

// findFileInFS searches for a file in common asset directories
// Returns the file content if found, error otherwise
// ===
// findFileInFS 在常見的資源目錄中搜索文件
// 如果找到則返回文件內容，否則返回錯誤
func findFileInFS(clientDir fs.FS, filename string) ([]byte, error) {
	paths := []string{
		"assets/" + filename,
		"entries/" + filename,
		"chunks/" + filename,
	}
	for _, path := range paths {
		if content, err := fs.ReadFile(clientDir, path); err == nil {
			return content, nil
		}
	}
	return nil, fmt.Errorf("file %s not found", filename)
}

// extractAttributesFromLine extracts HTML attributes from a line of text
// Uses regex to parse attribute key-value pairs
// ===
// extractAttributesFromLine 從文本行中提取 HTML 屬性
// 使用正則表達式解析屬性鍵值對
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
