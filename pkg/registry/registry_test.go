package registry

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/skiff-sh/api/go/skiff/registry/v1alpha1"

	"github.com/skiff-sh/skiff/pkg/mocks/registrymocks"
	"github.com/skiff-sh/skiff/pkg/protoencode"
	"github.com/skiff-sh/skiff/pkg/testutil"
)

type RegistryTestSuite struct {
	suite.Suite
}

func (r *RegistryTestSuite) TestFileLoader() {
	type test struct {
		GivenPackage  *v1alpha1.Package
		GivenRegistry *v1alpha1.Registry
		ExpectedErr   string
	}

	tests := map[string]test{
		"file": {
			GivenRegistry: &v1alpha1.Registry{Name: "registry"},
			GivenPackage:  &v1alpha1.Package{Name: "package"},
		},
	}

	for desc, v := range tests {
		r.Run(desc, func() {
			rootDir, _ := os.MkdirTemp(os.TempDir(), "*")
			defer func() {
				_ = os.RemoveAll(rootDir)
			}()
			b, _ := protoencode.Marshal(v.GivenPackage)
			pkgPath := filepath.Join(rootDir, "package.json")
			_ = os.WriteFile(pkgPath, b, 0644)
			b, _ = protoencode.Marshal(v.GivenRegistry)
			regPath := filepath.Join(rootDir, "registry.json")
			_ = os.WriteFile(regPath, b, 0644)

			loader := NewFileLoader()
			actualPkg, err := loader.LoadPackage(r.T().Context(), pkgPath)
			if v.ExpectedErr != "" || !r.NoError(err) {
				r.ErrorContains(err, v.ExpectedErr)
				return
			}

			r.Empty(testutil.DiffProto(v.GivenPackage, actualPkg))

			actualReg, err := loader.LoadRegistry(r.T().Context(), regPath)
			if v.ExpectedErr != "" || !r.NoError(err) {
				r.ErrorContains(err, v.ExpectedErr)
				return
			}
			r.Empty(testutil.DiffProto(v.GivenRegistry, actualReg))
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
				return NewHTTPLoader(cl)
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
			r.Empty(testutil.DiffProto(v.ExpectedPackage, pkg))
		})
	}
}

func TestRegistryTestSuite(t *testing.T) {
	suite.Run(t, new(RegistryTestSuite))
}
