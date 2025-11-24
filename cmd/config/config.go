package config

import (
	"os"
	"path/filepath"

	"github.com/skiff-sh/config"
)

type Config struct {
	Log config.Log `koanf:"log"  yaml:"log"  json:"log"`
	// The root of the project. If not set, uses cwd.
	Root string `koanf:"root" yaml:"root" json:"root"`
}

func NewConfig() (*Config, error) {
	k := config.InitKoanf("skiff", Default())
	out := new(Config)
	err := k.Unmarshal("", out)
	if err != nil {
		return nil, err
	}

	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	if out.Root == "" {
		out.Root = wd
	} else if !filepath.IsAbs(out.Root) {
		out.Root = filepath.Join(wd, out.Root)
	}

	return out, nil
}

func Default() *Config {
	return &Config{
		Log: config.Log{
			Level:   "info",
			Outputs: "stdout",
		},
	}
}
