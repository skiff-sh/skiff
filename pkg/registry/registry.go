package registry

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/skiff-sh/api/go/skiff/registry/v1alpha1"
	"github.com/skiff-sh/skiff/pkg/bufferpool"
	"github.com/skiff-sh/skiff/pkg/filesystem"
	"github.com/skiff-sh/skiff/pkg/protoencode"
	"github.com/skiff-sh/skiff/pkg/tmpl"
	"github.com/skiff-sh/skiff/pkg/valid"
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
	LoadRegistry(ctx context.Context, path string) (*v1alpha1.Registry, error)
	LoadPackage(ctx context.Context, path string) (*PackageGenerator, error)
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

var _ Loader = (*FileLoader)(nil)

type FileLoader struct {
	TemplateFactory tmpl.Factory
}

func NewFileLoader(tmplFact tmpl.Factory) *FileLoader {
	return &FileLoader{
		TemplateFactory: tmplFact,
	}
}

func (f *FileLoader) LoadRegistry(_ context.Context, path string) (*v1alpha1.Registry, error) {
	fi, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = fi.Close()
	}()

	return Load(fi)
}

func (f *FileLoader) LoadPackage(_ context.Context, path string) (*PackageGenerator, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	pkg := new(v1alpha1.Package)
	err = protoencode.Unmarshaller.Unmarshal(b, pkg)
	if err != nil {
		return nil, err
	}

	return NewPackageGenerator(f.TemplateFactory, pkg)
}

var _ Loader = (*HTTPLoader)(nil)

type HTTPLoader struct {
	Client          HTTPClient
	TemplateFactory tmpl.Factory
}

func NewHTTPLoader(tmplFact tmpl.Factory, cl HTTPClient) *HTTPLoader {
	return &HTTPLoader{
		Client:          cl,
		TemplateFactory: tmplFact,
	}
}

func (h *HTTPLoader) LoadRegistry(ctx context.Context, path string) (*v1alpha1.Registry, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, path, nil)
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

func (h *HTTPLoader) LoadPackage(ctx context.Context, path string) (*PackageGenerator, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, path, nil)
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

func ValidateRegistry(reg *v1alpha1.Registry, fsys filesystem.Filesystem) error {
	err := valid.ValidateProto(reg)
	if err != nil {
		return err
	}

	for _, pkg := range reg.GetPackages() {
		for _, v := range pkg.GetFiles() {
			if _, err := fsys.AsRel(v.GetTarget()); err != nil {
				return fmt.Errorf(
					"file %s in package %s has an invalid target %s: %w",
					v.GetPath(),
					pkg.GetName(),
					v.GetTarget(),
					err,
				)
			}
		}
	}

	return nil
}

func ValidatePackage(pkg *Package, fsys filesystem.Filesystem) error {
	err := valid.ValidateProto(pkg.Proto)
	if err != nil {
		return err
	}

	for _, v := range pkg.Files {
		if _, err := fsys.AsRel(v.Proto.GetTarget()); err != nil {
			return fmt.Errorf(
				"file %s in package %s has an invalid target %s: %w",
				v.Proto.GetPath(),
				pkg.Proto.GetName(),
				v.Proto.GetTarget(),
				err,
			)
		}
	}

	return nil
}
