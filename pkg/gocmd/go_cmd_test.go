package gocmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/skiff-sh/skiff/pkg/execcmd"
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
				cmd.Buffers.Stdout = bytes.NewBufferString(v.Given)
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
		Given           BuildArgs
		ExpectedArgs    []string
		ExpectedEnvVars map[string]string
	}

	tests := map[string]test{
		"all args": {
			ExpectedArgs: []string{"go", "build", "-buildmode=c-shared", "-o", "./derp.wasm", "./plugin.go"},
			ExpectedEnvVars: map[string]string{
				"GOOS":   "wasip1",
				"GOARCH": "wasm",
			},
			Given: BuildArgs{
				BuildMode:  BuildModeCShared,
				OutputPath: "./derp.wasm",
				Packages:   []string{"./plugin.go"},
				GoOS:       OSWASIP1,
				GoArch:     ArchWASM,
			},
		},
	}

	for desc, v := range tests {
		g.Run(desc, func() {
			var actualCmd *execcmd.Cmd
			execcmd.DefaultRunner = execcmd.RunnerFunc(func(cmd *execcmd.Cmd) error {
				actualCmd = cmd
				return nil
			})

			gocmd, err := New()
			if !g.NoError(err) {
				return
			}

			ctx := g.T().Context()
			_, err = gocmd.Build(ctx, v.Given)
			if !g.NoError(err) {
				return
			}

			g.Equal(v.ExpectedArgs, actualCmd.Cmd.Args)
			g.Equal(v.ExpectedEnvVars, execcmd.EnvVarsToMap(actualCmd.Cmd.Env))
		})
	}
}

func TestGoCmdTestSuite(t *testing.T) {
	suite.Run(t, new(GoCmdTestSuite))
}
