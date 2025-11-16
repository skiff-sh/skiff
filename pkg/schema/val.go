package schema

import (
	"fmt"

	"github.com/skiff-sh/config/ptr"
	"github.com/skiff-sh/skiff/api/go/skiff/registry/v1alpha1"
	"github.com/skiff-sh/skiff/pkg/collection"
	"github.com/skiff-sh/skiff/pkg/fields"
)

type Value interface {
	Any() any
	String() string
	Number() float64
	Bool() bool
	Strings() []string
	Numbers() []float64
	Type() v1alpha1.Field_Type
	Items() *v1alpha1.Field_Type
}

type PrimitiveType interface {
	string | float64 | bool
}

type SliceType interface {
	[]string | []float64
}

type ValueType interface {
	PrimitiveType | SliceType
}

func NewValidatedValFromField(val any, f *v1alpha1.Field) Value {
	var items *v1alpha1.Field_Type
	if f.Items != nil {
		items = f.Items.Type
	}
	return NewValidatedVal(val, f.GetType(), items)
}

func NewValidatedVal(a any, typ v1alpha1.Field_Type, itemsTyp *v1alpha1.Field_Type) Value {
	return &value{
		Val:       a,
		Typ:       typ,
		ItemsType: itemsTyp,
	}
}

func NewVal(a any) (Value, error) {
	out := &value{
		Val: a,
	}
	switch a.(type) {
	case string:
		out.Typ = v1alpha1.Field_string
	case float64:
		out.Typ = v1alpha1.Field_number
	case bool:
		out.Typ = v1alpha1.Field_bool
	case []string:
		out.Typ = v1alpha1.Field_array
		out.ItemsType = ptr.Ptr(v1alpha1.Field_string)
	case []float64:
		out.Typ = v1alpha1.Field_array
		out.ItemsType = ptr.Ptr(v1alpha1.Field_number)
	default:
		return nil, fmt.Errorf("%T is not a supported type", a)
	}

	return out, nil
}

type value struct {
	Val       any
	Typ       v1alpha1.Field_Type
	ItemsType *v1alpha1.Field_Type
}

func (v *value) Any() any {
	return v.Val
}

func (v *value) String() string {
	return fields.Cast[string](v.Val)
}

func (v *value) Number() float64 {
	return fields.Cast[float64](v.Val)
}

func (v *value) Bool() bool {
	return fields.Cast[bool](v.Val)
}

func (v *value) Strings() []string {
	return collection.Map(fields.Cast[[]any](v.Val), fields.Cast[string])
}

func (v *value) Numbers() []float64 {
	return collection.Map(fields.Cast[[]any](v.Val), fields.Cast[float64])
}

func (v *value) Type() v1alpha1.Field_Type {
	return v.Typ
}

func (v *value) Items() *v1alpha1.Field_Type {
	return v.ItemsType
}

type ValueSource interface {
	Value() Value
}

var _ ValueSource = (ValueSourceFunc)(nil)

type ValueSourceFunc func() Value

func (v ValueSourceFunc) Value() Value {
	return v()
}
