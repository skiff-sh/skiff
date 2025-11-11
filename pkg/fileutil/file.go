package fileutil

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// FindSibling recursively searches upwards from the "from" parameter until a sibling directory of "target" is found. If the "target"
// directory is found, the fullfile path of the "target" directory is returned, otherwise, an error is returned. If the
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
	return errors.Is(err, fs.ErrNotExist)
}

func ExistsFS(f fs.FS, fp string) bool {
	_, err := fs.Stat(f, fp)
	return errors.Is(err, fs.ErrNotExist)
}

func IsRel(root, fp string) bool {
	if filepath.IsAbs(fp) {
		return strings.HasPrefix(fp, root)
	}
	return true
}
