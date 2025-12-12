package registry

import (
	"context"
	"fmt"

	"github.com/skiff-sh/api/go/skiff/registry/v1alpha1"

	"github.com/skiff-sh/skiff/pkg/filesystem"
	"github.com/skiff-sh/skiff/pkg/plugin"
	"github.com/skiff-sh/skiff/pkg/schema"
	"github.com/skiff-sh/skiff/pkg/system"
	"github.com/skiff-sh/skiff/pkg/tmpl"
)

type PackageGenerator struct {
	Proto  *v1alpha1.Package
	Files  []*PackageFile
	Schema *schema.Schema
}

func NewPackageGenerator(
	ctx context.Context,
	compiler plugin.Compiler,
	sys system.System,
	t tmpl.Factory,
	p *v1alpha1.Package,
) (*PackageGenerator, error) {
	out := &PackageGenerator{
		Proto: p,
		Files: make([]*PackageFile, 0, len(p.GetFiles())),
	}

	var err error
	out.Schema, err = schema.NewSchema(p.GetSchema())
	if err != nil {
		return nil, err
	}

	for _, v := range p.GetFiles() {
		fi, err := NewPackageFile(ctx, compiler, sys, t, p, v)
		if err != nil {
			return nil, fmt.Errorf("file %s: %w", v.GetPath(), err)
		}

		out.Files = append(out.Files, fi)
	}

	return out, nil
}

func (p *PackageGenerator) Generate(ctx context.Context, d schema.PackageDataSource) (*Package, error) {
	out := &Package{
		Proto: p.Proto,
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
