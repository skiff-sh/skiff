package gocmd

import (
	"bytes"
	"testing"

	"github.com/skiff-sh/skiff/pkg/execcmd"
	"github.com/stretchr/testify/suite"
)

type GoCmdTestSuite struct {
	suite.Suite
}

func (g *GoCmdTestSuite) TestVersion() {
	type test struct {
		Given    string
		Expected *Version
	}

	tests := map[string]test{
		"basic": {
			Given: "go version go1.25.4 darwin/arm64",
			Expected: &Version{
				Major: 1,
				Minor: 25,
				Patch: 4,
			},
		},
	}

	for desc, v := range tests {
		g.Run(desc, func() {
			execcmd.DefaultRunner = execcmd.RunnerFunc(func(cmd *execcmd.Cmd) error {
				cmd.Buffers.Stdout = bytes.NewBuffer([]byte(v.Given))
				return nil
			})

			gocmd, err := New()
			if !g.NoError(err) {
				return
			}

			ver, err := gocmd.Version(g.T().Context())
			if !g.NoError(err) {
				return
			}

			g.Equal(v.Expected, ver)
		})
	}
}

func (g *GoCmdTestSuite) TestBuild() {
	type test struct {
		Given        BuildArgs
		ExpectedArgs []string
	}

	tests := map[string]test{
		"all args": {},
	}

	for desc, v := range tests {
		g.Run(desc, func() {
			execcmd.DefaultRunner = execcmd.RunnerFunc(func(cmd *execcmd.Cmd) error {
				cmd.Buffers.Stdout = bytes.NewBuffer([]byte(v.Given))
				return nil
			})

			gocmd, err := New()
			if !g.NoError(err) {
				return
			}

			ver, err := gocmd.Version(g.T().Context())
			if !g.NoError(err) {
				return
			}

			g.Equal(v.Expected, ver)
		}
	}
}

func TestGoCmdTestSuite(t *testing.T) {
	suite.Run(t, new(GoCmdTestSuite))
}
