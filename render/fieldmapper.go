package render

import (
	"reflect"
	"strings"
	"sync"

	"github.com/dop251/goja/parser"
)

// fieldMapper implements [goja.FieldNameMapper]
// It provides custom field name mapping between Go structs and JavaScript objects
// Used by the goja JavaScript engine for property access and method calls
// ===
// fieldMapper 實現了 [goja.FieldNameMapper] 接口
// 提供 Go 結構體和 JavaScript 對象之間的字段名稱映射
// 由 goja JavaScript 引擎用於屬性訪問和方法調用
type fieldMapper struct {
	tag      string
	mappings sync.Map
	pool     sync.Pool
}

// NewFieldMapper creates a new fieldMapper instance with the specified tag
// The tag determines how Go struct fields are exposed to JavaScript
// ===
// NewFieldMapper 創建一個新的 fieldMapper 實例
// tag 參數決定 Go 結構體字段如何暴露給 JavaScript
func NewFieldMapper(tag string) *fieldMapper {
	return &fieldMapper{
		tag: tag,
		pool: sync.Pool{
			New: func() interface{} {
				// 返回指針以避免分配
				return &[]byte{}
			},
		},
	}
}

// FieldName maps Go struct field names to JavaScript property names
// It checks struct tags first, falling back to the original field name if:
// - The field has no tag
// - The tag is "-" (indicating the field should be ignored)
// - The tag value is not a valid JavaScript identifier
// Results are cached for performance
// ===
// FieldName 將 Go 結構體字段名映射為 JavaScript 屬性名
// 它首先檢查結構體標籤，在以下情況下使用原始字段名：
// - 字段沒有標籤
// - 標籤為 "-"（表示應忽略該字段）
// - 標籤值不是有效的 JavaScript 標識符
// 結果會被緩存以提高性能
func (m *fieldMapper) FieldName(t reflect.Type, field reflect.StructField) string {
	bufPtr := m.pool.Get().(*[]byte)
	buf := (*bufPtr)[:0]

	buf = append(buf, t.Name()...)
	buf = append(buf, '.')
	buf = append(buf, field.Name...)
	key := string(buf)

	m.pool.Put(bufPtr)

	if mapped, ok := m.mappings.Load(key); ok {
		return mapped.(string)
	}

	result := m.computeMapping(field)
	go m.mappings.Store(key, result)
	return result
}

// computeMapping 計算字段的映射名稱
func (m *fieldMapper) computeMapping(field reflect.StructField) string {
	tag, ok := field.Tag.Lookup(m.tag)
	if !ok || tag == "-" {
		return field.Name
	}

	if idx := strings.IndexByte(tag, ','); idx != -1 {
		tag = tag[:idx]
	}

	if parser.IsIdentifier(tag) {
		return tag
	}

	return field.Name
}

// MethodName maps Go method names to JavaScript method names
// Currently returns the original method name unchanged
// This allows Go methods to be called from JavaScript
// ===
// MethodName 將 Go 方法名映射為 JavaScript 方法名
// 目前直接返回原始方法名不做更改
// 這使得 JavaScript 可以調用 Go 的方法
func (m *fieldMapper) MethodName(_ reflect.Type, method reflect.Method) string {
	return method.Name
}
