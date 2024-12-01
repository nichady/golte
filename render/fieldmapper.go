package render

import (
	"reflect"
	"strings"
	"sync"

	"github.com/dop251/goja/parser"
)

// fieldMapper implements [goja.FieldNameMapper]
// 支援標籤映射並加入緩存機制，適合高效且靈活的場景
type fieldMapper struct {
	tag      string
	mappings sync.Map // 緩存結果，確保高效且線程安全
}

// NewFieldMapper 創建一個新的 fieldMapper 實例
func NewFieldMapper(tag string) *fieldMapper {
	return &fieldMapper{
		tag: tag,
	}
}

// FieldName 根據結構字段的標籤返回對應名稱
func (m *fieldMapper) FieldName(t reflect.Type, field reflect.StructField) string {
	// 使用結構名稱和字段名稱作為鍵
	key := t.Name() + "." + field.Name

	// 如果緩存中已存在，直接返回
	if mapped, ok := m.mappings.Load(key); ok {
		return mapped.(string)
	}

	// 從標籤中解析名稱
	tag, ok := field.Tag.Lookup(m.tag)
	if !ok || tag == "-" { // "-" 表示忽略該字段
		m.mappings.Store(key, field.Name)
		return field.Name
	}

	// 處理逗號分隔的標籤，僅取第一部分
	idx := strings.IndexByte(tag, ',')
	if idx != -1 {
		tag = tag[:idx]
	}

	// 確保標籤名稱符合合法標識符
	if parser.IsIdentifier(tag) {
		m.mappings.Store(key, tag)
		return tag
	}

	// 無效標籤時，返回字段名稱
	m.mappings.Store(key, field.Name)
	return field.Name
}

// MethodName 直接返回方法名稱
func (m *fieldMapper) MethodName(_ reflect.Type, method reflect.Method) string {
	return method.Name
}
