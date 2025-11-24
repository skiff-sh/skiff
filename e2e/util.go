package e2e

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/skiff-sh/skiff/cmd/cmdinit"
	"github.com/skiff-sh/skiff/pkg/commands"
	"github.com/skiff-sh/skiff/pkg/fileutil"
)

type CLI struct {
	Command *commands.RootCommand
}

func New() (*CLI, error) {
	cmd, err := cmdinit.NewCommand()
	if err != nil {
		return nil, err
	}

	return &CLI{Command: cmd}, nil
}

func ExamplesPath() string {
	return filepath.Join(filepath.Dir(filepath.Dir(fileutil.CallerPath(1))), "examples")
}

func CloneExample(examples fs.FS, dirName string) (string, error) {
	tmp, err := os.MkdirTemp(os.TempDir(), "*")
	if err != nil {
		return "", err
	}

	err = fs.WalkDir(examples, dirName, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		target := filepath.Join(tmp, path)
		_ = os.MkdirAll(filepath.Dir(target), 0755)

		b, err := fs.ReadFile(examples, path)
		if err != nil {
			return err
		}

		return os.WriteFile(target, b, 0644)
	})
	if err != nil {
		return "", err
	}

	return filepath.Join(tmp, dirName), nil
}
