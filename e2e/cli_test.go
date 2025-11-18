package e2e

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/skiff-sh/skiff/api/go/skiff/registry/v1alpha1"
	"github.com/skiff-sh/skiff/pkg/collection"
	"github.com/skiff-sh/skiff/pkg/fileutil"
	"github.com/skiff-sh/skiff/pkg/protoencode"
	"github.com/stretchr/testify/suite"
)

type CliTestSuite struct {
	suite.Suite
}

func (c *CliTestSuite) TestHelp() {
	type output struct {
		Stdout *bytes.Buffer
	}

	type test struct {
		Args         []string
		Setup        func() error
		ExpectedFunc func(o *output)
	}

	tests := map[string]test{
		"root help": {
			Args: []string{"--help"},
			ExpectedFunc: func(o *output) {
				fmt.Println(o.Stdout.String())
				c.NotEmpty(o.Stdout.String())
			},
		},
		"add help": {
			Args: []string{"add", "--help"},
			ExpectedFunc: func(o *output) {
				fmt.Println(o.Stdout.String())
				c.NotEmpty(o.Stdout.String())
			},
		},
		"add help with registry": {
			Args: []string{"add", filepath.Join(ExamplesPath(), "go-fiber-controller", ".skiff", ""), "--help"},
			ExpectedFunc: func(o *output) {
				fmt.Println(o.Stdout.String())
				c.NotEmpty(o.Stdout.String())
			},
		},
		"build help": {
			Args: []string{"build", "--help"},
			ExpectedFunc: func(o *output) {
				fmt.Println(o.Stdout.String())
				c.NotEmpty(o.Stdout.String())
			},
		},
	}

	for desc, v := range tests {
		c.Run(desc, func() {
			ctx := c.T().Context()
			cli, err := New()
			if !c.NoError(err) {
				return
			}

			buf := bytes.NewBuffer(nil)
			cli.Command.CLI.Writer = buf

			if v.Setup != nil {
				if !c.NoError(v.Setup()) {
					return
				}
			}

			err = cli.Command.Run(ctx, append([]string{"skiff"}, v.Args...))
			if !c.NoError(err) {
				return
			}

			v.ExpectedFunc(&output{
				Stdout: buf,
			})
		})
	}
}

func (c *CliTestSuite) TestBuild() {
	type params struct {
		// The directory housing the cloned example folder.
		ExampleDir    fs.FS
		ExampleFS     fileutil.MapFS
		BuildOutputFS fileutil.MapFS
		Actual        *BuildOutput
	}

	type test struct {
		Args        []string
		ExampleName string
		Expected    func(p *params)
		ExpectedErr string
	}

	tests := map[string]test{
		"go controller all files present": {
			ExampleName: "go-fiber-controller",
			Expected: func(p *params) {
				c.ElementsMatch([]string{"registry.json", "create-http-route.json"}, collection.Keys(p.BuildOutputFS))
				c.Len(p.Actual.Packages, 1)

				// Check that the contents are the same as the original.
				for _, pkg := range p.Actual.Packages {
					for _, fi := range pkg.Files {
						c.Equal(p.ExampleFS[filepath.Join(".skiff", fi.Path)].Data, fi.Content)
					}
				}
			},
		},
	}

	examples := os.DirFS(ExamplesPath())
	for desc, v := range tests {
		c.Run(desc, func() {
			exaDir, err := CloneExample(examples, v.ExampleName)
			if !c.NoError(err) {
				return
			}
			defer func() {
				_ = os.RemoveAll(exaDir)
			}()

			build, ok := c.buildExample(exaDir, v.Args...)
			if !ok {
				return
			}

			actualFS := fileutil.FlatMapFS(os.DirFS(build.OutputDir))
			actual, ok := c.unmarshalBuildOutput(actualFS)
			if !ok {
				return
			}

			exaFS := os.DirFS(exaDir)
			v.Expected(&params{
				ExampleDir:    exaFS,
				ExampleFS:     fileutil.FlatMapFS(exaFS),
				BuildOutputFS: actualFS,
				Actual:        actual,
			})
		})
	}
}

