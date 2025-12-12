package commands

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"unicode/utf8"

	"github.com/skiff-sh/config/ptr"
	"github.com/urfave/cli/v3"
	"google.golang.org/protobuf/proto"

	"github.com/skiff-sh/skiff/pkg/filesystem"
	"github.com/skiff-sh/skiff/pkg/gocmd"
	"github.com/skiff-sh/skiff/pkg/plugin"

	"github.com/skiff-sh/api/go/skiff/registry/v1alpha1"

	"github.com/skiff-sh/skiff/pkg/fileutil"
	"github.com/skiff-sh/skiff/pkg/interact"
	"github.com/skiff-sh/skiff/pkg/protoencode"
)

var BuildFlagOutputDirectory = &cli.StringFlag{
	Name:    "output",
	Usage:   "Destination directory for registry json files.",
	Value:   "./public/r",
	Aliases: []string{"o", "out"},
}

var BuildArgRegistryPath = &cli.StringArg{
	Name:      "registry",
	UsageText: "registry file path",
}

type BuildCommandAction struct {
}

func NewBuildAction() *BuildCommandAction {
	return &BuildCommandAction{}
}

type BuildArgs struct {
	OutputDirectory string
	// Path to the registry file
	RegistryPath string
}

func (b *BuildCommandAction) Act(ctx context.Context, args *BuildArgs) error {
	_ = os.MkdirAll(args.OutputDirectory, 0755)

	regPath := fileutil.MustAbs(args.RegistryPath)
	if !fileutil.Exists(regPath) {
		return fmt.Errorf("%s does not exist", regPath)
	}

	regDir := filepath.Dir(regPath)
	regFS := filesystem.New(regDir)

	reg := new(v1alpha1.Registry)
	err := protoencode.LoadFile(regPath, reg)
	if err != nil {
		return fmt.Errorf("failed to load registry at %s: %w", regPath, err)
	}
	slog.DebugContext(ctx, "Loading registry file.", "path", regPath)

	for _, v := range reg.GetPackages() {
		targetPath := filepath.Join(args.OutputDirectory, v.GetName()+".json")
		interact.Infof("Writing file %s", targetPath)
		pkg, err := HydratePackage(ctx, sync.OnceValue(newPluginTools), v, regFS)
		if err != nil {
			return fmt.Errorf("package %s: %w", v.GetName(), err)
		}

		err = WritePackage(pkg, targetPath)
		if err != nil {
			return fmt.Errorf("package %s: %w", v.GetName(), err)
		}
	}

	for _, v := range reg.GetPackages() {
		for _, fi := range v.GetFiles() {
			// zero out source so it's not included in the registry catalog.
			fi.Source = nil
		}
	}

	raw, err := protoencode.PrettyMarshaller.Marshal(reg)
	if err != nil {
		return fmt.Errorf("registry invalid: %w", err)
	}
	targetPath := filepath.Join(args.OutputDirectory, "registry.json")
	interact.Infof("Writing file %s", targetPath)
	err = os.WriteFile(targetPath, raw, 0644)
	if err != nil {
		return fmt.Errorf("failed to write registry: %w", err)
	}

	return nil
}

type pluginTools struct {
	GoCLI          gocmd.CLI
	BuildDirectory string
	PluginCompiler plugin.Compiler
	Err            error
}

const minMinorGoVersion = 24

func newPluginTools() *pluginTools {
	out := &pluginTools{}
	var err error
	out.BuildDirectory, err = os.MkdirTemp(os.TempDir(), "*")
	if err != nil {
		out.Err = fmt.Errorf("failed to make temp build directory in %s: %w", os.TempDir(), err)
		return out
	}

	out.GoCLI, err = gocmd.New()
	if err != nil {
		out.Err = errors.New("go CLI is required to build plugins. Get the CLI here => https://go.dev/doc/install")
		return out
	}

	ver, err := out.GoCLI.Version(context.Background())
	if err != nil {
		out.Err = fmt.Errorf("failed to get go CLI version: %w", err)
		return out
	}
	if ver.Minor < minMinorGoVersion {
		out.Err = fmt.Errorf(
			"go CLI (at %s) is version %s. please upgrade to at least 1.%d to properly build plugins",
			out.GoCLI.Path(),
			ver.String(),
			minMinorGoVersion,
		)
		return out
	}

	out.PluginCompiler, err = plugin.NewWazeroCompiler()
	if err != nil {
		out.Err = fmt.Errorf("failed to create WASM compiler: %w", err)
		return out
	}

	return out
}

func HydratePackage(
	ctx context.Context,
	toolsProvider func() *pluginTools,
	pkg *v1alpha1.Package,
	registryRoot filesystem.Filesystem,
) (*v1alpha1.Package, error) {
	out := proto.CloneOf(pkg)
	for _, v := range out.GetFiles() {
		src := v.GetSource()
		if src == nil {
			src = &v1alpha1.File_Source{}
		}

		switch v.GetType() {
		case v1alpha1.File_file:
			content, err := fs.ReadFile(registryRoot, v.GetPath())
			if err != nil {
				return nil, fmt.Errorf("failed to read file %s: %w", v.GetPath(), err)
			}

			if utf8.Valid(content) {
				src.Text = ptr.Ptr(string(content))
			} else {
				src.Raw = content
			}
		case v1alpha1.File_plugin:
			idx := slices.IndexFunc(out.GetFiles(), func(file *v1alpha1.File) bool {
				return file.GetType() == v1alpha1.File_plugin && file.GetPath() == v.GetPath() &&
					len(v.GetSource().GetRaw()) > 0
			})
			if idx >= 0 {
				src.FileIndex = ptr.Ptr(int32(idx))
			} else {
				tools := toolsProvider()
				if tools.Err != nil {
					return nil, fmt.Errorf("file %s: %w", v.GetPath(), tools.Err)
				}

				interact.Infof("Building plugin %s", v.GetPath())
				args := gocmd.BuildArgs{
					BuildMode:  gocmd.BuildModeCShared,
					OutputPath: filepath.Join(tools.BuildDirectory, "plugin.go"),
					Packages:   []string{registryRoot.AbsolutePath(v.GetPath())},
					GoOS:       gocmd.OSWASIP1,
					GoArch:     gocmd.ArchWASM,
					Env:        os.Environ(),
				}

				buffers, err := tools.GoCLI.Build(ctx, args)
				if err != nil {
					var stderr string
					if buffers != nil {
						stderr = buffers.Stderr.String()
					}
					return nil, fmt.Errorf("failed to build plugin %s: %w: %s", v.GetPath(), err, stderr)
				}

				interact.Info("Complete!")

				content, err := os.ReadFile(args.OutputPath)
				if err != nil {
					return nil, fmt.Errorf("failed to read WASM output at %s for %s: %w", args.OutputPath, v.GetPath(), err)
				}
				_ = os.Remove(args.OutputPath)

				_, err = tools.PluginCompiler.Compile(ctx, content, plugin.CompileOpts{})
				if err != nil {
					return nil, fmt.Errorf("failed to compile plugin %s: %w", v.GetPath(), err)
				}

				src.Raw = content
			}
		}
		v.Source = src
	}
	return out, nil
}

func WritePackage(pkg *v1alpha1.Package, targetPath string) error {
	b, err := protoencode.PrettyMarshaller.Marshal(pkg)
	if err != nil {
		return err
	}

	return os.WriteFile(targetPath, b, 0644)
}
