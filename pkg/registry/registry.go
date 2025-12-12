package registry

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/skiff-sh/api/go/skiff/registry/v1alpha1"
	"google.golang.org/protobuf/proto"

	"github.com/skiff-sh/skiff/pkg/filesystem"
	"github.com/skiff-sh/skiff/pkg/protoencode"
	"github.com/skiff-sh/skiff/pkg/valid"
)

func IsHTTPPath(p string) bool {
	return strings.HasPrefix(p, "http") || strings.HasPrefix(p, "https")
}

type Loader interface {
	LoadRegistry(ctx context.Context, path string) (*v1alpha1.Registry, error)
	LoadPackage(ctx context.Context, path string) (*v1alpha1.Package, error)
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

var _ Loader = (*FileLoader)(nil)

type FileLoader struct {
}

func NewFileLoader() *FileLoader {
	return &FileLoader{}
}

func (f *FileLoader) LoadRegistry(_ context.Context, path string) (*v1alpha1.Registry, error) {
	reg := new(v1alpha1.Registry)
	err := protoencode.LoadFile(path, reg)
	return reg, err
}

func (f *FileLoader) LoadPackage(_ context.Context, path string) (*v1alpha1.Package, error) {
	reg := new(v1alpha1.Package)
	err := protoencode.LoadFile(path, reg)
	return reg, err
}

var _ Loader = (*HTTPLoader)(nil)

type HTTPLoader struct {
	Client HTTPClient
}

func NewHTTPLoader(cl HTTPClient) *HTTPLoader {
	return &HTTPLoader{
		Client: cl,
	}
}

func (h *HTTPLoader) LoadRegistry(ctx context.Context, path string) (*v1alpha1.Registry, error) {
	msg := new(v1alpha1.Registry)
	err := h.load(ctx, path, msg)
	return msg, err
}

func (h *HTTPLoader) LoadPackage(ctx context.Context, path string) (*v1alpha1.Package, error) {
	msg := new(v1alpha1.Package)
	err := h.load(ctx, path, msg)
	return msg, err
}

func (h *HTTPLoader) load(ctx context.Context, path string, p proto.Message) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, path, nil)
	if err != nil {
		return err
	}

	resp, err := h.Client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	return protoencode.Load(resp.Body, p)
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
		if _, err := fsys.AsRel(v.Path); err != nil {
			return fmt.Errorf(
				"file %s in package %s has an invalid target %s: %w",
				v.SourcePath,
				pkg.Proto.GetName(),
				v.Path,
				err,
			)
		}
	}

	return nil
}