func (c *CliTestSuite) TestAdd() {
	type output struct {
	}

	type test struct {
		Args        func(buildOutputDir string) []string
		ExampleName string
		Inputs      string
		Expected    func(p *output)
		ExpectedErr string
	}

	tests := map[string]test{
		"help shows schema flags": {
			ExampleName: "go-fiber-controller",
			Args: func(buildOutputDir string) []string {
				return []string{filepath.Join(buildOutputDir, "create-http-route.json")}
			},
		},
		"go-fiber-controller example": {
			ExampleName: "go-fiber-controller",
			Args: func(buildOutputDir string) []string {
				return []string{filepath.Join(buildOutputDir, "create-http-route.json")}
			},
			Expected: func(p *output) {

			},
		},
	}

	examples := os.DirFS(ExamplesPath())
	for desc, v := range tests {
		c.Run(desc, func() {
			ctx := c.T().Context()
			exaDir, err := CloneExample(examples, v.ExampleName)
			if !c.NoError(err) {
				return
			}
			defer func() {
				_ = os.RemoveAll(exaDir)
			}()

			build, ok := c.buildExample(exaDir)
			if !ok {
				return
			}

			cmd, err := New()
			if !c.NoError(err) {
				return
			}

			output := bytes.NewBuffer(nil)
			input := bytes.NewBuffer(nil)

			tea.KeyDown.String()

			input.WriteString(v.Inputs)
			cmd.Command.CLI.Reader = input
			cmd.Command.CLI.Writer = output

			err = cmd.Command.Run(ctx, append([]string{"skiff", "add"}, v.Args(build.OutputDir)...))
			if !c.NoError(err) {
				return
			}

			v.Expected(&output{})
		})
	}
}

type BuildCmdOutput struct {
	OutputDir string
	Stdout    []byte
}

func (c *CliTestSuite) buildExample(exaDir string, args ...string) (*BuildCmdOutput, bool) {
	ctx := c.T().Context()

	cmd, err := New()
	if !c.NoError(err) {
		return nil, false
	}

	buf := bytes.NewBuffer(nil)
	cmd.Command.CLI.Writer = buf
	outputDir := filepath.Join(exaDir, "public", "r")

	err = cmd.Command.Run(ctx, slices.Concat([]string{"skiff", "build", filepath.Join(exaDir, ".skiff", "registry.json")}, slices.Concat(args, []string{"-o", outputDir})))
	if !c.NoError(err) {
		return nil, false
	}

	return &BuildCmdOutput{
		OutputDir: outputDir,
		Stdout:    buf.Bytes(),
	}, true
}

type BuildOutput struct {
	Registry *v1alpha1.Registry
	// Filename to contents
	Packages map[string]*v1alpha1.Package
}

func (c *CliTestSuite) unmarshalBuildOutput(f fileutil.MapFS) (*BuildOutput, bool) {
	out := &BuildOutput{
		Registry: new(v1alpha1.Registry),
		Packages: make(map[string]*v1alpha1.Package),
	}

	if !c.NotNil(f["registry.json"], "registry.json is missing") {
		return nil, false
	}

	err := protoencode.Unmarshaller.Unmarshal(f["registry.json"].Data, out.Registry)
	if !c.NoError(err, "failed to unmarshal the registry.json file") {
		return nil, false
	}

	for k, v := range f {
		if k == "registry.json" {
			continue
		}
		out.Packages[k] = new(v1alpha1.Package)
		err = protoencode.Unmarshaller.Unmarshal(v.Data, out.Packages[k])
		if !c.NoError(err, "failed to unmarshal package %s", k) {
			return nil, false
		}
	}

	return out, true
}

func TestE2ETestSuite(t *testing.T) {
	suite.Run(t, new(CliTestSuite))
}
