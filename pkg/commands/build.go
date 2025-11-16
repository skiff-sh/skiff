package commands

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/skiff-sh/config/ptr"
	"github.com/skiff-sh/skiff/api/go/skiff/registry/v1alpha1"
	"github.com/skiff-sh/skiff/pkg/fileutil"
	"github.com/skiff-sh/skiff/pkg/interact"
	"github.com/skiff-sh/skiff/pkg/protoencode"
	"github.com/skiff-sh/skiff/pkg/registry"
	"github.com/urfave/cli/v3"
	"google.golang.org/protobuf/proto"
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

func NewBuildAction() *BuildCommandAction {
	return &BuildCommandAction{}
}

type BuildCommandAction struct {
}

type BuildArgs struct {
	OutputDirectory string
	// Path to the registry file
	RegistryPath string
}

func (b *BuildCommandAction) Act(_ context.Context, args *BuildArgs) error {
	_ = os.MkdirAll(args.OutputDirectory, 0755)

	if !fileutil.Exists(args.RegistryPath) {
		return fmt.Errorf("%s does not exist", args.RegistryPath)
	}

	regDir, regName := filepath.Split(args.RegistryPath)
	regFS := os.DirFS(regDir)
	fi, err := regFS.Open(regName)
	if err != nil {
		return err
	}
	defer func() {
		_ = fi.Close()
	}()

	reg, err := registry.Load(fi)
	if err != nil {
		return fmt.Errorf("failed to load registry at %s: %w", args.RegistryPath, err)
	}

	for _, v := range reg.Packages {
		targetPath := filepath.Join(args.OutputDirectory, v.Name+".json")
		interact.Infof("Writing file %s", targetPath)
		pkg, err := HydratePackage(v, regFS)
		if err != nil {
			return fmt.Errorf("package %s: %w", v.Name, err)
		}

		err = WritePackage(pkg, targetPath)
		if err != nil {
			return fmt.Errorf("package %s: %w", v.Name, err)
		}
	}

	for _, v := range reg.Packages {
		for _, fi := range v.Files {
			// zero out content so it's not included in the registry catalog.
			fi.Content = nil
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

func HydratePackage(pkg *v1alpha1.Package, registryRoot fs.FS) (*v1alpha1.Package, error) {
	out := proto.CloneOf(pkg)
	for _, v := range out.Files {
		if v.Content != nil && len(v.Raw) == 0 {
			continue
		}
		content, err := fs.ReadFile(registryRoot, v.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", v.Path, err)
		}

		if v.Type == v1alpha1.File_plugin {
			v.Raw = content
		} else {
			v.Content = ptr.Ptr(string(content))
		}
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
