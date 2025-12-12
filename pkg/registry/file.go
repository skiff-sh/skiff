package registry

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	pluginv1alpha1 "github.com/skiff-sh/api/go/skiff/plugin/v1alpha1"

	"github.com/skiff-sh/skiff/pkg/interact"
	"github.com/skiff-sh/skiff/pkg/plugin"
	"github.com/skiff-sh/skiff/pkg/schema"
	"github.com/skiff-sh/skiff/pkg/system"

	"github.com/skiff-sh/api/go/skiff/registry/v1alpha1"

	"github.com/skiff-sh/skiff/pkg/bufferpool"
	"github.com/skiff-sh/skiff/pkg/filesystem"
	"github.com/skiff-sh/skiff/pkg/tmpl"
)

type PackageFile struct {
	// The generator for the files contents.
	Renderer ContentRenderer

	// The target path for the file.
	Target tmpl.Template

	// The raw contents of the file.
	Contents []byte

	// The file housing the contents of the file. This is often the same as File except when it references another file within the package.
	SourceFile *v1alpha1.File

	// The original file.
	File *v1alpha1.File

	// The package that this file belongs to.
	Package *v1alpha1.Package
}

func NewPackageFile(
	ctx context.Context,
	compiler plugin.Compiler,
	sys system.System,
	tmplFact tmpl.Factory,
	pkg *v1alpha1.Package,
	v *v1alpha1.File,
) (*PackageFile, error) {
	src, srcFile, err := resolveSource(pkg, v)
	if err != nil {
		return nil, err
	}

	var fi ContentRenderer
	switch v.GetType() {
	case v1alpha1.File_file:
		fi, err = NewTemplateFileContentRenderer(tmplFact, src)
	case v1alpha1.File_plugin:
		fi, err = NewPluginContentRenderer(ctx, src, compiler, sys)
	default:
		return nil, fmt.Errorf("%s is not a valid file type", v.GetType().String())
	}
	if err != nil {
		return nil, err
	}

	target, err := tmplFact.NewTemplate([]byte(v.GetTarget()))
	if err != nil {
		return nil, fmt.Errorf("target path is invalid: %w", err)
	}

	return &PackageFile{
		Renderer:   fi,
		Target:     target,
		Contents:   src,
		SourceFile: srcFile,
		File:       v,
		Package:    pkg,
	}, nil
}

func (p *PackageFile) GenerateFile(ctx context.Context, d schema.PackageDataSource) (*File, error) {
	out := &File{
		SourcePath: p.File.GetPath(),
	}

	var err error
	buff := bufferpool.GetBytesBuffer()
	defer bufferpool.PutBytesBuffers(buff)

	err = p.Target.Render(d.RawData(), buff)
	if err != nil {
		return nil, fmt.Errorf("failed to render file path: %w", err)
	}
	out.Path = buff.String()

	c := &ContentRenderContext{
		Ctx:     ctx,
		Package: p.Package,
		Target:  out.Path,
		Path:    p.File.GetPath(),
	}

	out.Content, err = p.Renderer.RenderContent(c, d)
	if err != nil {
		return nil, fmt.Errorf("failed to render file contents: %w", err)
	}

	return out, nil
}

type ContentRenderContext struct {
	Ctx context.Context

	// The package housing the file.
	Package *v1alpha1.Package

	// The rendered target of the file.
	Target string

	// The path to the file
	Path string
}

type ContentRenderer interface {
	RenderContent(c *ContentRenderContext, d schema.PackageDataSource) ([]byte, error)
}

var _ ContentRenderer = (*TemplateContentRenderer)(nil)

type TemplateContentRenderer struct {
	Content tmpl.Template
}

func NewTemplateFileContentRenderer(t tmpl.Factory, src []byte) (*TemplateContentRenderer, error) {
	out := &TemplateContentRenderer{}

	var err error
	out.Content, err = t.NewTemplate(src)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (f *TemplateContentRenderer) RenderContent(_ *ContentRenderContext, d schema.PackageDataSource) ([]byte, error) {
	// Not using the pool because the bytes must be held after this method returns.
	buff := bytes.NewBuffer(nil)
	err := f.Content.Render(d.RawData(), buff)
	if err != nil {
		return nil, fmt.Errorf("failed to render 'contents' template: %w", err)
	}

	return buff.Bytes(), nil
}

var _ ContentRenderer = (*PluginContentRenderer)(nil)

type PluginContentRenderer struct {
	Plugin plugin.Plugin
}

func NewPluginContentRenderer(
	ctx context.Context,
	src []byte,
	compiler plugin.Compiler,
	sys system.System,
) (*PluginContentRenderer, error) {
	out := &PluginContentRenderer{}

	var err error
	out.Plugin, err = compiler.Compile(ctx, src, plugin.CompileOpts{
		CWDPath: sys.CWD(),
	})
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (p *PluginContentRenderer) RenderContent(c *ContentRenderContext, d schema.PackageDataSource) ([]byte, error) {
	resp, err := p.Plugin.SendRequest(c.Ctx, &pluginv1alpha1.Request{
		Metadata: &pluginv1alpha1.RequestMetadata{
			Package: c.Package.GetName(),
			Target:  c.Target,
			Path:    c.Path,
		},
		Data:      d.PluginData(),
		WriteFile: &pluginv1alpha1.WriteFileRequest{},
	})
	if err != nil {
		return nil, fmt.Errorf("%w\nLogs:\n%s", err, string(resp.Logs()))
	}

	var errs []error
	for _, v := range resp.Body.Issues {
		switch v.Level {
		case pluginv1alpha1.IssueLevel_LEVEL_WARN:
			interact.Warnf("Plugin %s: %s", c.Path, v.Message)
		case pluginv1alpha1.IssueLevel_LEVEL_ERROR:
			errs = append(errs, errors.New(v.Message))
		case pluginv1alpha1.IssueLevel_LEVEL_UNSPECIFIED:
			continue
		}
	}
	if len(errs) > 0 {
		return nil, fmt.Errorf("\n%w\nLogs:\n%s", errors.Join(errs...), string(resp.Logs()))
	}

	var content []byte
	if resp.Body.WriteFile != nil {
		content = resp.Body.WriteFile.Contents
	}

	return content, nil
}

type File struct {
	Path       string
	SourcePath string
	Content    []byte
}

func (f *File) WriteTo(fsys filesystem.Filesystem) error {
	return fsys.WriteFile(f.Path, f.Content)
}

func resolveSource(pkg *v1alpha1.Package, v *v1alpha1.File) (raw []byte, srcFile *v1alpha1.File, err error) {
	if v.GetSource() == nil {
		return nil, nil, fmt.Errorf("file %s is missing the source", v.GetPath())
	}

	switch {
	case v.Source.Text != nil:
		return []byte(v.GetSource().GetText()), v, nil
	case len(v.GetSource().GetRaw()) > 0:
		return v.GetSource().GetRaw(), v, nil
	case v.Source.FileIndex != nil:
		idx := int(v.GetSource().GetFileIndex())
		if idx < 0 || idx > len(pkg.GetFiles()) {
			return nil, nil, fmt.Errorf("references non-existent file index %d", idx)
		}
		fi := pkg.GetFiles()[idx]

		if len(fi.GetSource().GetRaw()) == 0 {
			return nil, nil, fmt.Errorf("references empty file %s", fi.GetPath())
		}
		return fi.GetSource().GetRaw(), fi, nil
	}

	return nil, nil, fmt.Errorf("file %s is missing a source", v.GetPath())
}
