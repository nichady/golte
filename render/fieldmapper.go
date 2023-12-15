package render

import (
	"reflect"
	"strings"

	"github.com/dop251/goja/parser"
)

// fieldMapper implements goja.FieldNameMapper
// it maps fields using the specified tag if set, or simply the field name if tag is not set
type fieldMapper struct {
	tag string
}

func (m fieldMapper) FieldName(_ reflect.Type, field reflect.StructField) string {
	tag, ok := field.Tag.Lookup(m.tag)
	if !ok {
		return field.Name
	}

	idx := strings.IndexByte(tag, ',')
	if idx != -1 {
		tag = tag[:idx]
	}

	if parser.IsIdentifier(tag) {
		return tag
	}
	return ""
}

func (_ fieldMapper) MethodName(_ reflect.Type, method reflect.Method) string {
	return method.Name
}
