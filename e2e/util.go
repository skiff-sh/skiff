package e2e

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/skiff-sh/skiff/cmd/cmdinit"
	"github.com/urfave/cli/v3"
)

type CLI struct {
	Command *cli.Command
}

func New() (*CLI, error) {
	cmd, err := cmdinit.NewCommand()
	if err != nil {
		return nil, err
	}

	return &CLI{Command: cmd}, nil
}

//go:embed all:examples/*
var examples embed.FS

func CloneExample(dirName string) (string, error) {
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
		_ = os.MkdirAll(target, 0755)

		b, err := fs.ReadFile(examples, path)
		if err != nil {
			return err
		}

		return os.WriteFile(target, b, 0644)
	})
	if err != nil {
		return "", err
	}

	return tmp, nil
}
