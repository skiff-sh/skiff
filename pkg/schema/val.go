package schema

import (
	"errors"

	pluginv1alpha1 "github.com/skiff-sh/api/go/skiff/plugin/v1alpha1"
	"github.com/skiff-sh/api/go/skiff/registry/v1alpha1"
	"github.com/skiff-sh/config/ptr"
	"github.com/spf13/cast"

	"github.com/skiff-sh/skiff/pkg/collection"
	"github.com/skiff-sh/skiff/pkg/fields"
)

// Value operates similar to the structpb.Value type but is restricted to a subset of values:
// * float64
// * bool
// * string
// * []string
// * []float64.
type Value interface {
	Any() any
	Plugin() *pluginv1alpha1.Value
	String() string
	Number() float64
	Bool() bool
	Strings() []string
	Numbers() []float64
	Type() v1alpha1.Field_Type
	Items() *v1alpha1.Field_Type
}

var ErrUnsupportedType = errors.New("invalid type")

type PrimitiveType interface {
	string | float64 | bool
}

type SliceType interface {
	[]string | []float64
}

type ValueType interface {
	PrimitiveType | SliceType
}

func NewVal(v any) (Value, error) {
	var t v1alpha1.Field_Type
	var itemsTyp *v1alpha1.Field_Type

	switch v.(type) {
	case string:
		t = v1alpha1.Field_string
	case bool:
		t = v1alpha1.Field_bool
	case float64, float32, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, complex64, complex128, uintptr:
		t = v1alpha1.Field_number
		v = cast.ToFloat64(v)
	case []string:
		t = v1alpha1.Field_array
		itemsTyp = ptr.Ptr(v1alpha1.Field_string)
	case []float64, []float32, []int, []int8, []int16, []int32, []int64, []uint, []uint8, []uint16, []uint32, []uint64, []complex64, []complex128, []uintptr:
		t = v1alpha1.Field_number
		itemsTyp = ptr.Ptr(v1alpha1.Field_number)
		v = cast.ToFloat64Slice(v)
	default:
		return nil, ErrUnsupportedType
	}

	return NewValidatedVal(v, t, itemsTyp), nil
}

// NewValidatedValFromField is a convenience func for NewValidatedVal.
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

func (v *value) Plugin() *pluginv1alpha1.Value {
	out := &pluginv1alpha1.Value{}

	switch v.Typ {
	case v1alpha1.Field_string:
		out.String = ptr.Ptr(v.String())
	case v1alpha1.Field_bool:
		out.Bool = ptr.Ptr(v.Bool())
	case v1alpha1.Field_number:
		out.Number = ptr.Ptr(v.Number())
	case v1alpha1.Field_array:
		out.List = &pluginv1alpha1.ValueList{}
	}

	if v.ItemsType != nil {
		it := *v.ItemsType
		if it == v1alpha1.Field_number {
			out.List.Numbers = v.Numbers()
		} else {
			out.List.Strings = v.Strings()
		}
	}

	return out
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
