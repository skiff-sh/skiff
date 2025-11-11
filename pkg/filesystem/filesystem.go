package filesystem

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type Filesystem interface {
	// WriteFile writes a file to the project root. If name is absolute and not within the project root, an error is returned. Automatically creates directories for file recursively.
	WriteFile(name string, content []byte) error

	ReadFile(name string) ([]byte, error)

	Open(name string) (fs.File, error)

	// AsRel returns the enforced relative path to the root. If name is absolute and not within the path, an error is returned.
	AsRel(name string) (string, error)
}

type WriterTo interface {
	WriteTo(fsys Filesystem) error
}

func NewFilesystem(fp string, f fs.FS) Filesystem {
	return &fsys{
		RootP:  fp,
		RootFS: f,
	}
}

type fsys struct {
	RootP  string
	RootFS fs.FS
}

func (f *fsys) Open(name string) (fs.File, error) {
	rel, err := f.AsRel(name)
	if err != nil {
		return nil, err
	}
	return f.RootFS.Open(rel)
}

func (f *fsys) ReadFile(name string) ([]byte, error) {
	rel, err := f.AsRel(name)
	if err != nil {
		return nil, err
	}
	return fs.ReadFile(f.RootFS, rel)
}

func (f *fsys) AsRel(name string) (string, error) {
	if filepath.IsAbs(name) {
		if !strings.HasPrefix(name, f.RootP) {
			return "", fmt.Errorf("relative to the project root %s", f.RootP)
		}
	} else {
		name = filepath.Join(f.RootP, name)
	}

	o, err := filepath.Rel(f.RootP, name)
	if err != nil {
		return "", fmt.Errorf("not relative to project root %s", f.RootP)
	}

	return o, nil
}

func (f *fsys) WriteFile(name string, content []byte) error {
	rel, err := f.AsRel(name)
	if err != nil {
		return err
	}

	target := filepath.Join(f.RootP, rel)
	_ = os.MkdirAll(filepath.Dir(target), 0755)

	return os.WriteFile(target, content, 0644)
}
