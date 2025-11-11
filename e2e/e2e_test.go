package e2e

import (
	"bytes"
	"fmt"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/suite"
)

type E2ETestSuite struct {
	suite.Suite
}

func (e *E2ETestSuite) TestHelp() {
	// -- Given
	//
	ctx := e.T().Context()
	cli, err := New()
	if !e.NoError(err) {
		return
	}

	buf := bytes.NewBuffer(nil)
	cli.Command.Writer = buf

	// -- When
	//
	err = cli.Command.Run(ctx, []string{"skiff", "build", "--help"})

	// -- Then
	//
	if !e.NoError(err) {
		return
	}

	fmt.Println(buf.String())

	e.NotEmpty(buf.String())
}

func (e *E2ETestSuite) TestBuild() {
	type test struct {
		Args           []string
		ExampleName    string
		ExpectedOutput string
		ExpectedErr    string
	}

	tests := map[string]test{}

	for desc, v := range tests {
		e.Run(desc, func() {
			ctx := e.T().Context()
			exaDir, err := CloneExample("go-fiber-controller")
			if !e.NoError(err) {
				return
			}

			cmd, err := New()
			if !e.NoError(err) {
				return
			}

			buf := bytes.NewBuffer(nil)
			cmd.Command.Writer = buf

			err = cmd.Command.Run(ctx, slices.Concat([]string{"skiff", "build", filepath.Join(exaDir)}, v.Args))
			if v.ExpectedErr != "" || !e.NoError(err) {
				e.EqualError(err, v.ExpectedErr)
				return
			}

			if v.ExpectedOutput != "" {
				e.Equal(v.ExpectedOutput, buf.String())
			}
		})
	}
}

func TestE2ETestSuite(t *testing.T) {
	suite.Run(t, new(E2ETestSuite))
}
