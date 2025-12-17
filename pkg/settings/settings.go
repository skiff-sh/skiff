package settings

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/skiff-sh/skiff/pkg/fileutil"
)

const (
	BuildDirName = "skiff"
)

// BuildDir directory housing all build tooling.
func BuildDir() (string, error) {
	return BuildDirFunc()
}

var BuildDirFunc = sync.OnceValues(func() (string, error) {
	var baseDir string
	var err error
	switch runtime.GOOS {
	case "windows":
		baseDir, err = os.UserCacheDir()
	default:
		baseDir, err = os.UserConfigDir()
	}
	if err != nil {
		return "", err
	}

	fp := filepath.Join(baseDir, BuildDirName)
	_ = os.MkdirAll(fp, fileutil.DefaultDirMode)

	return fp, nil
})
