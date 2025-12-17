package artifact

import (
	"embed"
	"path/filepath"
	"testing"

	"github.com/skiff-sh/skiff/pkg/filesystem"
	"github.com/skiff-sh/skiff/pkg/testutil"
	"github.com/stretchr/testify/suite"
)

type ArchiveTestSuite struct {
	testutil.Suite
}

//go:embed testdata/*
var testdata embed.FS

func (a *ArchiveTestSuite) TestExtract() {
	type test struct {
		GivenName   string
		Archiver    Archiver
		Expected    testutil.MapFS
		ExpectedErr string
	}

	tests := map[string]test{
		"tar.gz": {
			GivenName: "source.tar.gz",
			Archiver:  new(TarGZArchiver),
			Expected: testutil.MapFS{
				"source/dir/another.txt":   {Data: []byte("another\n")},
				"source/derp.txt":          {Data: []byte("derp\n")},
				"source/link.txt":          {Data: []byte("link\n")},
				"source/dir/to-link.txt":   {Data: []byte("link\n")},
				"source/dir/hard-link.txt": {Data: []byte("link\n")},
				"source/hard-link.txt":     {Data: []byte("link\n")},
			},
		},
		"zip": {
			GivenName: "source.zip",
			Archiver:  new(ZipArchiver),
			Expected: testutil.MapFS{
				"source/dir/another.txt":   {Data: []byte("another\n")},
				"source/derp.txt":          {Data: []byte("derp\n")},
				"source/link.txt":          {Data: []byte("link\n")},
				"source/dir/to-link.txt":   {Data: []byte("link\n")},
				"source/dir/hard-link.txt": {Data: []byte("link\n")},
				"source/hard-link.txt":     {Data: []byte("link\n")},
			},
		},
	}

	for desc, v := range tests {
		a.Run(desc, func() {
			ctx := a.T().Context()

			to := filesystem.New(a.T().TempDir())

			f, err := testdata.Open(filepath.Join("testdata", v.GivenName))
			if !a.NoError(err) {
				return
			}
			defer func() {
				_ = f.Close()
			}()

			err = v.Archiver.Extract(ctx, f, to)
			if !a.NoErrorOrContains(err, v.ExpectedErr) {
				return
			}

			actual := testutil.FlatMapFS(to)
			a.Equal(v.Expected, actual)
		})
	}
}

func TestArchiveTestSuite(t *testing.T) {
	suite.Run(t, new(ArchiveTestSuite))
}
