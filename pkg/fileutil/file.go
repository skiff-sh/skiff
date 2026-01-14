package fileutil

import (
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/skiff-sh/skiff/pkg/system"
)

const (
	DefaultFileMode = 0o644
	DefaultDirMode  = 0o755
)

// FindSibling recursively searches upwards from the "from" parameter until a sibling file of "target" is found. If the "target"
// file is found, the fullfile path of the "target" file is returned, otherwise, an error is returned. If the
// "target" cannot be found, an error is returned. If the root of the filesystem is reached, an error is returned.
func FindSibling(from, target string) (string, error) {
	var err error
	ogFrom := from
	from, err = filepath.Abs(from)
	if err != nil {
		return "", err
	}
	sep := string(filepath.Separator)
	for from != sep && from != "" {
		targetPath := filepath.Join(from, target)
		if Exists(targetPath) {
			return targetPath, nil
		}
		from, _ = filepath.Split(filepath.Clean(from))
	}
	return "", fmt.Errorf("failed to find sibling %s in %s", target, ogFrom)
}

func Exists(fp string) bool {
	_, err := os.Stat(fp)
	return err == nil
}

func ExistsFS(f fs.FS, fp string) bool {
	_, err := fs.Stat(f, fp)
	return err == nil
}

func IsRel(root, fp string) bool {
	if filepath.IsAbs(fp) {
		return strings.HasPrefix(fp, root)
	}
	return true
}

func CallerPath(skip int) string {
	_, file, _, _ := runtime.Caller(skip)
	return file
}

// SplitFilename splits a filename into base name and extension.
// For example: "asd.txt" → ("asd", "txt").
func SplitFilename(filename string) (base, ext string) {
	ext = filepath.Ext(filename) // e.g. ".txt"
	base = strings.TrimSuffix(filename, ext)

	// Remove leading dot from extension, if present
	ext = strings.TrimPrefix(ext, ".")
	return base, ext
}

// MustAbs same as Abs but panics if an error is encountered.
func MustAbs(fp string) string {
	a, err := Abs(fp)
	if err != nil {
		panic(err)
	}
	return a
}

// Abs ensures fp is an absolute path. Uses the system.CWD variable (if set).
func Abs(fp string) (string, error) {
	if filepath.IsAbs(fp) {
		return fp, nil
	}

	wd, err := system.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Join(wd, fp), nil
}

func IsHTTPPath(p string) bool {
	return strings.HasPrefix(p, "http") || strings.HasPrefix(p, "https")
}

func NewPath(ha string) *Path {
	var u *url.URL
	if IsHTTPPath(ha) {
		u, _ = url.Parse(ha)
	}
	return &Path{
		URL:  u,
		Path: ha,
	}
}

type Path struct {
	URL  *url.URL
	Path string
}

func (p *Path) Join(s ...string) *Path {
	var u *url.URL
	var pa string
	if p.URL != nil {
		u = &url.URL{}
		u = u.JoinPath(s...)
	} else {
		pa = filepath.Join(append([]string{p.Path}, s...)...)
	}
	return &Path{
		URL:  u,
		Path: pa,
	}
}

func (p *Path) Dir() *Path {
	var u *url.URL
	var pa string
	if p.URL != nil {
		u = &(*p.URL)
		u.Path = path.Dir(u.Path)
	} else {
		pa = filepath.Dir(p.Path)
	}
	return &Path{
		URL:  u,
		Path: pa,
	}
}

func (p *Path) Ext() string {
	if p.URL != nil {
		return path.Ext(p.URL.Path)
	}
	return filepath.Ext(p.Path)
}

func (p *Path) Base() string {
	if p.URL != nil {
		return path.Base(p.URL.Path)
	}
	return filepath.Base(p.Path)
}

func (p *Path) String() string {
	if p.URL != nil {
		return p.URL.String()
	}
	return p.Path
}

func (p *Path) Scheme() string {
	if p.URL != nil {
		return p.URL.Scheme
	}
	return "file"
}

func (p *Path) EditPath(f func(s string) string) *Path {
	if p.URL != nil {
		p.Path = f(p.Path)
	} else {
		p.Path = f(p.Path)
	}
	return p
}
