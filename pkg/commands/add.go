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
	"github.com/skiff-sh/api/go/skiff/registry/v1alpha1"
	"github.com/urfave/cli/v3"

	"github.com/skiff-sh/skiff/pkg/accesscontrol"
	"github.com/skiff-sh/skiff/pkg/filesystem"
	"github.com/skiff-sh/skiff/pkg/interact"
	"github.com/skiff-sh/skiff/pkg/plugin"
	"github.com/skiff-sh/skiff/pkg/registry"
	"github.com/skiff-sh/skiff/pkg/schema"
	"github.com/skiff-sh/skiff/pkg/system"
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

var AddFlagPermissions = &cli.StringSliceFlag{
	Name: "permission",
	Usage: "Grant permissions for plugins running on your machine. By default, none are granted. Valid permissions are:\n" + strings.Join(
		accesscontrol.PermUsageListPretty(accesscontrol.AllPerms()),
		"\n",
	),
	Aliases: []string{"p"},
	Config: cli.StringConfig{
		TrimSpace: true,
	},
	Validator: func(strs []string) error {
		for _, v := range strs {
			_, ok := v1alpha1.PackagePermissions_Plugin_value[v]
			if !ok {
				return fmt.Errorf("%s is not a valid permission", v)
			}
		}
		return nil
	},
}

var ErrSchema = errors.New("schema error")

type AddAction struct {
	Packages     []*v1alpha1.Package
	PackageFlags map[string][]*schema.Flag
}

// NewAddAction constructor for AddAction. Packages should be retrieved prior to the construction
// of this action because flags are dynamically added based on the package schema.
func NewAddAction(flags map[string][]*schema.Flag, packages []*v1alpha1.Package) *AddAction {
	return &AddAction{
		Packages:     packages,
		PackageFlags: flags,
	}
}

func LoadPackages(ctx context.Context, packages []string) ([]*v1alpha1.Package, error) {
	if len(packages) == 0 {
		return nil, errors.New("path to package required")
	}

	generators := make([]*v1alpha1.Package, 0, len(packages))
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

func FlagsFromPackages(nonInteractive bool, pkgs []*v1alpha1.Package) (map[string][]*schema.Flag, error) {
	out := make(map[string][]*schema.Flag, len(pkgs))
	for _, pkg := range pkgs {
		sc, err := schema.NewSchema(pkg.GetSchema())
		if err != nil {
			return nil, fmt.Errorf("package %s: %w", pkg.GetName(), err)
		}
		flags := make([]*schema.Flag, 0, len(pkgs))
		for _, field := range sc.Fields {
			fl := schema.FieldToCLIFlag(field)
			if fl == nil {
				return nil, fmt.Errorf("package %s: invalid flag %s", pkg.GetName(), field.Proto.GetName())
			}

			fl.Package = pkg.GetName()
			namespaced := pkg.GetName() + "." + fl.Accessor.Name()
			fl.Accessor.SetCategory(fmt.Sprintf("%s flags", pkg.GetName()))
			// Names must be namespaces to avoid conflicts.
			fl.Accessor.SetName(namespaced)
			if nonInteractive {
				fl.Accessor.SetRequired(true)
			}
			flags = append(flags, fl)
		}
		out[pkg.GetName()] = flags
	}

	return out, nil
}

type AddArgs struct {
	ProjectRoot  filesystem.Filesystem
	CreateAll    bool
	GrantedPerms []v1alpha1.PackagePermissions_Plugin
}

func (a *AddAction) Act(ctx context.Context, args *AddArgs) error {
	pkgs := a.Packages
	pkgFlags := a.PackageFlags
	pkgSystems := map[string]system.System{}
	granter := accesscontrol.NewTerminalGranter()
	mediator := system.NewMediator()

	var removeIdx []int
	for i, pkg := range pkgs {
		policy := accesscontrol.NewPluginAccessPolicy(args.GrantedPerms)
		needed := policy.Diff(pkg.GetPermissions().GetPlugin()...)
		if len(needed) > 0 && !granter.RequestAccess(ctx, pkg.GetName(), needed) {
			removeIdx = append(removeIdx, i)
		} else {
			policy.Grant(needed...)
			pkgSystems[pkg.GetName()] = mediator.MediatedSystem(policy)
		}
	}

	for _, v := range removeIdx {
		pkg := pkgs[v]
		delete(pkgFlags, pkg.GetName())
		pkgs = slices.Delete(pkgs, v, v+1)
	}

	if len(pkgs) == 0 {
		return nil
	}

	data := schema.NewDataSource()

	missingPackageFlags := map[string][]*schema.Flag{}
	for packageName, flags := range pkgFlags {
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

		pkg := pkgs[slices.IndexFunc(pkgs, func(p *v1alpha1.Package) bool {
			return p.GetName() == packageName
		})]

		pkgFormFields[packageName] = formFields

		group := interact.NewHuhGroup(schema.FlattenHuhFields(formFields)...)
		group.Title(fmt.Sprintf("Package %s", pkg.GetName())).Description(pkg.GetDescription())
		groups = append(groups, group)
	}

	form := interact.NewHuhForm(groups...)
	err := interact.DefaultFormRunner(ctx, form)
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

	compiler, err := plugin.NewWazeroCompiler()
	if err != nil {
		return err
	}

	tmplFact := tmpl.NewGoFactory()

	for _, v := range pkgs {
		data := data.Package(v.GetName())

		gen, err := registry.NewPackageGenerator(ctx, compiler, pkgSystems[v.GetName()], tmplFact, v)
		if err != nil {
			return fmt.Errorf("package %s: %w", v.GetName(), err)
		}

		pkg, err := gen.Generate(ctx, data)
		if err != nil {
			return err
		}

		for _, fi := range pkg.Files {
			ok, err := confirmer(ctx, fi)
			if err != nil {
				return err
			}

			if !ok {
				continue
			}

			err = fi.WriteTo(args.ProjectRoot)
			if err != nil {
				interact.Errorf("Failed to write file %s: %s", fi.Path, err.Error())
			}
		}
	}

	return nil
}

func initLoader(pa string) registry.Loader {
	if registry.IsHTTPPath(pa) {
		cl := &http.Client{
			Timeout: 1 * time.Second,
		}
		return registry.NewHTTPLoader(cl)
	}
	return registry.NewFileLoader()
}
