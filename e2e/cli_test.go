package e2e

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/stretchr/testify/suite"

	"github.com/skiff-sh/skiff/pkg/bufferpool"

	"github.com/skiff-sh/skiff/pkg/registry"

	"github.com/skiff-sh/skiff/pkg/execcmd"
	"github.com/skiff-sh/skiff/pkg/settings"

	"github.com/skiff-sh/skiff/pkg/filesystem"

	"github.com/skiff-sh/api/go/skiff/registry/v1alpha1"

	"github.com/skiff-sh/skiff/pkg/collection"
	"github.com/skiff-sh/skiff/pkg/interact"
	"github.com/skiff-sh/skiff/pkg/protoencode"
	"github.com/skiff-sh/skiff/pkg/testutil"
)

type CliTestSuite struct {
	testutil.Suite
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
		ExampleFS     testutil.MapFS
		BuildOutputFS testutil.MapFS
		Actual        *BuildOutput
	}

	type test struct {
		Args        []string
		ExampleName string
		Setup       func()
		Cleanup     func(buildDir string)
		Expected    func(p *params)
		ExpectedErr string
	}

	oldPathLooker := execcmd.DefaultPathLooker
	oldEnvs := os.Environ()
	tests := map[string]test{
		"go controller all files present": {
			ExampleName: "go-fiber-controller",
			Expected: func(p *params) {
				c.ElementsMatch(
					[]string{"registry.json", "create-http-route.json", "plugins/plugin.wasm"},
					collection.Keys(p.BuildOutputFS),
				)
				c.Len(p.Actual.Packages, 1)

				// Check that the contents are the same as the original.
				for _, pkg := range p.Actual.Packages {
					for _, fi := range pkg.GetFiles() {
						c.NotEmpty(string(p.ExampleFS[filepath.Join(".skiff", fi.GetPath())].Data))
					}
				}
			},
		},
		"build with managed go CLI": {
			ExampleName: "go-fiber-controller",
			Setup: func() {
				// We assume the user has home.
				home := os.Getenv("HOME")
				os.Clearenv()
				_ = os.Setenv("HOME", home)
				execcmd.DefaultPathLooker = execcmd.PathLookerFunc(func(fi string) (string, error) {
					if fi == "go" || fi == "go.exe" {
						return "", exec.ErrNotFound
					}
					return exec.LookPath(fi)
				})
			},
			Cleanup: func(buildDir string) {
				execcmd.DefaultPathLooker = oldPathLooker
				for _, pair := range oldEnvs {
					// Environ() returns strings in "KEY=VALUE" format
					kv := strings.SplitN(pair, "=", 2)
					_ = os.Setenv(kv[0], kv[1])
				}

				// For some reason, go deps are installed protected. maybe it's gvm? not 100% sure if this is consistent on linux or windows.
				out, err := exec.CommandContext(c.T().Context(), "chmod", "-R", "u+w", buildDir).CombinedOutput()
				if err != nil {
					fmt.Println(string(out))
				}
			},
			Expected: func(p *params) {
				c.ElementsMatch(
					[]string{"registry.json", "create-http-route.json", "plugins/plugin.wasm"},
					collection.Keys(p.BuildOutputFS),
				)
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

			oldBuildDir := settings.BuildDirFunc
			buildDir := c.T().TempDir()
			settings.BuildDirFunc = func() (string, error) {
				return buildDir, nil
			}
			defer func() {
				settings.BuildDirFunc = oldBuildDir
			}()

			defer c.SetWd(exaDir)()

			if v.Setup != nil {
				v.Setup()
			}

			if v.Cleanup != nil {
				defer v.Cleanup(buildDir)
			}

			build, ok := c.buildExample(exaDir, v.Args...)
			if !ok {
				return
			}

			outputFS := os.DirFS(build.OutputDir)
			actualFS := testutil.FlatMapFS(outputFS)
			actual, ok := c.unmarshalBuildOutput(outputFS)
			if !ok {
				return
			}

			exaFS := os.DirFS(exaDir)
			v.Expected(&params{
				ExampleDir:    exaFS,
				ExampleFS:     testutil.FlatMapFS(exaFS),
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
					"--non-i",
					filepath.Join(b.OutputDir, "create-http-route.json"),
				}
			},
			Expected: func(p *output) {
				c.ErrorContains(p.Err, "Required flags")
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

func (c *CliTestSuite) TestMCP() {
	ctx := c.T().Context()
	examples := os.DirFS(ExamplesPath())
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

	go func() {
		c.NoError(cmd.Command.Run(ctx, []string{"skiff", "mcp", "--addr", ":8080"}))
	}()

	codex := exec.CommandContext(
		ctx,
		"codex",
		"-c",
		"mcp_servers.skiff.url=http://localhost:8080",
		"-a",
		"never",
		"exec",
		"-s",
		"workspace-write",
		"--skip-git-repo-check",
		"add a route called derp",
	)
	codex.Dir = build.RootDir
	out, err := codex.CombinedOutput()
	if !c.NoError(err) {
		return
	}
	// TODO: validate that this works by hitting the derp endpoint. Need to validate that the token count is minimized and it shouldn't take very long.

	fmt.Println(string(out))
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
	outputDir := filepath.Join(exaDir, "public", "r")

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
	Plugins  []*registry.File
}

func (c *CliTestSuite) unmarshalBuildOutput(f fs.FS) (*BuildOutput, bool) {
	out := &BuildOutput{
		Registry: new(v1alpha1.Registry),
		Packages: make(map[string]*v1alpha1.Package),
	}

	reg, err := fs.ReadFile(f, "registry.json")
	if !c.NoError(err, "registry.json is missing") {
		return nil, false
	}

	err = protoencode.Unmarshaller.Unmarshal(reg, out.Registry)
	if !c.NoError(err, "failed to unmarshal the registry.json file") {
		return nil, false
	}

	files, err := fs.ReadDir(f, ".")
	if !c.NoError(err) {
		return nil, false
	}

	for _, v := range files {
		if v.Name() == "registry.json" || v.IsDir() {
			continue
		}
		out.Packages[v.Name()] = new(v1alpha1.Package)
		con, err := fs.ReadFile(f, v.Name())
		if !c.NoError(err, "failed to read package", v.Name()) {
			return nil, false
		}

		err = protoencode.Unmarshaller.Unmarshal(con, out.Packages[v.Name()])
		if !c.NoError(err, "failed to unmarshal package %s", v.Name()) {
			return nil, false
		}
	}

	plugins, _ := fs.ReadDir(f, "plugins")
	out.Plugins = make([]*registry.File, 0, len(plugins))
	for _, v := range plugins {
		fp := filepath.Join("plugins", v.Name())
		content, _ := fs.ReadFile(f, fp)
		buff := bufferpool.GetBytesBuffer()
		buff.Write(content)
		out.Plugins = append(out.Plugins, &registry.File{
			Path:    fp,
			Content: buff,
		})
	}

	return out, true
}

func TestE2ETestSuite(t *testing.T) {
	suite.Run(t, new(CliTestSuite))
}
