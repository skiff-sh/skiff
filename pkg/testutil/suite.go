package testutil

import (
	"fmt"
	"io/fs"

	"github.com/stretchr/testify/suite"

	"github.com/skiff-sh/skiff/pkg/system"
)

type Suite struct {
	suite.Suite
}

func (s *Suite) EqualFiles(e fs.FS, ePath string, a fs.FS, aPath string) bool {
	expected, err := fs.ReadFile(e, ePath)
	if !s.NoError(err) {
		return false
	}

	actual, err := fs.ReadFile(a, aPath)
	if !s.NoError(err) {
		return false
	}

	return s.Equal(string(expected), string(actual))
}

func (s *Suite) FileContains(f fs.FS, p, contains string) bool {
	return s.FileContainsAll(f, p, []string{contains})
}

func (s *Suite) FileContainsAll(f fs.FS, p string, contains []string) bool {
	fi, err := fs.ReadFile(f, p)
	if !s.NoError(err) {
		return false
	}

	content := string(fi)
	for _, v := range contains {
		if !s.Contains(content, v) {
			fmt.Println(content)
			return false
		}
	}

	return true
}

func (s *Suite) SetWd(dir string) func() {
	cwd, _ := system.Getwd()
	system.Setwd(dir)
	return func() {
		system.Setwd(cwd)
	}
}

// NoErrorOrContains checks that the error is nil. If contains is non-empty, it will check that the error exists and has
// contains. Returns false if there was an error.
func (s *Suite) NoErrorOrContains(err error, contains string) bool {
	if contains != "" || !s.NoError(err) {
		s.ErrorContains(err, contains)
		return false
	}
	return true
}

type File struct {
	Data  []byte
	IsDir bool
}

type MapFS map[string]File

// FlatMapFS converts a fs.FS into a map of flat paths to their contents. Similar to [fstest.MapFS].
func FlatMapFS(f fs.FS) MapFS {
	out := MapFS{}

	_ = fs.WalkDir(f, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		b, err := fs.ReadFile(f, path)
		if err != nil {
			//nolint:nilerr // only testing.
			return nil
		}

		out[path] = File{
			Data:  b,
			IsDir: d.IsDir(),
		}

		return nil
	})
	return out
}
