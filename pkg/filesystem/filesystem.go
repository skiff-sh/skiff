package filesystem

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/skiff-sh/skiff/pkg/fileutil"
)

type Filesystem interface {
	fs.FS
	fs.ReadFileFS
	fs.StatFS
	// ReadFileIn same as fs.ReadFile but reads into a buffer rather than into a slice of bytes.
	ReadFileIn(name string, buff io.Writer) error
	// WriteFile writes a file to the project root. If name is absolute and not within the project root, an error is returned. Automatically creates directories for file recursively.
	WriteFile(name string, content []byte) error

	WriteFileFrom(name string, content io.Reader) error

	// AsRel returns the enforced relative path to the root. If name is absolute and not within the path, an error is returned.
	AsRel(name string) (string, error)

	Exists(name string) bool

	Abs(name string) (string, error)

	MkdirAll(name string, mode fs.FileMode) error
	Chmod(name string, mode fs.FileMode) error
	Chtimes(name string, atime time.Time, mtime time.Time) error
	OpenFile(name string, flag int, perm fs.FileMode) (*os.File, error)
	Remove(name string) error
	// Link same as os.Link. This method deviates from the others because it allows the oldname to be outside the scope of the Filesystem.
	Link(oldname, newname string) error
	// Symlink same as os.Symlink. oldname can be outside the scope of the Filesystem.
	Symlink(oldname, newname string) error
	Create(name string) (*os.File, error)
}

type WriterTo interface {
	WriteTo(fsys Filesystem) error
}

func New(fp string) Filesystem {
	return &fsys{
		RootP:  fileutil.MustAbs(fp),
		RootFS: os.DirFS(fp),
	}
}

type fsys struct {
	RootP  string
	RootFS fs.FS
}

func (f *fsys) WriteFileFrom(name string, content io.Reader) error {
	if v, ok := content.(interface{ Bytes() []byte }); ok {
		return f.WriteFile(name, v.Bytes())
	}

	fp, err := f.Abs(name)
	if err != nil {
		return err
	}

	fi, err := os.OpenFile(fp, os.O_RDWR|os.O_CREATE|os.O_TRUNC, fileutil.DefaultFileMode)
	if err != nil {
		return err
	}
	defer func() {
		_ = fi.Close()
	}()

	_, err = io.Copy(fi, content)
	return err
}

func (f *fsys) ReadFileIn(name string, buff io.Writer) error {
	rel, err := f.AsRel(name)
	if err != nil {
		return err
	}

	fi, err := f.RootFS.Open(rel)
	if err != nil {
		return err
	}

	_, err = io.Copy(buff, fi)
	if err != nil {
		return err
	}

	return nil
}

func (f *fsys) Stat(name string) (fs.FileInfo, error) {
	rel, err := f.AsRel(name)
	if err != nil {
		return nil, err
	}
	return fs.Stat(f.RootFS, rel)
}

func (f *fsys) Create(name string) (*os.File, error) {
	fp, err := f.Abs(name)
	if err != nil {
		return nil, err
	}

	return os.Create(fp)
}

func (f *fsys) Symlink(oldname, newname string) error {
	fp, err := f.Abs(newname)
	if err != nil {
		return err
	}

	return os.Symlink(oldname, fp)
}

func (f *fsys) Link(oldname, newname string) error {
	fp, err := f.Abs(newname)
	if err != nil {
		return err
	}

	return os.Link(oldname, fp)
}

func (f *fsys) Remove(name string) error {
	fp, err := f.Abs(name)
	if err != nil {
		return err
	}

	return os.Remove(fp)
}

func (f *fsys) OpenFile(name string, flag int, perm fs.FileMode) (*os.File, error) {
	fp, err := f.Abs(name)
	if err != nil {
		return nil, err
	}

	return os.OpenFile(fp, flag, perm)
}

func (f *fsys) Chtimes(name string, atime time.Time, mtime time.Time) error {
	fp, err := f.Abs(name)
	if err != nil {
		return err
	}

	return os.Chtimes(fp, atime, mtime)
}

func (f *fsys) Chmod(name string, mode fs.FileMode) error {
	fp, err := f.Abs(name)
	if err != nil {
		return err
	}

	return os.Chmod(fp, mode)
}

func (f *fsys) MkdirAll(name string, mode fs.FileMode) error {
	fp, err := f.Abs(name)
	if err != nil {
		return err
	}
	return os.MkdirAll(fp, mode)
}

func (f *fsys) Abs(name string) (string, error) {
	rel, err := f.AsRel(name)
	if err != nil {
		return "", err
	}
	return filepath.Join(f.RootP, rel), nil
}

func (f *fsys) Exists(name string) bool {
	return fileutil.ExistsFS(f.RootFS, name)
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
	abs := filepath.Clean(name)
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(f.RootP, name)
	}

	if name == f.RootP {
		return abs, nil
	}

	rel, err := filepath.Rel(f.RootP, abs)
	if err != nil {
		return "", err
	}

	if strings.HasPrefix(rel, "..") || rel == ".." {
		return "", fmt.Errorf("relative to the project root %s", f.RootP)
	}

	return rel, nil
}

func (f *fsys) WriteFile(name string, content []byte) error {
	target, err := f.Abs(name)
	if err != nil {
		return err
	}

	_ = os.MkdirAll(filepath.Dir(target), fileutil.DefaultDirMode)

	var mode os.FileMode
	st, err := os.Stat(target)
	if err != nil {
		mode = fileutil.DefaultFileMode
	} else {
		mode = st.Mode()
	}

	return os.WriteFile(target, content, mode)
}
