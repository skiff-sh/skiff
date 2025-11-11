package cmdinit

import (
	"log/slog"

	skiffconfig "github.com/skiff-sh/config"
	"github.com/skiff-sh/skiff/cmd/config"
	"github.com/skiff-sh/skiff/pkg/commands"
	"github.com/urfave/cli/v3"
)

func NewCommand() (*cli.Command, error) {
	conf, err := config.NewConfig()
	if err != nil {
		return nil, err
	}

	logger, err := skiffconfig.NewLogger(conf.Log)
	if err != nil {
		return nil, err
	}

	slog.SetDefault(logger)

	cmd, err := commands.NewCommand(conf.Root).Command()
	if err != nil {
		return nil, err
	}

	return cmd, nil
}
