package schema

import (
	"fmt"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/skiff-sh/config/ptr"
	"github.com/stretchr/testify/suite"

	"github.com/skiff-sh/api/go/skiff/registry/v1alpha1"

	"github.com/skiff-sh/skiff/pkg/fields"
	"github.com/skiff-sh/skiff/pkg/interact"
	"github.com/skiff-sh/skiff/pkg/testutil"
	"github.com/skiff-sh/skiff/pkg/valid"
)

type FormTestSuite struct {
	suite.Suite
}

func (f *FormTestSuite) TestFormField() {
	type test struct {
		Given                 *v1alpha1.Field
		Input                 testutil.TeaInputs
		ExpectedOutput        string
		ExpectedValue         any
		ExpectedValidationErr string
		ExpectedNewSchemaErr  string
		ExpectedFormErr       string
	}

	tests := map[string]test{
		"string": {
			Given: &v1alpha1.Field{
				Name:        "field",
				Type:        ptr.Ptr(v1alpha1.Field_string),
				Description: ptr.Ptr("description"),
			},
			Input:         testutil.Inputs("derp", tea.KeyEnter),
			ExpectedValue: "derp",
		},
		"number": {
			Given: &v1alpha1.Field{
				Name: "field",
				Type: ptr.Ptr(v1alpha1.Field_number),
			},
			Input:         testutil.Inputs("123", tea.KeyEnter),
			ExpectedValue: float64(123),
		},
		"bool": {
			Given: &v1alpha1.Field{
				Name: "field",
				Type: ptr.Ptr(v1alpha1.Field_bool),
			},
			Input:         testutil.Inputs(tea.KeyRight, tea.KeyEnter),
			ExpectedValue: true,
		},
		"string select": {
			Given: &v1alpha1.Field{
				Name: "field",
				Type: ptr.Ptr(v1alpha1.Field_string),
				Enum: fields.NewListValue("a", "b", "c"),
			},
			Input:         testutil.Inputs(tea.KeyDown, tea.KeyEnter),
			ExpectedValue: "b",
		},
		"string select default": {
			Given: &v1alpha1.Field{
				Name:    "field",
				Type:    ptr.Ptr(v1alpha1.Field_string),
				Enum:    fields.NewListValue("a", "b", "c"),
				Default: fields.NewValue("c"),
			},
			Input:         testutil.Inputs(tea.KeyEnter),
			ExpectedValue: "c",
		},
		"string select bad default": {
			Given: &v1alpha1.Field{
				Name:    "field",
				Type:    ptr.Ptr(v1alpha1.Field_string),
				Enum:    fields.NewListValue("a", "b", "c"),
				Default: fields.NewValue("d"),
			},
			ExpectedValidationErr: "fields[0]: default value must be one of enum values when enum is set [default_must_be_in_enum]",
		},
		"enum types not matching": {
			Given: &v1alpha1.Field{
				Name:    "field",
				Type:    ptr.Ptr(v1alpha1.Field_string),
				Enum:    fields.NewListValue(1, 2, 3),
				Default: fields.NewValue(1),
			},
			ExpectedNewSchemaErr: "field 'field': 'default' value: got float64 but expected a string",
		},
		"string input": {
			Given: &v1alpha1.Field{
				Name: "field",
				Type: ptr.Ptr(v1alpha1.Field_string),
			},
			Input:         testutil.Inputs("123", tea.KeyEnter),
			ExpectedValue: "123",
		},
		"number input": {
			Given: &v1alpha1.Field{
				Name: "field",
				Type: ptr.Ptr(v1alpha1.Field_number),
			},
			Input:         testutil.Inputs("123", tea.KeyEnter),
			ExpectedValue: float64(123),
		},
		"number enum input": {
			Given: &v1alpha1.Field{
				Name: "field",
				Enum: fields.NewListValue(1, 2, 3),
				Type: ptr.Ptr(v1alpha1.Field_number),
			},
			Input:         testutil.Inputs(tea.KeyDown, tea.KeyDown, tea.KeyEnter),
			ExpectedValue: float64(3),
		},
		"number input invalid": {
			Given: &v1alpha1.Field{
				Name: "field",
				Type: ptr.Ptr(v1alpha1.Field_number),
			},
			Input:           testutil.Inputs("abc", tea.KeyEnter),
			ExpectedFormErr: "must be a number",
		},
		"list of strings": {
			Given: &v1alpha1.Field{
				Name: "field",
				Type: ptr.Ptr(v1alpha1.Field_array),
				Items: &v1alpha1.Field_SubField{
					Type: ptr.Ptr(v1alpha1.Field_string),
				},
			},
			Input:         testutil.Inputs("abc", "alt+enter", "def", tea.KeyEnter),
			ExpectedValue: []string{"abc", "def"},
		},
		"list of numbers": {
			Given: &v1alpha1.Field{
				Name: "field",
				Type: ptr.Ptr(v1alpha1.Field_array),
				Items: &v1alpha1.Field_SubField{
					Type: ptr.Ptr(v1alpha1.Field_number),
				},
			},
			Input:         testutil.Inputs("123", "alt+enter", "456", tea.KeyEnter),
			ExpectedValue: []float64{123, 456},
		},
		"list of invalid numbers": {
			Given: &v1alpha1.Field{
				Name: "field",
				Type: ptr.Ptr(v1alpha1.Field_array),
				Items: &v1alpha1.Field_SubField{
					Type: ptr.Ptr(v1alpha1.Field_number),
				},
			},
			Input:           testutil.Inputs("abc", "alt+enter", "def", tea.KeyEnter),
			ExpectedFormErr: "not a number",
		},
		"list of number enums": {
			Given: &v1alpha1.Field{
				Name: "field",
				Type: ptr.Ptr(v1alpha1.Field_array),
				Items: &v1alpha1.Field_SubField{
					Type: ptr.Ptr(v1alpha1.Field_number),
					Enum: fields.NewListValue(1, 2, 3),
				},
			},
			Input:         testutil.Inputs(tea.KeyDown, "x", tea.KeyDown, "x", tea.KeyEnter),
			ExpectedValue: []float64{2, 3},
		},
		"list of string enums": {
			Given: &v1alpha1.Field{
				Name: "field",
				Type: ptr.Ptr(v1alpha1.Field_array),
				Items: &v1alpha1.Field_SubField{
					Type: ptr.Ptr(v1alpha1.Field_string),
					Enum: fields.NewListValue("a", "b", "c"),
				},
			},
			Input:         testutil.Inputs(tea.KeyDown, "x", tea.KeyDown, "x", tea.KeyEnter),
			ExpectedValue: []string{"b", "c"},
		},
	}

	for desc, v := range tests {
		f.Run(desc, func() {
			ap := &v1alpha1.Schema{Fields: []*v1alpha1.Field{v.Given}}
			err := valid.ValidateProto(ap)
			if v.ExpectedValidationErr != "" || !f.NoError(err) {
				f.ErrorContains(err, v.ExpectedValidationErr)
				return
			}

			sch, err := NewSchema(ap)
			if v.ExpectedNewSchemaErr != "" || !f.NoError(err) {
				f.ErrorContains(err, v.ExpectedNewSchemaErr)
				return
			}

			ff := NewFormField(sch.Fields[0])
			form := huh.NewForm(interact.NewHuhGroup(ff.FormFields...))

			mod := teatest.NewTestModel(f.T(), form)
			v.Input.SendTo(mod, time.Millisecond)

			var waiter testutil.TeaWaitCond
			if v.ExpectedFormErr != "" {
				waiter = testutil.WaitRenderContains(v.ExpectedFormErr)
			} else {
				waiter = testutil.WaitFormDone(form)
			}
			teatest.WaitFor(
				f.T(),
				mod.Output(),
				waiter,
				teatest.WithCheckInterval(10*time.Millisecond),
				teatest.WithDuration(100*time.Millisecond),
			)

			if v.ExpectedValue == nil {
				f.Nil(ff.Value().Any())
			} else if !f.Equal(v.ExpectedValue, ff.Value().Any()) {
				fmt.Println(testutil.Dump(mod.Output()))
			}
		})
	}
}

func TestFormTestSuite(t *testing.T) {
	suite.Run(t, new(FormTestSuite))
}
