package render

import (
	"reflect"
	"sync"

	"github.com/dop251/goja/parser"
)

// fieldMapper implements [goja.FieldNameMapper]
type fieldMapper struct {
	tag string
	// 使用 sync.Map 確保線程安全
	mappings sync.Map
}

// NewFieldMapper 創建一個新的 fieldMapper 實例
func NewFieldMapper(tag string) *fieldMapper {
	return &fieldMapper{
		tag: tag,
	}
}

func (m *fieldMapper) FieldName(t reflect.Type, field reflect.StructField) string {
	key := t.Name() + "." + field.Name

	if mapped, ok := m.mappings.Load(key); ok {
		return mapped.(string)
	}

	tag, ok := field.Tag.Lookup(m.tag)
	if !ok {
		m.mappings.Store(key, field.Name)
		return field.Name
	}

	if tag == "" || tag == "-" {
		m.mappings.Store(key, field.Name)
		return field.Name
	}

	if parser.IsIdentifier(tag) {
		m.mappings.Store(key, tag)
		return tag
	}

	m.mappings.Store(key, field.Name)
	return field.Name
}

func (m *fieldMapper) MethodName(_ reflect.Type, method reflect.Method) string {
	return method.Name
}
