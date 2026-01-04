package commands

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/urfave/cli/v3"

	"github.com/skiff-sh/skiff/pkg/engine"
	"github.com/skiff-sh/skiff/pkg/filesystem"
	"github.com/skiff-sh/skiff/pkg/mcpserver"
	"github.com/skiff-sh/skiff/pkg/plugin"
	"github.com/skiff-sh/skiff/pkg/system"
	"github.com/skiff-sh/skiff/pkg/tmpl"
)

var MCPFlagRoot = &cli.StringFlag{
	Name:    "root",
	Usage:   "The root of your project. Defaults to the cwd.",
	Aliases: []string{"r"},
}

type MCPActionArgs struct {
	ProjectRoot filesystem.Filesystem
}

type MCPAction struct {
	Compiler plugin.Compiler
	System   system.System
}

func NewMCPAction(comp plugin.Compiler, sys system.System) *MCPAction {
	return &MCPAction{
		Compiler: comp,
		System:   sys,
	}
}

func (m *MCPAction) Act(ctx context.Context, args *MCPActionArgs) error {
	eng := engine.New(tmpl.NewGoFactory(), m.Compiler, m.System, args.ProjectRoot)
	srv, err := mcpserver.NewServer(eng)
	if err != nil {
		return err
	}

	return srv.MCP.Run(ctx, &mcp.StdioTransport{})
}
