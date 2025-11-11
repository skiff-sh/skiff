package commands

import (
	"context"

	"github.com/urfave/cli/v3"
)

func NewCommand(projectRoot string) *RootCommand {
	return &RootCommand{
		ProjectRoot: projectRoot,
	}
}

type CommandAction interface {
	Command() (*cli.Command, error)
}

var _ CommandAction = &RootCommand{}

type RootCommand struct {
	ProjectRoot string
}

func (r *RootCommand) Command() (*cli.Command, error) {
	cli.RootCommandHelpTemplate = `Name:
   {{template "helpNameTemplate" .}}

Usage:
   {{if .UsageText}}{{wrap .UsageText 3}}{{else}}{{.FullName}} {{if .VisibleFlags}}[global options]{{end}}{{if .VisibleCommands}} [command [command options]]{{end}}{{if .ArgsUsage}} {{.ArgsUsage}}{{else}}{{if .Arguments}} [arguments...]{{end}}{{end}}{{end}}{{if .Version}}{{if not .HideVersion}}

Version:
   {{.Version}}{{end}}{{end}}{{if .Description}}

Description:
   {{template "descriptionTemplate" .}}{{end}}
{{- if len .Authors}}

Commands:{{template "visibleCommandCategoryTemplate" .}}{{end}}{{if .VisibleFlagCategories}}

Global options:{{template "visibleFlagCategoryTemplate" .}}{{else if .VisibleFlags}}

Global options:{{template "visibleFlagTemplate" .}}{{end}}{{if .Copyright}}

Copyright:
   {{template "copyrightTemplate" .}}{{end}}
`

	cli.SubcommandHelpTemplate = `Name:
   {{template "helpNameTemplate" .}}

Usage:
   {{if .UsageText}}{{wrap .UsageText 3}}{{else}}{{.FullName}}{{if .VisibleCommands}} [command [command options]]{{end}}{{if .ArgsUsage}} {{.ArgsUsage}}{{else}}{{if .Arguments}} [arguments...]{{end}}{{end}}{{end}}{{if .Category}}

Category:
   {{.Category}}{{end}}{{if .Description}}

Description:
   {{template "descriptionTemplate" .}}{{end}}{{if .VisibleCommands}}

Commands:{{template "visibleCommandTemplate" .}}{{end}}{{if .VisibleFlagCategories}}

Options:{{template "visibleFlagCategoryTemplate" .}}{{else if .VisibleFlags}}

Options:{{template "visibleFlagTemplate" .}}{{end}}
`

	cli.CommandHelpTemplate = `Name:
   {{template "helpNameTemplate" .}}

Usage:
   {{if .UsageText}}{{wrap .UsageText 3}}{{else}}{{.FullName}}{{if .VisibleFlags}} [options]{{end}}{{if .VisibleCommands}} [command [command options]]{{end}}{{if .ArgsUsage}} {{.ArgsUsage}}{{else}}{{if .Arguments}} {{range .Arguments}}[{{.Usage}} {{.Value}}] {{end}}{{end}}{{end}}{{end}}{{if .Category}}

Category:
   {{.Category}}{{end}}{{if .Description}}

Description:
   {{template "descriptionTemplate" .}}{{end}}{{if .VisibleFlagCategories}}

Options:{{template "visibleFlagCategoryTemplate" .}}{{else if .VisibleFlags}}

Options:{{template "visibleFlagTemplate" .}}{{end}}{{if .VisiblePersistentFlags}}

Global options:{{template "visiblePersistentFlagTemplate" .}}{{end}}
`

	cmd := &cli.Command{
		Name:  "skiff",
		Usage: "Share and reuse code in an LLM-friendly way.",
		Commands: []*cli.Command{
			{
				Name:  "build",
				Usage: "Build packages for a registry.",
				Flags: []cli.Flag{
					BuildOutputDirectoryFlag,
				},
				Arguments: []cli.Argument{
					BuildRegistryPathArg,
				},
				Action: func(ctx context.Context, command *cli.Command) error {
					bc := NewBuildCommand()
					return bc.Build(ctx, &BuildArgs{
						OutputDirectory: command.String(BuildOutputDirectoryFlag.Name),
						RegistryPath:    command.StringArg(BuildRegistryPathArg.Name),
					})
				},
			},
		},
	}

	return cmd, nil
}
