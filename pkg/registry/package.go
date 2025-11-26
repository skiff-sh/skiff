package registry

import (
	"fmt"

	"github.com/skiff-sh/api/go/skiff/registry/v1alpha1"
	"github.com/skiff-sh/skiff/pkg/filesystem"
	"github.com/skiff-sh/skiff/pkg/schema"
	"github.com/skiff-sh/skiff/pkg/tmpl"
)

type PackageGenerator struct {
	Proto  *v1alpha1.Package
	Files  []*FileGenerator
	Schema *schema.Schema
}

func NewPackageGenerator(t tmpl.Factory, p *v1alpha1.Package) (*PackageGenerator, error) {
	out := &PackageGenerator{
		Proto: p,
		Files: make([]*FileGenerator, 0, len(p.GetFiles())),
	}

	var err error
	out.Schema, err = schema.NewSchema(p.GetSchema())
	if err != nil {
		return nil, err
	}

	for _, v := range p.GetFiles() {
		fi, err := NewFileGenerator(t, v)
		if err != nil {
			return nil, fmt.Errorf("file %s: %w", v.GetPath(), err)
		}

		out.Files = append(out.Files, fi)
	}

	return out, nil
}

func (p *PackageGenerator) Generate(d map[string]any) (*Package, error) {
	out := &Package{
		Proto: p.Proto,
		Files: make([]*File, 0, len(p.Files)),
	}

	for _, v := range p.Files {
		fi, err := v.Generate(d)
		if err != nil {
			return nil, fmt.Errorf("file %s: %w", v.Proto.GetPath(), err)
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
			return fmt.Errorf("failed to write file %s to %s: %w", v.Proto.GetPath(), v.Proto.GetTarget(), err)
		}
	}

	return nil
}
