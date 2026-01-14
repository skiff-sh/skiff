package registry

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/skiff-sh/api/go/skiff/registry/v1alpha1"
	"google.golang.org/protobuf/proto"

	"github.com/skiff-sh/skiff/pkg/filesystem"
	"github.com/skiff-sh/skiff/pkg/protoencode"
	"github.com/skiff-sh/skiff/pkg/valid"
)

type Fetcher interface {
	Fetch(ctx context.Context, pa string) (io.ReadCloser, error)
	LoadRegistry(ctx context.Context, path string) (*v1alpha1.Registry, error)
	LoadPackage(ctx context.Context, path string) (*v1alpha1.Package, error)
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

var _ Fetcher = (*FileFetcher)(nil)

type FileFetcher struct {
}

func (f *FileFetcher) Fetch(_ context.Context, pa string) (io.ReadCloser, error) {
	fi, err := os.Open(pa)
	if err != nil {
		return nil, err
	}
	return fi, nil
}

func NewFileFetcher() *FileFetcher {
	return &FileFetcher{}
}

func (f *FileFetcher) LoadRegistry(_ context.Context, path string) (*v1alpha1.Registry, error) {
	reg := new(v1alpha1.Registry)
	err := protoencode.LoadFile(path, reg)
	return reg, err
}

func (f *FileFetcher) LoadPackage(_ context.Context, path string) (*v1alpha1.Package, error) {
	reg := new(v1alpha1.Package)
	err := protoencode.LoadFile(path, reg)
	return reg, err
}

var _ Fetcher = (*HTTPFetcher)(nil)

type HTTPFetcher struct {
	Client HTTPClient
}

func (h *HTTPFetcher) Fetch(ctx context.Context, pa string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pa, nil)
	if err != nil {
		return nil, err
	}

	resp, err := h.Client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

func NewHTTPFetcher(cl HTTPClient) *HTTPFetcher {
	return &HTTPFetcher{
		Client: cl,
	}
}

func (h *HTTPFetcher) LoadRegistry(ctx context.Context, path string) (*v1alpha1.Registry, error) {
	msg := new(v1alpha1.Registry)
	err := h.load(ctx, path, msg)
	return msg, err
}

func (h *HTTPFetcher) LoadPackage(ctx context.Context, path string) (*v1alpha1.Package, error) {
	msg := new(v1alpha1.Package)
	err := h.load(ctx, path, msg)
	return msg, err
}

func (h *HTTPFetcher) load(ctx context.Context, path string, p proto.Message) error {
	reader, err := h.Fetch(ctx, path)
	if err != nil {
		return err
	}
	defer func() {
		_ = reader.Close()
	}()

	return protoencode.Load(reader, p)
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
