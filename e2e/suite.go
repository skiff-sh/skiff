package e2e

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
