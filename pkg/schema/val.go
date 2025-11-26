package schema

import (
	"github.com/skiff-sh/api/go/skiff/registry/v1alpha1"
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
	if f.GetItems() != nil {
		items = f.GetItems().Type
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

type value struct {
	Val       any
	Typ       v1alpha1.Field_Type
	ItemsType *v1alpha1.Field_Type
}

func (v *value) Any() any {
	if v == nil {
		return nil
	}
	return v.Val
}

func (v *value) String() string {
	if v == nil {
		return ""
	}
	o, _ := v.Val.(string)
	return o
}

func (v *value) Number() float64 {
	if v == nil {
		return 0
	}
	o, _ := v.Val.(float64)
	return o
}

func (v *value) Bool() bool {
	if v == nil {
		return false
	}
	o, _ := v.Val.(bool)
	return o
}

func (v *value) Strings() []string {
	if v == nil {
		return nil
	}
	return collection.Map(fields.Cast[[]any](v.Val), fields.Cast[string])
}

func (v *value) Numbers() []float64 {
	if v == nil {
		return nil
	}
	return collection.Map(fields.Cast[[]any](v.Val), fields.Cast[float64])
}

func (v *value) Type() v1alpha1.Field_Type {
	if v == nil {
		return v1alpha1.Field_string
	}
	return v.Typ
}

func (v *value) Items() *v1alpha1.Field_Type {
	if v == nil {
		return nil
	}
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
