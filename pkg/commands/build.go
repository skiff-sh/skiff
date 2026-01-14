package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/skiff-sh/config/contexts"
	"github.com/urfave/cli/v3"

	"github.com/skiff-sh/skiff/pkg/build"

	"github.com/skiff-sh/skiff/pkg/filesystem"
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
	logger := contexts.GetLogger(ctx)
	_ = os.MkdirAll(args.OutputDirectory, fileutil.DefaultDirMode)

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
	logger.DebugContext(ctx, "Reading packages for registry file.", "registry", regPath)

	output := filesystem.New(args.OutputDirectory)

	builder := build.NewBuilder(build.NewToolsProvider(&plugin.InstallHooks{
		OnDownload: func() {
			interact.Info("Installing Go...")
		},
		OnDownloadComplete: func() {
			interact.Info("Done!")
		},
	}), &build.Hooks{
		BeforeBuild: func(f *v1alpha1.File) {
			interact.Infof("Building plugin %s", f.Path)
		},
		OnBuild: func(_ *v1alpha1.File) {
			interact.Info("Done!")
		},
	})

	builds := make([]*build.Package, 0, len(reg.Packages))
	for _, v := range reg.Packages {
		interact.Infof("Building package %s", v.Name)

		b, err := builder.Build(ctx, regFS, v)
		if err != nil {
			return fmt.Errorf("package %s: %w", v.Name, err)
		}
		builds = append(builds, b)
		interact.Info("Done!")
	}

	for _, v := range builds {
		err = v.WriteTo(output)
		if err != nil {
			return fmt.Errorf("package %s: %w", v.Proto.Name, err)
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
	err = os.WriteFile(targetPath, raw, fileutil.DefaultFileMode)
	if err != nil {
		return fmt.Errorf("failed to write registry: %w", err)
	}

	return nil
}
