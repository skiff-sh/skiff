package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"path/filepath"

	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
	cmdv1alpha1 "github.com/skiff-sh/api/go/skiff/cmd/v1alpha1"
	"github.com/skiff-sh/api/go/skiff/registry/v1alpha1"

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
		pkg *registry.LoadedPackage,
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
	pkg *registry.LoadedPackage,
	data schema.PackageDataSource,
) (*AddPackageResult, error) {
	generator, err := registry.NewPackageGenerator(ctx, e.Compiler, e.System, e.TemplateFactory, pkg)
	if err != nil {
		return nil, err
	}

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

		edits := myers.ComputeEdits(span.URIFromPath(fi.Path), before, string(fi.Content))
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

	ha := newHandler(registryPath)

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
			Path:        ha.Dir().Join(pkg.GetName()).String(),
			JsonSchema:  string(b),
		})
	}

	return out, nil
}

func (e *engine) ViewPackage(ctx context.Context, pkg string) (*v1alpha1.Package, error) {
	p, err := registry.NewLoader(pkg).LoadPackage(ctx, pkg)
	if err != nil {
		return nil, err
	}

	for _, v := range p.GetFiles() {
		if len(v.GetSource().GetRaw()) > 0 {
			v.Source.Raw = nil
		}
	}

	return p, nil
}

func newHandler(ha string) *pathHandler {
	var u *url.URL
	if registry.IsHTTPPath(ha) {
		u, _ = url.Parse(ha)
	}
	return &pathHandler{
		URL:  u,
		Path: ha,
	}
}

type pathHandler struct {
	URL  *url.URL
	Path string
}

func (p *pathHandler) Join(s ...string) *pathHandler {
	var u url.URL
	var pa string
	if p.URL != nil {
		u = *u.JoinPath(s...)
	} else {
		pa = filepath.Join(append([]string{p.Path}, s...)...)
	}
	return &pathHandler{
		URL:  &u,
		Path: pa,
	}
}

func (p *pathHandler) Dir() *pathHandler {
	var u url.URL
	var pa string
	if p.URL != nil {
		u = *p.URL
		u.Path = path.Dir(u.Path)
	} else {
		pa = filepath.Dir(p.Path)
	}
	return &pathHandler{
		URL:  &u,
		Path: pa,
	}
}

func (p *pathHandler) String() string {
	return p.Path
}
