package commands

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/skiff-sh/skiff/pkg/engine"
	"github.com/skiff-sh/skiff/pkg/plugin"
	"github.com/skiff-sh/skiff/pkg/tmpl"

	"github.com/skiff-sh/skiff/pkg/vars"

	"github.com/skiff-sh/skiff/pkg/system"

	"github.com/skiff-sh/skiff/pkg/filesystem"
)

type RootCommand struct {
	ProjectRoot string
	CLI         *cli.Command
	System      system.System
}

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

	sys := system.New()
	cmd := &cli.Command{
		Name:  vars.AppName,
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
			{
				Name:  "mcp",
				Usage: "MCP server",
				Flags: []cli.Flag{
					MCPFlagRoot,
					MCPFlagAddr,
				},
				Action: func(ctx context.Context, command *cli.Command) error {
					projectPath := command.String(MCPFlagRoot.Name)
					if projectPath == "" {
						projectPath = sys.CWD()
					}

					compiler, err := plugin.NewWazeroCompiler()
					if err != nil {
						return fmt.Errorf("failed to create WASM compiler: %w", err)
					}

					bc := NewMCPAction(compiler, sys)

					return bc.Act(ctx, &MCPActionArgs{
						ProjectRoot: filesystem.New(projectPath),
						Address:     command.String(MCPFlagAddr.Name),
					})
				},
			},
		},
	}

	return &RootCommand{
		ProjectRoot: projectRoot,
		CLI:         cmd,
		System:      sys,
	}
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
	addCmd := &cli.Command{
		Name:  "add",
		Usage: "Add code to your project.",
		Flags: []cli.Flag{
			AddFlagNonInteractive,
			AddFlagCreateAll,
			AddFlagRoot,
		},
		Arguments: []cli.Argument{
			AddArgPackages,
		},
		Action: func(_ context.Context, _ *cli.Command) error {
			return nil
		},
	}

	cmdArgs := filterFlagsFromArgs(args, addCmd.Flags)
	isAddCmd := false
	//nolint:mnd // not magic
	if len(cmdArgs) > 2 {
		isAddCmd = cmdArgs[1] == "add"
	}

	if !isAddCmd {
		return addCmd, nil
	}

	// Did the user specify an arg.
	//nolint:mnd // not magic
	if len(cmdArgs) < 3 {
		return addCmd, nil
	}

	pkgs, err := LoadPackages(ctx, cmdArgs[2:])
	if err != nil {
		return nil, err
	}

	flags, err := FlagsFromPackages(argsHaveFlag(args, AddFlagNonInteractive), pkgs)
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
			root, err = system.Getwd()
			if err != nil {
				return err
			}
		}

		compiler, err := plugin.NewWazeroCompiler()
		if err != nil {
			return fmt.Errorf("failed to create WASM compiler: %w", err)
		}

		fsys := filesystem.New(root)

		err = act.Act(ctx, &AddArgs{
			ProjectRoot: fsys,
			Engine:      engine.New(tmpl.NewGoFactory(), compiler, system.New(), fsys),
			CreateAll:   command.Bool(AddFlagCreateAll.Name),
		})
		if err != nil {
			if errors.Is(err, ErrSchema) {
				_ = cli.ShowSubcommandHelp(command)
			}
			return err
		}

		return nil
	}
	return addCmd, nil
}

func filterFlagsFromArgs(s []string, possibleFlags []cli.Flag) []string {
	possibleFlags = append(possibleFlags, cli.HelpFlag, cli.VersionFlag)
	skipNext := false
	out := make([]string, 0, len(s))
	for _, v := range s {
		if skipNext {
			skipNext = false
			continue
		}

		if strings.HasPrefix(v, "-") {
			flagName := strings.Trim(v, "-")
			idx := slices.IndexFunc(possibleFlags, func(c cli.Flag) bool {
				return slices.Contains(c.Names(), flagName)
			})
			if idx >= 0 {
				_, isBool := possibleFlags[idx].(*cli.BoolFlag)
				if !isBool {
					skipNext = true
				}
			}
			continue
		}

		out = append(out, v)
	}

	return out
}

func argsHaveFlag(args []string, fl cli.Flag) bool {
	names := fl.Names()
	return slices.ContainsFunc(args, func(s string) bool {
		return slices.Contains(names, strings.TrimLeft(s, "-"))
	})
}
