package schema

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/skiff-sh/skiff/api/go/skiff/registry/v1alpha1"
	"github.com/skiff-sh/skiff/pkg/collection"
	"github.com/skiff-sh/skiff/pkg/fields"
)

var _ Entry = (*FormField)(nil)

type FormField struct {
	FormFields  []huh.Field
	Accessor    HuhValueAccessor
	SchemaField *Field
}

func (h *FormField) FieldName() string {
	return h.SchemaField.Proto.Name
}

func (h *FormField) Value() Value {
	return h.Accessor.Value()
}

type HuhValueAccessor interface {
	Key() string
	SetKey(s string)
	SetDescription(s string)
	SetTitle(s string)
	Description() string
	Value() Value
}

func newSelectHuhAccessor[T comparable](sel *huh.Select[T], f *Field) HuhValueAccessor {
	return &huhValueAccessor{
		Field: sel,
		KeySetter: func(key string) {
			sel.Key(key)
		},
		DescriptionSetter: func(desc string) {
			sel.Description(desc)
		},
		TitleSetter: func(title string) {
			sel.Title(title)
		},
		ValueSource: ValueSourceFunc(func() Value {
			if f.Proto.GetType() == v1alpha1.Field_number {
				val := sel.GetValue()
				switch typ := val.(type) {
				case float64:
					return NewValidatedVal(typ, f.Proto.GetType(), nil)
				case string:
					fl, _ := strconv.ParseFloat(sel.GetValue().(string), 64)
					return NewValidatedVal(fl, f.Proto.GetType(), nil)
				}
			}
			return NewValidatedVal(sel.GetValue(), f.Proto.GetType(), nil)
		}),
	}
}

func newInputHuhAccessor(in *huh.Input, typ v1alpha1.Field_Type) HuhValueAccessor {
	return &huhValueAccessor{
		Field: in,
		KeySetter: func(key string) {
			in.Key(key)
		},
		DescriptionSetter: func(desc string) {
			in.Description(desc)
		},
		TitleSetter: func(title string) {
			in.Title(title)
		},
		ValueSource: ValueSourceFunc(func() Value {
			val := in.GetValue()
			if typ == v1alpha1.Field_number {
				fl, err := strconv.ParseFloat(in.GetValue().(string), 64)
				if err != nil {
					val = nil
				} else {
					val = fl
				}
			}
			return NewValidatedVal(val, typ, nil)
		}),
	}
}

func newMultiSelectHuhAccessor[T comparable](h *huh.MultiSelect[T], fi *Field) HuhValueAccessor {
	return &huhValueAccessor{
		Field: h,
		KeySetter: func(key string) {
			h.Key(key)
		},
		DescriptionSetter: func(desc string) {
			h.Description(desc)
		},
		TitleSetter: func(title string) {
			h.Title(title)
		},
		ValueSource: ValueSourceFunc(func() Value {
			vals := h.GetValue().([]T)
			return NewValidatedValFromField(vals, fi.Proto)
		}),
	}
}

func newConfirmHuhAccessor(h *huh.Confirm) HuhValueAccessor {
	return &huhValueAccessor{
		Field: h,
		KeySetter: func(key string) {
			h.Key(key)
		},
		DescriptionSetter: func(desc string) {
			h.Description(desc)
		},
		TitleSetter: func(title string) {
			h.Title(title)
		},
		ValueSource: ValueSourceFunc(func() Value {
			return NewValidatedVal(h.GetValue(), v1alpha1.Field_bool, nil)
		}),
	}
}

func newTextHuhAccessor(txt *huh.Text, itemsTyp v1alpha1.Field_Type) HuhValueAccessor {
	return &huhValueAccessor{
		Field: txt,
		KeySetter: func(key string) {
			txt.Key(key)
		},
		DescriptionSetter: func(desc string) {
			txt.Description(desc)
		},
		TitleSetter: func(title string) {
			txt.Title(title)
		},
		ValueSource: ValueSourceFunc(func() Value {
			var val any
			lines := strings.Split(txt.GetValue().(string), "\n")
			if itemsTyp == v1alpha1.Field_number {
				val, _ = collection.MapOrErr(lines, fields.ParseFloat[float64])
			} else {
				val = lines
			}
			return NewValidatedVal(val, v1alpha1.Field_array, &itemsTyp)
		}),
	}
}

type huhValueAccessor struct {
	Field             huh.Field
	DescriptionSetter func(s string)
	Descript          string
	KeySetter         func(key string)
	TitleSetter       func(title string)
	ValueSource       ValueSource
}

func (h *huhValueAccessor) Description() string {
	return h.Descript
}

func (h *huhValueAccessor) SetDescription(s string) {
	h.Descript = s
	h.DescriptionSetter(s)
}

func (h *huhValueAccessor) SetTitle(s string) {
	h.TitleSetter(s)
}

func (h *huhValueAccessor) Key() string {
	return h.Field.GetKey()
}

