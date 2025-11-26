package schema

import (
	"errors"
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/skiff-sh/api/go/skiff/registry/v1alpha1"
	"github.com/skiff-sh/skiff/pkg/fields"
)

type Schema struct {
	Proto  *v1alpha1.Schema
	Fields []*Field
}

func NewSchema(sch *v1alpha1.Schema) (*Schema, error) {
	out := &Schema{
		Proto:  sch,
		Fields: make([]*Field, 0, len(sch.GetFields())),
	}

	for _, field := range sch.GetFields() {
		f, err := NewField(field)
		if err != nil {
			return nil, fmt.Errorf("field '%s': %w", field.GetName(), err)
		}

		f.Default, err = getDefault(field)
		if err != nil {
			return nil, fmt.Errorf("field '%s': %w", field.GetName(), err)
		}

		var enums []*structpb.Value
		var expectedType v1alpha1.Field_Type
		if field.GetType() == v1alpha1.Field_array {
			expectedType = field.GetItems().GetType()
			enums = field.GetItems().GetEnum().GetValues()
		} else {
			expectedType = field.GetType()
			enums = field.GetEnum().GetValues()
		}

		if len(enums) > 0 {
			f.Enum = make([]any, 0, len(enums))
			for i, v := range enums {
				val, err := primitiveAs(v, expectedType)
				if err != nil {
					return nil, fmt.Errorf("field '%s': enum value %v (#%d): %w", field.GetName(), v, i, err)
				}

				f.Enum = append(f.Enum, val)
			}
		}

		out.Fields = append(out.Fields, f)
	}

	return out, nil
}

type Field struct {
	Proto *v1alpha1.Field
	// Set if the default field is present.
	Default any
	// Set if the enum field is present.
	Enum []any
}

func NewField(p *v1alpha1.Field) (*Field, error) {
	out := &Field{
		Proto:   p,
		Default: p.GetDefault().AsInterface(),
		Enum:    p.GetEnum().AsSlice(),
	}

	return out, nil
}

func getDefault(p *v1alpha1.Field) (any, error) {
	if p.GetDefault() == nil {
		return nil, nil
	}
	switch p.GetType() {
	case v1alpha1.Field_string, v1alpha1.Field_number, v1alpha1.Field_bool:
		v, err := primitiveAs(p.GetDefault(), p.GetType())
		if err != nil {
			return nil, fmt.Errorf("'default' value: %w", err)
		}
		return v, nil
	case v1alpha1.Field_array:
		listVal := p.GetDefault().GetListValue()
		if listVal == nil {
			return nil, fmt.Errorf("expected 'default' to be array got %T", p.GetDefault().AsInterface())
		}
		vals := listVal.GetValues()
		raw := make([]any, 0, len(vals))
		for i, val := range vals {
			o, err := primitiveAs(val, p.GetItems().GetType())
			if err != nil {
				return nil, fmt.Errorf("'default' value #%d: %w", i, err)
			}
			raw = append(raw, o)
		}
		return raw, nil
	}
	return nil, fmt.Errorf("'default' value: unknown type %s", p.GetType().String())
}

func primitiveAs(val *structpb.Value, typ v1alpha1.Field_Type) (any, error) {
	i := val.AsInterface()
	//nolint:exhaustive // can only be a primitive.
	switch typ {
	case v1alpha1.Field_string:
		v, ok := fields.As[string](i)
		if !ok {
			return nil, fmt.Errorf("got %T but expected a string", i)
		}
		return v, nil
	case v1alpha1.Field_number:
		v, ok := fields.As[float64](i)
		if !ok {
			return nil, fmt.Errorf("got %T but expected a number", i)
		}
		return v, nil
	case v1alpha1.Field_bool:
		v, ok := fields.As[bool](i)
		if !ok {
			return nil, fmt.Errorf("got %T but expected a bool", i)
		}
		return v, nil
	}
	return nil, errors.New("expected string, number, or bool")
}
