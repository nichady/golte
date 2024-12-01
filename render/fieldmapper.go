package render

import (
	"reflect"
	"sync"
)

// fieldMapper implements [goja.FieldNameMapper]
type fieldMapper struct {
	tag      string
	mappings sync.Map
}

// NewFieldMapper creates a new fieldMapper instance.
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
	if !ok || tag == "" || tag == "-" {
		m.mappings.Store(key, field.Name)
		return field.Name
	}

	m.mappings.Store(key, tag)
	return tag
}

func (m *fieldMapper) MethodName(_ reflect.Type, method reflect.Method) string {
	return method.Name
}
