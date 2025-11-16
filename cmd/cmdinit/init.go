package cmdinit

import (
	"log/slog"

	skiffconfig "github.com/skiff-sh/config"
	"github.com/skiff-sh/skiff/cmd/config"
	"github.com/skiff-sh/skiff/pkg/commands"
)

func NewCommand() (*commands.RootCommand, error) {
	conf, err := config.NewConfig()
	if err != nil {
		return nil, err
	}

	logger, err := skiffconfig.NewLogger(conf.Log)
	if err != nil {
		return nil, err
	}

	slog.SetDefault(logger)

	return commands.NewCommand(conf.Root), nil
}
