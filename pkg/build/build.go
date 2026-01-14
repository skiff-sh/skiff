package build

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"unicode/utf8"

	"github.com/skiff-sh/api/go/skiff/registry/v1alpha1"
	"github.com/skiff-sh/config/contexts"
	"github.com/skiff-sh/config/ptr"

	"github.com/skiff-sh/skiff/pkg/registry"

	"github.com/skiff-sh/skiff/pkg/fileutil"

	"github.com/skiff-sh/skiff/pkg/protoencode"

	"github.com/skiff-sh/skiff/pkg/bufferpool"

	"github.com/skiff-sh/skiff/pkg/plugin"

	"github.com/skiff-sh/skiff/pkg/filesystem"
)

type Builder interface {
	Build(ctx context.Context, registryRoot filesystem.Filesystem, pkg *v1alpha1.Package) (*Package, error)
}

type Package struct {
	Proto   *v1alpha1.Package
	Plugins []*Plugin
}

func (p *Package) WriteTo(fs filesystem.Filesystem) error {
	if len(p.Plugins) > 0 {
		defer func() {
			for _, v := range p.Plugins {
				_ = v.Content.Close()
			}
		}()
	}

	for _, v := range p.Plugins {
		if v.File.GetSource().GetPath() == "" {
			return fmt.Errorf("plugin %s has no source path", v.File.GetPath())
		}
		err := fs.WriteFileFrom(v.File.GetSource().GetPath(), v.Content)
		if err != nil {
			return fmt.Errorf("failed to write plugin file for %s: %w", v.File.Path, err)
		}
	}

	b, err := protoencode.PrettyMarshaller.Marshal(p.Proto)
	if err != nil {
		return err
	}

	return fs.WriteFile(p.Proto.Name+".json", b)
}

type Plugin struct {
	File    *v1alpha1.File
	Content io.ReadCloser
}

type Hooks struct {
	BeforeBuild func(f *v1alpha1.File)
	OnBuild     func(f *v1alpha1.File)
}

func NewBuilder(prov ToolsProvider, hooks *Hooks) Builder {
	if hooks == nil {
		hooks = &Hooks{}
	}
	out := &builder{
		Tools: prov,
		Hooks: hooks,
	}
	return out
}

func NewToolsProvider(hooks *plugin.InstallHooks) ToolsProvider {
	return &toolsProvider{
		Hooks: hooks,
	}
}

type ToolsProvider interface {
	Get(ctx context.Context) (*Tools, error)
}

type Tools struct {
	Builder        plugin.Builder
	PluginCompiler plugin.Compiler
}

type toolsProvider struct {
	Err   error
	Tools *Tools
	Hooks *plugin.InstallHooks
}

func (t *toolsProvider) Get(ctx context.Context) (*Tools, error) {
	if t.Tools == nil && t.Err == nil {
		t.Tools, t.Err = newBuildTools(ctx, t.Hooks)
	}

	return t.Tools, t.Err
}

type builder struct {
	Tools ToolsProvider
	Hooks *Hooks
}

func (b *builder) Build(
	ctx context.Context,
	registryRoot filesystem.Filesystem,
	pkg *v1alpha1.Package,
) (*Package, error) {
	logger := contexts.GetLogger(ctx)
	out := &Package{
		Proto:   pkg,
		Plugins: make([]*Plugin, 0),
	}
	for i := range pkg.GetFiles() {
		v := pkg.Files[i]
		src := v.GetSource()
		if src == nil {
			src = &v1alpha1.File_Source{}
		}

		switch v.GetType() {
		case v1alpha1.File_file:
			content, err := fs.ReadFile(registryRoot, v.GetPath())
			if err != nil {
				return nil, fmt.Errorf("failed to read file %s: %w", v.GetPath(), err)
			}
			src.Text = ptr.Ptr(string(content))
		case v1alpha1.File_plugin:
			var content io.ReadCloser
			if src.Path != nil {
				pa := *src.Path
				logger.DebugContext(ctx, "Plugin already pre-built. Loading.", "path", pa)
				var err error
				content, err = registry.NewFetcher(pa).Fetch(ctx, pa)
				if err != nil {
					return nil, fmt.Errorf("failed to load plugin from %s for file %s: %w", pa, v.Path, err)
				}
			} else {
				tools, err := b.Tools.Get(ctx)
				if err != nil {
					return nil, fmt.Errorf("failed to build plugin %s: %w", v.Path, err)
				}

				abs, err := registryRoot.Abs(v.Path)
				if err != nil {
					return nil, fmt.Errorf("failed to build plugin %s: %w", v.Path, err)
				}

				if b.Hooks.BeforeBuild != nil {
					b.Hooks.BeforeBuild(v)
				}

				bu, err := b.buildPlugin(ctx, tools, abs)
				if err != nil {
					return nil, fmt.Errorf("failed to build plugin %s: %w", v.Path, err)
				}

				content = bufferpool.AsReadCloser(bu)
				if b.Hooks.OnBuild != nil {
					b.Hooks.OnBuild(v)
				}
			}
			multi := fileutil.NewPath(v.Path)
			base := multi.Base()
			if base == "." {
				base = v.Path
			}
			filename, _ := fileutil.SplitFilename(base)
			src.Path = ptr.Ptr(filepath.Join("plugins", filename+".wasm"))

			out.Plugins = append(out.Plugins, &Plugin{
				File:    v,
				Content: content,
			})
		}

		if src.Text != nil && !utf8.Valid([]byte(*src.Text)) {
			return nil, fmt.Errorf("%s contains invalid utf-8 text", v.Path)
		}

		v.Source = src
	}

	return out, nil
}

func (b *builder) buildPlugin(
	ctx context.Context,
	tools *Tools,
	absPath string,
) (*bytes.Buffer, error) {
	buff := bufferpool.GetBytesBuffer()
	err := tools.Builder.Build(ctx, absPath, buff)
	if err != nil {
		return nil, err
	}

	_, err = tools.PluginCompiler.Compile(ctx, buff.Bytes(), plugin.CompileOpts{})
	if err != nil {
		return nil, fmt.Errorf("failed to compile plugin %s: %w", absPath, err)
	}

	return buff, nil
}

func newBuildTools(ctx context.Context, hooks *plugin.InstallHooks) (*Tools, error) {
	out := &Tools{}
	var err error
	out.Builder, err = plugin.CreateOrInstallGoBuilder(ctx, hooks)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate Go CLI: %w", err)
	}

	out.PluginCompiler, err = plugin.NewWazeroCompiler()
	if err != nil {
		return out, fmt.Errorf("failed to create WASM compiler: %w", err)
	}

	return out, nil
}
