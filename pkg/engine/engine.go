package engine

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
	cmdv1alpha1 "github.com/skiff-sh/api/go/skiff/cmd/v1alpha1"
	"github.com/skiff-sh/api/go/skiff/registry/v1alpha1"

	"github.com/skiff-sh/skiff/pkg/fileutil"

	"github.com/skiff-sh/skiff/pkg/bufferpool"

	"github.com/skiff-sh/skiff/pkg/filesystem"

	"github.com/skiff-sh/skiff/pkg/schema"

	"github.com/skiff-sh/skiff/pkg/system"

	"github.com/skiff-sh/skiff/pkg/registry"

	"github.com/skiff-sh/skiff/pkg/plugin"
	"github.com/skiff-sh/skiff/pkg/tmpl"
)

type Engine interface {
	AddPackage(
		ctx context.Context,
		pkg *registry.CompiledPackage,
		data schema.PackageDataSource,
	) (*AddPackageResult, error)

	ListPackages(ctx context.Context, registry string) ([]*cmdv1alpha1.ListPackagesResponse_PackagePreview, error)

	ViewPackage(ctx context.Context, name string) (*v1alpha1.Package, error)
}

type ListPackagesRequest struct {
	Registries []string
}

type AddPackageResult struct {
	Diffs []*Diff
}

type Diff struct {
	UnifiedDiff []byte
	File        *registry.File
}

func New(tmplFact tmpl.Factory, comp plugin.Compiler, sys system.System, project filesystem.Filesystem) Engine {
	out := &engine{
		TemplateFactory: tmplFact,
		Compiler:        comp,
		System:          sys,
		Project:         project,
	}

	return out
}

type engine struct {
	TemplateFactory tmpl.Factory
	Compiler        plugin.Compiler
	System          system.System
	Project         filesystem.Filesystem
}

func (e *engine) AddPackage(
	ctx context.Context,
	pkg *registry.CompiledPackage,
	data schema.PackageDataSource,
) (*AddPackageResult, error) {
	generator, err := registry.NewPackageGenerator(ctx, e.Compiler, e.System, e.TemplateFactory, pkg)
	if err != nil {
		return nil, err
	}
	defer generator.Close()

	gen, err := generator.Generate(ctx, data)
	if err != nil {
		return nil, err
	}

	resp := &AddPackageResult{
		Diffs: make([]*Diff, 0, len(gen.Files)),
	}

	buff := bufferpool.GetBytesBuffer()
	defer bufferpool.PutBytesBuffers(buff)
	for i, fi := range gen.Files {
		var before string
		if e.Project.Exists(fi.Path) {
			err := e.Project.ReadFileIn(fi.Path, buff)
			if err != nil {
				return nil, fmt.Errorf("file %s: %w", fi.Path, err)
			}

			before = buff.String()
		}

		edits := myers.ComputeEdits(span.URIFromPath(fi.Path), before, fi.Content.String())
		uni := gotextdiff.ToUnified(fi.Path, fi.Path, before, edits)
		diff := &Diff{
			UnifiedDiff: fmt.Append(nil, uni),
			File:        gen.Files[i],
		}
		resp.Diffs = append(resp.Diffs, diff)
		buff.Reset()
	}

	return resp, nil
}

func (e *engine) ListPackages(
	ctx context.Context,
	registryPath string,
) ([]*cmdv1alpha1.ListPackagesResponse_PackagePreview, error) {
	reg, err := registry.LoadRegistry(ctx, registryPath)
	if err != nil {
		return nil, fmt.Errorf("registry %s not found: %w", registryPath, err)
	}

	ha := fileutil.NewPath(registryPath)

	out := make([]*cmdv1alpha1.ListPackagesResponse_PackagePreview, 0, len(reg.GetPackages()))
	for _, pkg := range reg.GetPackages() {
		sch, err := schema.New(pkg.GetSchema())
		if err != nil {
			return nil, fmt.Errorf("package %s contains invalid schema: %w", pkg.GetName(), err)
		}

		js, err := schema.NewJSONSchema(sch)
		if err != nil {
			return nil, fmt.Errorf("package %s contains invalid schema: %w", pkg.GetName(), err)
		}

		b, _ := json.Marshal(js)

		out = append(out, &cmdv1alpha1.ListPackagesResponse_PackagePreview{
			Name:        pkg.GetName(),
			Registry:    reg.GetName(),
			Description: pkg.GetDescription(),
			Path:        ha.Dir().Join(pkg.GetName() + ".json").String(),
			JsonSchema:  string(b),
		})
	}

	return out, nil
}

func (e *engine) ViewPackage(ctx context.Context, pkg string) (*v1alpha1.Package, error) {
	p, err := registry.NewFetcher(pkg).LoadPackage(ctx, pkg)
	if err != nil {
		return nil, err
	}

	return p, nil
}
