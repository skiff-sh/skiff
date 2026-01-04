package commands

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/urfave/cli/v3"

	"github.com/skiff-sh/skiff/pkg/engine"

	"github.com/skiff-sh/skiff/pkg/filesystem"
	"github.com/skiff-sh/skiff/pkg/interact"
	"github.com/skiff-sh/skiff/pkg/registry"
	"github.com/skiff-sh/skiff/pkg/schema"
)

var AddArgPackages = &cli.StringArgs{
	Name:      "packages",
	UsageText: "URL or local path to package JSON file",
	Min:       1,
	Max:       -1,
}

var AddFlagNonInteractive = &cli.BoolFlag{
	Name:    "non-interactive",
	Usage:   "Disable all form prompts. All package schema's will be required via flags.",
	Aliases: []string{"noi", "non-i"},
}

var AddFlagCreateAll = &cli.BoolFlag{
	Name:    "create",
	Usage:   "Auto-confirm all prompts to create files but prompt to run plugins.",
	Aliases: []string{"y"},
}

var AddFlagRoot = &cli.StringFlag{
	Name:    "root",
	Usage:   "The root of your project. All files are written relative to the root. Defaults to the cwd.",
	Aliases: []string{"r"},
}

var ErrSchema = errors.New("schema error")

type AddAction struct {
	Packages     []*registry.LoadedPackage
	PackageFlags map[string][]*schema.Flag
}

// NewAddAction constructor for AddAction. Packages should be retrieved prior to the construction
// of this action because flags are dynamically added based on the package schema.
func NewAddAction(flags map[string][]*schema.Flag, packages []*registry.LoadedPackage) *AddAction {
	return &AddAction{
		Packages:     packages,
		PackageFlags: flags,
	}
}

func LoadPackages(ctx context.Context, packages []string) ([]*registry.LoadedPackage, error) {
	if len(packages) == 0 {
		return nil, errors.New("path to package required")
	}

	pkgs := make([]*registry.LoadedPackage, 0, len(packages))
	for _, v := range packages {
		pkg, err := registry.LoadPackage(ctx, v)
		if err != nil {
			return nil, fmt.Errorf("package %s: %w", v, err)
		}

		pkgs = append(pkgs, pkg)
	}

	return pkgs, nil
}

func FlagsFromPackages(nonInteractive bool, pkgs []*registry.LoadedPackage) (map[string][]*schema.Flag, error) {
	out := make(map[string][]*schema.Flag, len(pkgs))
	for _, pkg := range pkgs {
		fl, err := pkg.CLIFlags(nonInteractive)
		if err != nil {
			return nil, fmt.Errorf("package %s: %w", pkg.Proto.GetName(), err)
		}
		out[pkg.Proto.GetName()] = fl
	}

	return out, nil
}

type AddArgs struct {
	ProjectRoot filesystem.Filesystem
	CreateAll   bool
	Engine      engine.Engine
}

func (a *AddAction) Act(ctx context.Context, args *AddArgs) error {
	pkgs := a.Packages
	pkgFlags := a.PackageFlags

	if len(pkgs) == 0 {
		return nil
	}

	data := schema.NewDataSource()

	missingPackageFlags := map[string][]*schema.Flag{}
	for packageName, flags := range pkgFlags {
		for i := range flags {
			if flags[i].Flag.IsSet() {
				data.AddPackageEntries(packageName, flags[i])
			} else {
				missingPackageFlags[packageName] = append(missingPackageFlags[packageName], flags[i])
			}
		}
	}

	groups := make([]*huh.Group, 0, len(missingPackageFlags))
	pkgFormFields := make(map[string][]*schema.FormField, len(missingPackageFlags))
	for packageName, flags := range missingPackageFlags {
		formFields := make([]*schema.FormField, 0, len(flags))
		for _, fl := range flags {
			ff := schema.NewFormField(fl.Field)
			if ff == nil {
				return errors.New("failed to create field")
			}

			ff.Accessor.SetDescription(
				strings.Join([]string{fl.Field.Proto.GetDescription(), ff.Accessor.Description()}, ". "),
			)
			ff.Accessor.SetTitle(fl.Field.Proto.GetName())
			formFields = append(formFields, ff)
		}

		pkg := pkgs[slices.IndexFunc(pkgs, func(p *registry.LoadedPackage) bool {
			return p.Proto.GetName() == packageName
		})]

		pkgFormFields[packageName] = formFields

		group := interact.NewHuhGroup(schema.FlattenHuhFields(formFields)...)
		group.Title(fmt.Sprintf("Package %s", pkg.Proto.GetName())).Description(pkg.Proto.GetDescription())
		groups = append(groups, group)
	}

	form := interact.NewHuhForm(groups...)
	err := interact.DefaultFormRunner(ctx, form)
	if err != nil {
		return err
	}

	for pkgName, inputs := range pkgFormFields {
		for i := range inputs {
			data.AddPackageEntries(pkgName, inputs[i])
		}
	}

	confirmer := func(ctx context.Context, f *registry.File) (bool, error) {
		var prompt string
		if args.ProjectRoot.Exists(f.Path) {
			prompt = fmt.Sprintf("Edit file %s", f.Path)
		} else {
			prompt = fmt.Sprintf("Create file %s", f.Path)
		}
		conf := interact.Confirm(ctx, func(c *huh.Confirm) *huh.Confirm {
			return c.Title(prompt)
		})
		return conf, nil
	}
	if args.CreateAll {
		confirmer = func(_ context.Context, _ *registry.File) (bool, error) {
			return true, nil
		}
	}

	for _, v := range pkgs {
		data := data.Package(v.Proto.GetName())

		resp, err := args.Engine.AddPackage(ctx, v, data)
		if err != nil {
			return fmt.Errorf("package %s: %w", v.Proto.GetName(), err)
		}

		for _, fi := range resp.Diffs {
			ok, err := confirmer(ctx, fi.File)
			if err != nil {
				return err
			}

			if !ok {
				continue
			}

			err = fi.File.WriteTo(args.ProjectRoot)
			if err != nil {
				interact.Errorf("Failed to write file %s: %s", fi.File.Path, err.Error())
			}
		}
	}

	return nil
}
