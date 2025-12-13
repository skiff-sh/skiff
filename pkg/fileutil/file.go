package fileutil

import (
	"fmt"
	"io/fs"
	"os"
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
			//nolint:nilerr // only testing.
			return nil
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
