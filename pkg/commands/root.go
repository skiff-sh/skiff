package commands

import (
	"context"
	"errors"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/skiff-sh/skiff/pkg/collection"
	"github.com/skiff-sh/skiff/pkg/filesystem"
	"github.com/urfave/cli/v3"
)

func NewCommand(projectRoot string) *RootCommand {
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
   {{if .UsageText}}{{wrap .UsageText 3}}{{else}}{{.FullName}}{{if .VisibleFlags}} [options]{{end}}{{if .VisibleCommands}} [command [command options]]{{end}}{{if .ArgsUsage}} {{.ArgsUsage}}{{else}}{{if .Arguments}} {{range .Arguments}}[{{.Name}}] {{end}}{{end}}{{end}}{{end}}{{if .Arguments}}

Arguments:
{{- range $i, $e := .Arguments}}
	 {{wrap (concat "   " $e.Name $e.UsageText) 6}}{{end}}{{end}}{{if .Category}}
	
Category:
   {{.Category}}{{end}}{{if .Description}}

Description:
   {{template "descriptionTemplate" .}}{{end}}{{if .VisibleFlagCategories}}

Options:{{template "visibleFlagCategoryTemplate" .}}{{else if .VisibleFlags}}

Options:{{template "visibleFlagTemplate" .}}{{end}}{{if .VisiblePersistentFlags}}

Global options:{{template "visiblePersistentFlagTemplate" .}}{{end}}
`

	cli.HelpPrinterCustom = func(w io.Writer, templ string, data any, customFunc map[string]any) {
		if customFunc == nil {
			customFunc = map[string]any{}
		}

		customFunc["concat"] = func(sep string, args ...string) string {
			return strings.Join(args, sep)
		}
		cli.DefaultPrintHelpCustom(w, templ, data, customFunc)
	}

	cmd := &cli.Command{
		Name:  "skiff",
		Usage: "Share and reuse code in an LLM-friendly way.",
		Commands: []*cli.Command{
			{
				Name:  "build",
				Usage: "Build packages for a registry.",
				Flags: []cli.Flag{
					BuildFlagOutputDirectory,
				},
				Arguments: []cli.Argument{
					BuildArgRegistryPath,
				},
				Action: func(ctx context.Context, command *cli.Command) error {
					registryPath := command.StringArg(BuildArgRegistryPath.Name)
					if registryPath == "" {
						return errors.New("registry path is required")
					}
					bc := NewBuildAction()

					return bc.Act(ctx, &BuildArgs{
						OutputDirectory: command.String(BuildFlagOutputDirectory.Name),
						RegistryPath:    registryPath,
					})
				},
			},
		},
	}

	return &RootCommand{
		ProjectRoot: projectRoot,
		CLI:         cmd,
	}
}

type RootCommand struct {
	ProjectRoot string
	CLI         *cli.Command
}

func (r *RootCommand) Run(ctx context.Context, args []string) error {
	addCmd, err := newAddCmd(ctx, args)
	if err != nil {
		return err
	}

	r.CLI.Commands = append(r.CLI.Commands, addCmd)

	return r.CLI.Run(ctx, args)
}

func newAddCmd(ctx context.Context, args []string) (*cli.Command, error) {
	cmdArgs := filterFlagsFromArgs(args)
	isAddCmd := false
	if len(cmdArgs) > 2 {
		isAddCmd = cmdArgs[1] == "add"
	}

	addCmd := &cli.Command{
		Name:  "add",
		Usage: "Add code to your project.",
		Flags: []cli.Flag{
			AddFlagNonInteractive,
			AddFlagCreateAll,
		},
		Arguments: []cli.Argument{
			AddArgPackages,
		},
	}

	if isAddCmd {
		// Did the user specify an arg.
		if len(cmdArgs) >= 3 {
			pkgs, err := LoadPackages(ctx, cmdArgs[2:])
			if err != nil {
				return nil, err
			}

			flags, err := FlagsFromPackages(argsHaveFlag(cmdArgs, AddFlagNonInteractive), pkgs)
			if err != nil {
				return nil, err
			}

			act := NewAddAction(flags, pkgs)
			flatFlags := make([]cli.Flag, 0, len(act.PackageFlags))
			for _, pkgFlags := range flags {
				for i := range pkgFlags {
					flatFlags = append(flatFlags, pkgFlags[i].Flag)
				}
			}
			addCmd.Flags = append(addCmd.Flags, flatFlags...)
			addCmd.Action = func(ctx context.Context, command *cli.Command) error {
				root := command.String(AddFlagRoot.Name)
				if root == "" {
					root, err = os.Getwd()
					if err != nil {
						return err
					}
				}
				return act.Act(ctx, &AddArgs{
					ProjectRoot: filesystem.New(root),
				})
			}
		} else {
			addCmd.Action = func(ctx context.Context, command *cli.Command) error {
				return nil
			}
		}
	}
	return addCmd, nil
}

func filterFlagsFromArgs(s []string) []string {
	return collection.Filter(s, func(e string) bool {
		return !strings.HasPrefix(e, "-")
	})
}

func argsHaveFlag(args []string, fl cli.Flag) bool {
	names := fl.Names()
	return slices.ContainsFunc(args, func(s string) bool {
		return slices.Contains(names, s)
	})
}
