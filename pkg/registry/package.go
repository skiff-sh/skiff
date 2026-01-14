package registry

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/skiff-sh/api/go/skiff/registry/v1alpha1"
	"google.golang.org/protobuf/proto"

	"github.com/skiff-sh/skiff/pkg/protoencode"

	"github.com/skiff-sh/skiff/pkg/bufferpool"

	"github.com/skiff-sh/skiff/pkg/fileutil"

	"github.com/skiff-sh/skiff/pkg/filesystem"
	"github.com/skiff-sh/skiff/pkg/plugin"
	"github.com/skiff-sh/skiff/pkg/schema"
	"github.com/skiff-sh/skiff/pkg/system"
	"github.com/skiff-sh/skiff/pkg/tmpl"
)

func LoadPackage(ctx context.Context, pa string) (*CompiledPackage, error) {
	p := new(v1alpha1.Package)
	err := Fetch(ctx, pa, p)
	if err != nil {
		return nil, err
	}

	return CompilePackage(pa, p)
}

// CompilePackage constructor for CompiledPackage.
func CompilePackage(srcPath string, out *v1alpha1.Package) (*CompiledPackage, error) {
	sch, err := schema.New(out.GetSchema())
	if err != nil {
		return nil, fmt.Errorf("invalid schema: %w", err)
	}

	js, err := schema.NewJSONSchema(sch)
	if err != nil {
		return nil, fmt.Errorf("failed to convert schema to JSON schema: %w", err)
	}

	res, err := js.Resolve(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to convert schema to JSON schema: %w", err)
	}

	return &CompiledPackage{
		Proto:      out,
		SourcePath: srcPath,
		Schema:     sch,
		JSONSchema: res,
	}, nil
}

func Fetch(ctx context.Context, p string, msg proto.Message) error {
	r, err := NewFetcher(p).Fetch(ctx, p)
	if err != nil {
		return err
	}
	defer func() {
		_ = r.Close()
	}()

	return protoencode.Load(r, msg)
}

type CompiledPackage struct {
	Proto      *v1alpha1.Package
	SourcePath string
	Schema     *schema.Schema
	JSONSchema *jsonschema.Resolved
}

func LoadRegistry(ctx context.Context, reg string) (*v1alpha1.Registry, error) {
	return NewFetcher(reg).LoadRegistry(ctx, reg)
}

func (l *CompiledPackage) CLIFlags(required bool) ([]*schema.Flag, error) {
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

func NewFetcher(u string) Fetcher {
	if fileutil.IsHTTPPath(u) {
		cl := &http.Client{
			Timeout: 1 * time.Second,
		}
		return NewHTTPFetcher(cl)
	}
	return NewFileFetcher()
}

func NewPackageGenerator(
	ctx context.Context,
	compiler plugin.Compiler,
	sys system.System,
	t tmpl.Factory,
	p *CompiledPackage,
) (*PackageGenerator, error) {
	out := &PackageGenerator{
		Package: p,
		Files:   make([]*PackageFile, 0, len(p.Proto.GetFiles())),
	}

	for _, v := range p.Proto.GetFiles() {
		fi, err := NewPackageFile(ctx, p.SourcePath, compiler, sys, t, p.Proto, v)
		if err != nil {
			return nil, fmt.Errorf("file %s: %w", v.GetPath(), err)
		}

		out.Files = append(out.Files, fi)
	}

	return out, nil
}

type PackageGenerator struct {
	Package *CompiledPackage
	Files   []*PackageFile
}

func (g *PackageGenerator) Close() {
	for _, v := range g.Files {
		bufferpool.PutBytesBuffer(v.Contents)
	}
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
