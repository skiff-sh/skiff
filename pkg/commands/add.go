package commands

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/urfave/cli/v3"

	"github.com/skiff-sh/skiff/pkg/filesystem"
	"github.com/skiff-sh/skiff/pkg/interact"
	"github.com/skiff-sh/skiff/pkg/registry"
	"github.com/skiff-sh/skiff/pkg/schema"
	"github.com/skiff-sh/skiff/pkg/tmpl"
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

type AddAction struct {
	Packages     []*registry.PackageGenerator
	PackageFlags map[string][]*schema.Flag
}

// NewAddAction constructor for AddAction. Packages should be retrieved prior to the construction
// of this action because flags are dynamically added based on the package schema.
func NewAddAction(flags map[string][]*schema.Flag, packages []*registry.PackageGenerator) *AddAction {
	return &AddAction{
		Packages:     packages,
		PackageFlags: flags,
	}
}

func LoadPackages(ctx context.Context, packages []string) ([]*registry.PackageGenerator, error) {
	if len(packages) == 0 {
		return nil, errors.New("path to package required")
	}

	generators := make([]*registry.PackageGenerator, 0, len(packages))
	loader := initLoader(packages[0])
	for _, v := range packages {
		pkg, err := loader.LoadPackage(ctx, v)
		if err != nil {
			return nil, fmt.Errorf("package %s: %w", v, err)
		}

		generators = append(generators, pkg)
	}

	return generators, nil
}

func FlagsFromPackages(nonInteractive bool, pkgs []*registry.PackageGenerator) (map[string][]*schema.Flag, error) {
	out := make(map[string][]*schema.Flag, len(pkgs))
	multiplePackages := len(pkgs) > 1
	for _, v := range pkgs {
		flags := make([]*schema.Flag, 0, len(pkgs))
		for _, field := range v.Schema.Fields {
			fl := schema.FieldToCLIFlag(field)
			if fl == nil {
				return nil, fmt.Errorf("invalid flag %s", v.Proto.GetName())
			}

			fl.Package = v.Proto.GetName()
			namespaced := v.Proto.GetName() + "." + fl.Accessor.Name()
			fl.Accessor.SetCategory(fmt.Sprintf("%s data", v.Proto.GetName()))
			if multiplePackages {
				// Names must be namespaces to avoid conflicts.
				fl.Accessor.SetName(namespaced)
			} else {
				// Set namespaced aliases so that the API is consistent with the multiple packages.
				fl.Accessor.SetAliases([]string{namespaced})
			}
			if nonInteractive {
				fl.Accessor.SetRequired(true)
			}
			flags = append(flags, fl)
		}
		out[v.Proto.GetName()] = flags
	}

	return out, nil
}

type AddArgs struct {
	ProjectRoot filesystem.Filesystem
	CreateAll   bool
}

func (a *AddAction) Act(ctx context.Context, args *AddArgs) error {
	data := schema.NewData()

	missingPackageFlags := map[string][]*schema.Flag{}
	for packageName, flags := range a.PackageFlags {
		for i := range flags {
			if flags[i].Flag.IsSet() {
				data.AddPackageEntry(packageName, flags[i])
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

		pkg := a.Packages[slices.IndexFunc(a.Packages, func(p *registry.PackageGenerator) bool {
			return p.Proto.GetName() == packageName
		})]

		pkgFormFields[packageName] = formFields

		group := interact.NewHuhGroup(schema.FlattenHuhFields(formFields)...)
		group.Title(fmt.Sprintf("Package %s", pkg.Proto.GetName())).Description(pkg.Proto.GetDescription())
		groups = append(groups, group)
	}

	form := interact.NewHuhForm(groups...)
	err := interact.FormRunner(ctx, form)
	if err != nil {
		return err
	}

	for pkgName, inputs := range pkgFormFields {
		for i := range inputs {
			data.AddPackageEntry(pkgName, inputs[i])
		}
	}

	confirmer := func(ctx context.Context, f *registry.File) (bool, error) {
		var prompt string
		if args.ProjectRoot.Exists(f.Proto.GetTarget()) {
			prompt = fmt.Sprintf("Edit file %s", f.Proto.GetTarget())
		} else {
			prompt = fmt.Sprintf("Create file %s", f.Proto.GetTarget())
		}
		return interact.Confirm(ctx, prompt)
	}
	if !args.CreateAll {
		confirmer = func(_ context.Context, _ *registry.File) (bool, error) {
			return true, nil
		}
	}

	for _, gen := range a.Packages {
		data := data.Package(gen.Proto.GetName())

		pkg, err := gen.Generate(data.RawData())
		if err != nil {
			return err
		}

		for _, v := range pkg.Files {
			ok, err := confirmer(ctx, v)
			if err != nil {
				return err
			}

			if !ok {
				continue
			}

			err = v.WriteTo(args.ProjectRoot)
			if err != nil {
				interact.Errorf("Failed to write file %s: %s", v.Proto.GetTarget(), err.Error())
			}
		}
	}

	return nil
}

func initLoader(pa string) registry.Loader {
	tmplFact := tmpl.NewGoFactory()
	if registry.IsHTTPPath(pa) {
		cl := &http.Client{
			Timeout: 1 * time.Second,
		}
		return registry.NewHTTPLoader(tmplFact, cl)
	}
	return registry.NewFileLoader(tmplFact)
}
