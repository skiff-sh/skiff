package e2e

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/stretchr/testify/suite"

	"github.com/skiff-sh/skiff/pkg/filesystem"

	"github.com/skiff-sh/api/go/skiff/registry/v1alpha1"

	"github.com/skiff-sh/skiff/pkg/collection"
	"github.com/skiff-sh/skiff/pkg/fileutil"
	"github.com/skiff-sh/skiff/pkg/interact"
	"github.com/skiff-sh/skiff/pkg/protoencode"
	"github.com/skiff-sh/skiff/pkg/testutil"
)

type CliTestSuite struct {
	Suite
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
					for _, fi := range pkg.GetFiles() {
						c.NotEmpty(string(p.ExampleFS[filepath.Join(".skiff", fi.GetPath())].Data))
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

			defer c.SetWd(exaDir)()

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
		Build           *BuildCmdOutput
		TestModel       *teatest.TestModel
		Form            *huh.Form
		FinalOutput     string
		OriginalExample filesystem.Filesystem
		BuildRoot       filesystem.Filesystem
		Err             error
	}

	type test struct {
		Args func(b *BuildCmdOutput) []string
		// Each list of inputs corresponds to a different form.
		Inputs      []testutil.TeaInputs
		Expected    func(p *output)
		ExpectedErr string
	}

	tests := map[string]test{
		"input all data interactive": {
			Args: func(b *BuildCmdOutput) []string {
				return []string{"--root", b.RootDir, filepath.Join(b.OutputDir, "create-http-route.json")}
			},
			Inputs: []testutil.TeaInputs{
				testutil.Inputs(tea.KeyLeft, tea.KeyEnter), // Grant access.
				testutil.Inputs(
					"derp", tea.KeyEnter, // provide the name
					tea.KeyDown, tea.KeyEnter, // provide the method
					"/derp", tea.KeyEnter, // provide the path
				),
				testutil.Inputs("y", tea.KeyEnter), // create file
				testutil.Inputs("y", tea.KeyEnter), // create file
			},
			Expected: func(p *output) {
				if !c.NoError(p.Err) {
					return
				}

				c.FileContainsAll(p.BuildRoot, filepath.Join("controller", "derp.go"), []string{"POST", "/derp"})
				c.FileContains(
					p.BuildRoot,
					filepath.Join("controller", "controller.go"),
					"var Controllers = []Controller{new(Hello), new(DerpController)}",
				)
			},
		},
		"non interactive forces flags to be required": {
			Args: func(b *BuildCmdOutput) []string {
				return []string{
					"--root",
					b.RootDir,
					"-p",
					"all",
					"--non-i",
					filepath.Join(b.OutputDir, "create-http-route.json"),
				}
			},
			Expected: func(p *output) {
				c.ErrorContains(p.Err, "Required flags")
			},
		},
		"no op if access is denied": {
			Args: func(b *BuildCmdOutput) []string {
				return []string{"--root", b.RootDir, filepath.Join(b.OutputDir, "create-http-route.json")}
			},
			Inputs: []testutil.TeaInputs{
				testutil.Inputs(tea.KeyEnter),
			},
			Expected: func(p *output) {
				if !c.NoError(p.Err) {
					return
				}

				fp := filepath.Join("controller", "controller.go")
				c.EqualFiles(p.OriginalExample, fp, p.BuildRoot, fp)
			},
		},
	}

	for desc, v := range tests {
		c.Run(desc, func() {
			examples := os.DirFS(ExamplesPath())
			ctx := c.T().Context()
			exaDir, err := CloneExample(examples, "go-fiber-controller")
			if !c.NoError(err) {
				return
			}
			defer func() {
				_ = os.RemoveAll(exaDir)
			}()

			defer c.SetWd(exaDir)()

			build, ok := c.buildExample(exaDir)
			if !ok {
				return
			}

			cmd, err := New()
			if !c.NoError(err) {
				return
			}

			var mod *teatest.TestModel
			var form *huh.Form
			var formIdx int
			formOutput := bytes.NewBuffer(nil)
			interact.DefaultFormRunner = func(_ context.Context, f *huh.Form) error {
				form = f
				mod = teatest.NewTestModel(c.T(), f)
				if formIdx >= len(v.Inputs) {
					return fmt.Errorf("unexpected form encountered. expected %d", len(v.Inputs))
				}
				v.Inputs[formIdx].SendTo(mod, 50*time.Millisecond)
				formIdx++
				reader := io.TeeReader(mod.Output(), formOutput)
				teatest.WaitFor(
					c.T(),
					reader,
					testutil.WaitFormDone(form),
					teatest.WithCheckInterval(10*time.Millisecond),
					teatest.WithDuration(1000*time.Millisecond),
				)
				return nil
			}

			err = cmd.Command.Run(ctx, append([]string{"skiff", "add"}, v.Args(build)...))
			out := &output{
				Err:             err,
				Build:           build,
				TestModel:       mod,
				Form:            form,
				BuildRoot:       filesystem.New(build.RootDir),
				OriginalExample: filesystem.New(filepath.Join(ExamplesPath(), "go-fiber-controller")),
			}
			if err == nil {
				out.FinalOutput = testutil.Dump(mod.Output())
			}
			v.Expected(out)
			if c.T().Failed() {
				fmt.Println(formOutput.String())
			}
		})
	}
}

type BuildCmdOutput struct {
	OutputDir string
	RootDir   string
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
	outputDir := filepath.Join("public", "r")

	err = cmd.Command.Run(
		ctx,
		slices.Concat(
			[]string{"skiff", "build", filepath.Join(".skiff", "registry.json")},
			slices.Concat(args, []string{"-o", outputDir}),
		),
	)
	if !c.NoError(err) {
		return nil, false
	}

	return &BuildCmdOutput{
		OutputDir: outputDir,
		Stdout:    buf.Bytes(),
		RootDir:   exaDir,
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
