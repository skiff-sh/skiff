package schema

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/skiff-sh/api/go/skiff/registry/v1alpha1"

	"github.com/skiff-sh/skiff/pkg/bufferpool"

	"github.com/skiff-sh/skiff/pkg/fields"
)

func NewJSONSchema(s *Schema) (*jsonschema.Schema, error) {
	sch := &jsonschema.Schema{
		Type:       "object",
		Properties: map[string]*jsonschema.Schema{},
	}

	for _, field := range s.Fields {
		fi, err := NewJSONSchemaField(field)
		if err != nil {
			return nil, fmt.Errorf("field %s: %w", field.Proto.GetName(), err)
		}
		sch.Properties[field.Proto.GetName()] = fi
	}

	return sch, nil
}

func NewJSONSchemaField(f *Field) (*jsonschema.Schema, error) {
	switch f.Proto.GetType() {
	case v1alpha1.Field_bool:
		return &jsonschema.Schema{
			Type:        "bool",
			Description: f.Proto.GetDescription(),
			Default:     formatValue(f.Default),
		}, nil
	case v1alpha1.Field_string:
		return &jsonschema.Schema{
			Type:        "string",
			Description: f.Proto.GetDescription(),
			Default:     formatValue(f.Default),
			Enum:        f.Enum,
		}, nil
	case v1alpha1.Field_number:
		return &jsonschema.Schema{
			Type:        "number",
			Description: f.Proto.GetDescription(),
			Default:     formatValue(f.Default),
			Enum:        f.Enum,
		}, nil
	case v1alpha1.Field_array:
		it := &jsonschema.Schema{
			Enum: f.Enum,
		}
		//nolint:exhaustive // only a subset.
		switch f.Proto.GetItems().GetType() {
		case v1alpha1.Field_string:
			it.Type = "string"
		case v1alpha1.Field_number:
			it.Type = "number"
		default:
			return nil, errors.New("array items type must be either string or number")
		}

		return &jsonschema.Schema{
			Type:        "array",
			Description: f.Proto.GetDescription(),
			Default:     formatValue(f.Default),
			Items:       it,
		}, nil
	default:
		return nil, fmt.Errorf("unknown type %s", f.Proto.GetType().String())
	}
}

func formatValue(a any) []byte {
	if a == nil {
		return []byte("null")
	}
	switch typ := a.(type) {
	case string:
		return []byte(typ)
	case float64:
		return []byte(fields.FormatFloat(typ))
	case bool:
		return []byte(strconv.FormatBool(fields.Cast[bool](a)))
	case []string:
		return arrWriter(typ, func(v string) string {
			return v
		})
	case []bool:
		return arrWriter(typ, strconv.FormatBool)
	case []float64:
		return arrWriter(typ, fields.FormatFloat)
	}
	return []byte("null")
}

func arrWriter[S ~[]E, E any](s S, f func(v E) string) []byte {
	buff := bufferpool.GetBytesBuffer()
	defer bufferpool.PutBytesBuffers(buff)
	buff.WriteRune('[')
	for i, v := range s {
		buff.WriteString(f(v))
		if i < len(s)-1 {
			buff.WriteString(", ")
		}
	}
	buff.WriteRune(']')
	return buff.Bytes()
}
