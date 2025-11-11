package registry

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"strings"

	"github.com/skiff-sh/skiff/api/go/skiff/registry/v1alpha1"
	"github.com/skiff-sh/skiff/pkg/bufferpool"
	"github.com/skiff-sh/skiff/pkg/except"
	"github.com/skiff-sh/skiff/pkg/filesystem"
	"github.com/skiff-sh/skiff/pkg/fileutil"
	"github.com/skiff-sh/skiff/pkg/protoencode"
	"github.com/skiff-sh/skiff/pkg/tmpl"
	"github.com/skiff-sh/skiff/pkg/valid"
	"google.golang.org/protobuf/proto"
)

func IsHTTPPath(p string) bool {
	return strings.HasPrefix(p, "http") || strings.HasPrefix(p, "https")
}

func Load(r io.Reader) (*v1alpha1.Registry, error) {
	buf := bufferpool.GetBytesBuffer()
	defer bufferpool.PutBytesBuffer(buf)
	_, err := io.Copy(buf, r)
	if err != nil {
		return nil, err
	}

	reg := new(v1alpha1.Registry)
	err = protoencode.Unmarshaller.Unmarshal(buf.Bytes(), reg)
	return reg, err
}

type Loader interface {
	LoadRegistry(ctx context.Context) (*v1alpha1.Registry, error)
	LoadPackage(ctx context.Context, name string) (*PackageGenerator, error)
}

func NewFileLoader(tmplFact tmpl.Factory, f fs.FS, registryPath string) *FileLoader {
	return &FileLoader{
		RegistryRoot:    f,
		RegistryPath:    registryPath,
		TemplateFactory: tmplFact,
	}
}

func NewHTTPLoader(tmplFact tmpl.Factory, cl *http.Client, pa string) *HTTPLoader {
	return &HTTPLoader{
		Client:          cl,
		RootPath:        pa,
		TemplateFactory: tmplFact,
	}
}

var _ Loader = (*FileLoader)(nil)

type FileLoader struct {
	// The directory housing the Registry file.
	RegistryRoot fs.FS
	// The path to the registry file within the RegistryRoot.
	RegistryPath    string
	TemplateFactory tmpl.Factory
}

func (f *FileLoader) LoadRegistry(_ context.Context) (*v1alpha1.Registry, error) {
	fi, err := f.RegistryRoot.Open(f.RegistryPath)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = fi.Close()
	}()

	return Load(fi)
}

func (f *FileLoader) LoadPackage(ctx context.Context, name string) (*PackageGenerator, error) {
	reg, err := f.LoadRegistry(ctx)
	if err != nil {
		return nil, err
	}

	for _, v := range reg.Packages {
		if v.Name != name {
			continue
		}

		hydrated, err := f.hydratePackage(v)
		if err != nil {
			return nil, err
		}

		return NewPackageGenerator(f.TemplateFactory, hydrated)
	}
	return nil, except.ErrNotFound
}

// Populates the contents of all files in the package.
func (f *FileLoader) hydratePackage(p *v1alpha1.Package) (*v1alpha1.Package, error) {
	var err error
	hydrated := proto.CloneOf(p)
	for _, v := range hydrated.Files {
		if len(v.Content) > 0 {
			continue
		}

		var p string

		// Check if this registry has been deployed or not. Both should be supported for now.
		if jsonName := hydrated.Name + ".json"; fileutil.ExistsFS(f.RegistryRoot, jsonName) {
			p = jsonName
		} else {
			p = v.Path
		}

		v.Content, err = fs.ReadFile(f.RegistryRoot, p)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", p, err)
		}
	}

	return hydrated, nil
}

var _ Loader = (*HTTPLoader)(nil)

type HTTPLoader struct {
	Client          *http.Client
	RootPath        string
	TemplateFactory tmpl.Factory
}

func (h *HTTPLoader) LoadRegistry(ctx context.Context) (*v1alpha1.Registry, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.RootPath, nil)
	if err != nil {
		return nil, err
	}

	resp, err := h.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	return Load(resp.Body)
}

func (h *HTTPLoader) LoadPackage(ctx context.Context, name string) (*PackageGenerator, error) {
	pa, err := url.JoinPath(h.RootPath, name)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pa, nil)
	if err != nil {
		return nil, err
	}

	resp, err := h.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	buf := bufferpool.GetBytesBuffer()
	defer bufferpool.PutBytesBuffer(buf)
	_, err = io.Copy(buf, resp.Body)
	if err != nil {
		return nil, err
	}

	pkg := new(v1alpha1.Package)
	err = protoencode.Unmarshaller.Unmarshal(buf.Bytes(), pkg)
	if err != nil {
		return nil, err
	}

	return NewPackageGenerator(h.TemplateFactory, pkg)
}

func Validate(reg *v1alpha1.Registry, fsys filesystem.Filesystem) error {
	err := valid.ValidateProto(reg)
	if err != nil {
		return err
	}

	for _, pkg := range reg.Packages {
		for _, v := range pkg.Files {
			if _, err := fsys.AsRel(v.Target); err != nil {
				return fmt.Errorf("file %s in package %s has an invalid target %s: %w", v.Path, pkg.Name, v.Target, err)
			}
		}
	}

	return nil
}
