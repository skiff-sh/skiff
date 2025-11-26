package registry

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/skiff-sh/config/ptr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/skiff-sh/api/go/skiff/registry/v1alpha1"
	"github.com/skiff-sh/skiff/pkg/filesystem"
	"github.com/skiff-sh/skiff/pkg/mocks/registrymocks"
	"github.com/skiff-sh/skiff/pkg/protoencode"
	"github.com/skiff-sh/skiff/pkg/testutil"
	"github.com/skiff-sh/skiff/pkg/tmpl"
)

type RegistryTestSuite struct {
	suite.Suite
}

func (r *RegistryTestSuite) TestFileLoader() {
	type test struct {
		Package             *v1alpha1.Package
		GivenFS             fstest.MapFS
		GivenData           map[string]any
		GivenPath           string
		ExpectedFunc        func(fi filesystem.Filesystem)
		ExpectedErr         string
		ExpectedGenerateErr string
		ExpectedWriteToErr  string
	}

	tests := map[string]test{
		"file": {
			GivenData: map[string]any{
				"field": "hi",
			},
			ExpectedFunc: func(fi filesystem.Filesystem) {
				cont, err := fi.ReadFile("derp.txt")
				if !r.NoError(err) {
					return
				}

				r.Equal("derp hi", string(cont))
			},
			Package: &v1alpha1.Package{
				Name:        "pkg",
				Description: "description",
				Files: []*v1alpha1.File{
					{
						Target:  "derp.txt",
						Type:    v1alpha1.File_template,
						Content: ptr.Ptr(`derp {{.field}}`),
					},
				},
				Schema: &v1alpha1.Schema{
					Fields: []*v1alpha1.Field{{Name: "field", Type: ptr.Ptr(v1alpha1.Field_string)}},
				},
			},
		},
	}

	for desc, v := range tests {
		r.Run(desc, func() {
			rootDir, _ := os.MkdirTemp(os.TempDir(), "*")
			defer func() {
				_ = os.RemoveAll(rootDir)
			}()
			b, _ := protoencode.Marshaller.Marshal(v.Package)
			pkgPath := filepath.Join(rootDir, "package.json")
			_ = os.WriteFile(pkgPath, b, 0644)

			gen, err := NewFileLoader(tmpl.NewGoFactory()).LoadPackage(r.T().Context(), pkgPath)
			if v.ExpectedErr != "" || !r.NoError(err) {
				r.ErrorContains(err, v.ExpectedErr)
				return
			}

			pkg, err := gen.Generate(v.GivenData)
			if v.ExpectedGenerateErr != "" || !r.NoError(err) {
				r.ErrorContains(err, v.ExpectedGenerateErr)
				return
			}

			fs := filesystem.New(rootDir)
			err = pkg.WriteTo(fs)
			if v.ExpectedWriteToErr != "" || !r.NoError(err) {
				r.ErrorContains(err, v.ExpectedWriteToErr)
				return
			}

			v.ExpectedFunc(fs)
		})
	}
}

func (r *RegistryTestSuite) TestHTTPLoader() {
	type test struct {
		GivenFactory     func() Loader
		ExpectedPackage  *v1alpha1.Package
		ExpectedRegistry *v1alpha1.Registry
		PackagePath      string
		RegistryPath     string
	}

	tests := map[string]test{
		"http": {
			PackagePath:  "package.com",
			RegistryPath: "registry.com",
			GivenFactory: func() Loader {
				cl := new(registrymocks.HTTPClient)
				cl.EXPECT().Do(mock.MatchedBy(func(req *http.Request) bool {
					return req.URL.String() == "package.com"
				})).Return(&http.Response{Body: io.NopCloser(bytes.NewBufferString(`{"name": "package"}`))}, nil)

				cl.EXPECT().Do(mock.MatchedBy(func(req *http.Request) bool {
					return req.URL.String() == "registry.com"
				})).Return(&http.Response{Body: io.NopCloser(bytes.NewBufferString(`{"name": "registry"}`))}, nil)
				return NewHTTPLoader(tmpl.NewGoFactory(), cl)
			},
			ExpectedPackage:  &v1alpha1.Package{Name: "package"},
			ExpectedRegistry: &v1alpha1.Registry{Name: "registry"},
		},
	}

	for desc, v := range tests {
		r.Run(desc, func() {
			given := v.GivenFactory()
			pkg, err := given.LoadPackage(r.T().Context(), v.PackagePath)
			if !r.NoError(err) {
				return
			}

			reg, err := given.LoadRegistry(r.T().Context(), v.RegistryPath)
			if !r.NoError(err) {
				return
			}

			r.Empty(testutil.DiffProto(v.ExpectedRegistry, reg))
			r.Empty(testutil.DiffProto(v.ExpectedPackage, pkg.Proto))
		})
	}
}

func TestRegistryTestSuite(t *testing.T) {
	suite.Run(t, new(RegistryTestSuite))
}