func (h *huhValueAccessor) SetKey(s string) {
	h.KeySetter(s)
}

func (h *huhValueAccessor) Value() Value {
	return h.ValueSource.Value()
}

func FlattenHuhFields(fields []*FormField) []huh.Field {
	out := make([]huh.Field, 0, len(fields))
	for _, v := range fields {
		out = append(out, v.FormFields...)
	}
	return out
}

func NewFormField(f *Field) *FormField {
	out := &FormField{
		SchemaField: f,
	}

	if len(f.Enum) > 0 {
		// Need to get the underlying type.
		typ := f.Proto.GetType()
		if typ == v1alpha1.Field_array {
			typ = f.Proto.GetItems().GetType()
		}

		switch typ {
		case v1alpha1.Field_string:
			sel, accessor := newSelect[string](f)
			if sel != nil {
				out.FormFields = append(out.FormFields, sel)
				out.Accessor = accessor
			}
		case v1alpha1.Field_number:
			sel, accessor := newSelect[float64](f)
			if sel != nil {
				out.FormFields = append(out.FormFields, sel)
				out.Accessor = accessor
			}
		}

		return out
	}

	switch f.Proto.GetType() {
	case v1alpha1.Field_string, v1alpha1.Field_number, v1alpha1.Field_bool:
		o, getter := newPrimitive(f.Proto.GetType())
		if o == nil {
			return out
		}
		out.FormFields = append(out.FormFields, o)
		out.Accessor = getter
	case v1alpha1.Field_array:
		var val string
		valParser := stringParser
		if f.Proto.GetItems().GetType() == v1alpha1.Field_number {
			valParser = floatParser
		}
		txt := huh.NewText().
			Lines(5).
			ShowLineNumbers(true).
			Value(&val).
			Validate(func(s string) error {
				_, err := parseStringList(s, valParser)
				return err
			})

		out.Accessor = newTextHuhAccessor(txt, f.Proto.GetItems().GetType())
		out.FormFields = append(out.FormFields, txt)

		out.Accessor.SetDescription("One line per entry.")
	}

	return out
}

type valueParser func(s string) (any, error)

var (
	floatParser valueParser = func(s string) (any, error) {
		fl, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return nil, errors.New("not a number")
		}
		return fl, nil
	}

	stringParser valueParser = func(s string) (any, error) {
		return s, nil
	}
)

func parseStringList(s string, parser valueParser) ([]any, error) {
	lines := strings.Split(s, "\n")
	out := make([]any, 0, len(lines))
	for i, v := range lines {
		li, err := parser(v)
		if err != nil {
			return nil, fmt.Errorf("entry #%d invalid: %w", i+1, err)
		}
		out = append(out, li)
	}

	return out, nil
}

func newPrimitive(typ v1alpha1.Field_Type) (huh.Field, HuhValueAccessor) {
	switch typ {
	case v1alpha1.Field_string:
		var val string
		out := huh.NewInput().
			Value(&val)
		return out, newInputHuhAccessor(out, typ)
	case v1alpha1.Field_number:
		var val string
		out := huh.NewInput().
			Value(&val).
			Validate(func(s string) error {
				_, err := strconv.ParseFloat(s, 64)
				if err != nil {
					return errors.New("must be a number")
				}
				return nil
			})
		return out, newInputHuhAccessor(out, typ)
	case v1alpha1.Field_bool:
		var val bool
		out := huh.NewConfirm().Value(&val)
		return out, newConfirmHuhAccessor(out)
	}
	return nil, nil
}

func newSelect[T string | float64](f *Field) (huh.Field, HuhValueAccessor) {
	var primitiveVal T
	anyStringer := newAnyStringer(primitiveVal)
	switch f.Proto.GetType() {
	case v1alpha1.Field_string, v1alpha1.Field_number:
		out := huh.NewSelect[T]().
			Value(&primitiveVal).
			Options(collection.Map(f.Enum, func(e any) huh.Option[T] {
				val := fields.Cast[T](e)
				return huh.NewOption(anyStringer(e), val).Selected(e == f.Default)
			})...)
		return out, newSelectHuhAccessor(out, f)
	case v1alpha1.Field_array:
		var val []T
		def := fields.Cast[[]any](f.Default)
		out := huh.NewMultiSelect[T]().
			Value(&val).
			Options(collection.Map(f.Enum, func(e any) huh.Option[T] {
				val := fields.Cast[T](e)
				return huh.NewOption(anyStringer(e), val).Selected(slices.Contains(def, e))
			})...)
		return out, newMultiSelectHuhAccessor(out, f)
	}
	return nil, nil
}

func newAnyStringer[T string | float64](t T) func(e any) string {
	switch any(t).(type) {
	case float64:
		return func(e any) string {
			return fields.FormatFloat(fields.Cast[float64](e))
		}
	}
	return func(e any) string {
		return fields.Cast[string](e)
	}
}
