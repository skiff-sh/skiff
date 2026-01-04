package schema

import (
	"testing"

	"github.com/skiff-sh/config/ptr"
	"github.com/stretchr/testify/suite"
	"github.com/urfave/cli/v3"

	"github.com/skiff-sh/api/go/skiff/registry/v1alpha1"

	"github.com/skiff-sh/skiff/pkg/fields"
)

type FlagsTestSuite struct {
	suite.Suite
}

func (f *FlagsTestSuite) TestFieldToCLIFlag() {
	type test struct {
		Given        *v1alpha1.Field
		ExpectedFunc func(v *Flag)
	}

	tests := map[string]test{
		"number": {
			Given: &v1alpha1.Field{
				Name: "field",
				Type: ptr.Ptr(v1alpha1.Field_number),
			},
			ExpectedFunc: func(v *Flag) {
				f.IsType(&cli.Float64Flag{}, v.Flag)
			},
		},
		"number enum": {
			Given: &v1alpha1.Field{
				Name: "field",
				Type: ptr.Ptr(v1alpha1.Field_number),
				Enum: fields.NewListValue(1, 2, 3),
			},
			ExpectedFunc: func(v *Flag) {
				f.NoError(v.Flag.(*cli.Float64Flag).Action(f.T().Context(), nil, 2))
			},
		},
		"number enum list": {
			Given: &v1alpha1.Field{
				Name: "field",
				Type: ptr.Ptr(v1alpha1.Field_array),
				Items: &v1alpha1.Field_SubField{
					Type: ptr.Ptr(v1alpha1.Field_number),
					Enum: fields.NewListValue(1, 2, 3),
				},
			},
			ExpectedFunc: func(v *Flag) {
				f.NoError(v.Flag.(*cli.Float64SliceFlag).Action(f.T().Context(), nil, []float64{2, 3}))
			},
		},
		"number enum list invalid": {
			Given: &v1alpha1.Field{
				Name: "field",
				Type: ptr.Ptr(v1alpha1.Field_array),
				Items: &v1alpha1.Field_SubField{
					Type: ptr.Ptr(v1alpha1.Field_number),
					Enum: fields.NewListValue(1, 2, 3),
				},
			},
			ExpectedFunc: func(v *Flag) {
				f.ErrorContains(
					v.Flag.(*cli.Float64SliceFlag).Action(f.T().Context(), nil, []float64{4, 5}),
					"expected one of 1, 2, 3",
				)
			},
		},
		"number enum invalid": {
			Given: &v1alpha1.Field{
				Name: "field",
				Type: ptr.Ptr(v1alpha1.Field_number),
				Enum: fields.NewListValue(1, 2, 3),
			},
			ExpectedFunc: func(v *Flag) {
				f.ErrorContains(v.Flag.(*cli.Float64Flag).Action(f.T().Context(), nil, 4), "expected one of 1, 2, 3")
			},
		},
		"string": {
			Given: &v1alpha1.Field{
				Name: "field",
				Type: ptr.Ptr(v1alpha1.Field_string),
			},
			ExpectedFunc: func(v *Flag) {
				f.IsType(&cli.StringFlag{}, v.Flag)
			},
		},
		"string enum": {
			Given: &v1alpha1.Field{
				Name: "field",
				Type: ptr.Ptr(v1alpha1.Field_string),
				Enum: fields.NewListValue("a", "b", "c"),
			},
			ExpectedFunc: func(v *Flag) {
				f.NoError(v.Flag.(*cli.StringFlag).Action(f.T().Context(), nil, "b"))
			},
		},
		"string enum invalid": {
			Given: &v1alpha1.Field{
				Name: "field",
				Type: ptr.Ptr(v1alpha1.Field_string),
				Enum: fields.NewListValue("a", "b", "c"),
			},
			ExpectedFunc: func(v *Flag) {
				f.ErrorContains(v.Flag.(*cli.StringFlag).Action(f.T().Context(), nil, "d"), "expected one of a, b, c")
			},
		},
		"string enum list": {
			Given: &v1alpha1.Field{
				Name: "field",
				Type: ptr.Ptr(v1alpha1.Field_array),
				Items: &v1alpha1.Field_SubField{
					Type: ptr.Ptr(v1alpha1.Field_string),
					Enum: fields.NewListValue("a", "b", "c"),
				},
			},
			ExpectedFunc: func(v *Flag) {
				f.NoError(v.Flag.(*cli.StringSliceFlag).Action(f.T().Context(), nil, []string{"b", "c"}))
			},
		},
		"string enum list invalid": {
			Given: &v1alpha1.Field{
				Name: "field",
				Type: ptr.Ptr(v1alpha1.Field_array),
				Items: &v1alpha1.Field_SubField{
					Type: ptr.Ptr(v1alpha1.Field_string),
					Enum: fields.NewListValue("a", "b", "c"),
				},
			},
			ExpectedFunc: func(v *Flag) {
				f.ErrorContains(
					v.Flag.(*cli.StringSliceFlag).Action(f.T().Context(), nil, []string{"d"}),
					"expected one of a, b, c",
				)
			},
		},
		"bool": {
			Given: &v1alpha1.Field{
				Name: "field",
				Type: ptr.Ptr(v1alpha1.Field_bool),
			},
			ExpectedFunc: func(v *Flag) {
				f.IsType(&cli.BoolFlag{}, v.Flag)
			},
		},
	}

	for desc, v := range tests {
		f.Run(desc, func() {
			sch, err := New(&v1alpha1.Schema{Fields: []*v1alpha1.Field{v.Given}})
			if !f.NoError(err) {
				return
			}

			actual := FieldToCLIFlag(sch.Fields[0])
			v.ExpectedFunc(actual)
		})
	}
}

func TestFlagsTestSuite(t *testing.T) {
	suite.Run(t, new(FlagsTestSuite))
}
