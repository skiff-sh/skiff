package schema

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/skiff-sh/config/ptr"
	"github.com/skiff-sh/skiff/api/go/skiff/registry/v1alpha1"
	"github.com/skiff-sh/skiff/pkg/bufferpool"
	"github.com/skiff-sh/skiff/pkg/interact"
	"github.com/stretchr/testify/suite"
)

type FormTestSuite struct {
	suite.Suite
}

func (f *FormTestSuite) TestFormField() {
	type test struct {
		Given          *v1alpha1.Field
		Input          string
		ExpectedOutput string
		ExpectedValue  any
	}

	tests := map[string]test{
		"string": {
			Given: &v1alpha1.Field{
				Name:        "field",
				Type:        ptr.Ptr(v1alpha1.Field_string),
				Description: ptr.Ptr("description"),
			},
			Input:         "derp",
			ExpectedValue: "derp",
		},
	}

	for desc, v := range tests {
		f.Run(desc, func() {
			fi, err := NewField(v.Given)
			if !f.NoError(err) {
				return
			}

			ctx := f.T().Context()
			ff := NewFormField(fi)
			input := strings.NewReader(v.Input)
			output := bufferpool.GetBytesBuffer()
			defer bufferpool.PutBytesBuffer(output)
			ctx, cancel := context.WithTimeout(ctx, 10000000*time.Millisecond)
			defer cancel()
			err = interact.NewHuhForm(interact.NewHuhGroup(ff.FormFields...)).WithInput(input).WithOutput(output).RunWithContext(ctx)
			if !f.NoError(err) {
				return
			}

			if v.ExpectedOutput != "" {
				f.Equal(v.ExpectedOutput, output.String())
			}

			f.Equal(v.ExpectedValue, ff.Value().Any())
		})
	}
}

func TestFormTestSuite(t *testing.T) {
	suite.Run(t, new(FormTestSuite))
}
