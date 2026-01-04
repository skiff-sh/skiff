package registry

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/skiff-sh/api/go/skiff/registry/v1alpha1"

	"github.com/skiff-sh/skiff/pkg/filesystem"
	"github.com/skiff-sh/skiff/pkg/plugin"
	"github.com/skiff-sh/skiff/pkg/schema"
	"github.com/skiff-sh/skiff/pkg/system"
	"github.com/skiff-sh/skiff/pkg/tmpl"
)

type LoadedPackage struct {
	Proto  *v1alpha1.Package
	Schema *schema.Schema
}

func LoadPackage(ctx context.Context, pkg string) (*LoadedPackage, error) {
	out, err := NewLoader(pkg).LoadPackage(ctx, pkg)
	if err != nil {
		return nil, fmt.Errorf("failed to load: %w", err)
	}

	sch, err := schema.New(out.GetSchema())
	if err != nil {
		return nil, fmt.Errorf("invalid schema: %w", err)
	}

	return &LoadedPackage{
		Proto:  out,
		Schema: sch,
	}, nil
}

func LoadRegistry(ctx context.Context, reg string) (*v1alpha1.Registry, error) {
	return NewLoader(reg).LoadRegistry(ctx, reg)
}

func (l *LoadedPackage) CLIFlags(required bool) ([]*schema.Flag, error) {
	flags := make([]*schema.Flag, 0, len(l.Schema.Fields))
	for _, field := range l.Schema.Fields {
		fl := schema.FieldToCLIFlag(field)
		if fl == nil {
			return nil, fmt.Errorf("package %s: invalid flag %s", l.Proto.GetName(), field.Proto.GetName())
		}

		fl.Package = l.Proto.GetName()
		namespaced := l.Proto.GetName() + "." + fl.Accessor.Name()
		fl.Accessor.SetCategory(fmt.Sprintf("%s flags", l.Proto.GetName()))
		// Names must be namespaces to avoid conflicts.
		fl.Accessor.SetName(namespaced)
		if required {
			fl.Accessor.SetRequired(true)
		}
		flags = append(flags, fl)
	}
	return flags, nil
}

func NewLoader(u string) Loader {
	if IsHTTPPath(u) {
		cl := &http.Client{
			Timeout: 1 * time.Second,
		}
		return NewHTTPLoader(cl)
	}
	return NewFileLoader()
}

type PackageGenerator struct {
	Package *LoadedPackage
	Files   []*PackageFile
}

func NewPackageGenerator(
	ctx context.Context,
	compiler plugin.Compiler,
	sys system.System,
	t tmpl.Factory,
	p *LoadedPackage,
) (*PackageGenerator, error) {
	out := &PackageGenerator{
		Package: p,
		Files:   make([]*PackageFile, 0, len(p.Proto.GetFiles())),
	}

	for _, v := range p.Proto.GetFiles() {
		fi, err := NewPackageFile(ctx, compiler, sys, t, p.Proto, v)
		if err != nil {
			return nil, fmt.Errorf("file %s: %w", v.GetPath(), err)
		}

		out.Files = append(out.Files, fi)
	}

	return out, nil
}

func (p *PackageGenerator) Generate(ctx context.Context, d schema.PackageDataSource) (*Package, error) {
	out := &Package{
		Proto: p.Package.Proto,
		Files: make([]*File, 0, len(p.Files)),
	}

	for _, v := range p.Files {
		fi, err := v.GenerateFile(ctx, d)
		if err != nil {
			return nil, fmt.Errorf("file %s: %w", v.File.GetPath(), err)
		}

		out.Files = append(out.Files, fi)
	}

	return out, nil
}

type Package struct {
	Proto *v1alpha1.Package
	Files []*File
}

func (p *Package) WriteTo(fsys filesystem.Filesystem) error {
	for _, v := range p.Files {
		err := v.WriteTo(fsys)
		if err != nil {
			return fmt.Errorf("failed to write file %s to %s: %w", v.SourcePath, v.Path, err)
		}
	}

	return nil
}
