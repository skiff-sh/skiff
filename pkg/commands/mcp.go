package commands

import (
	"context"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/skiff-sh/config/contexts"
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

var MCPFlagAddr = &cli.StringFlag{
	Name:    "addr",
	Usage:   "Set an address for the MCP server to listen on. This changes the server to run in HTTP stream mode instead of stdio.",
	Aliases: []string{"a"},
}

type MCPActionArgs struct {
	ProjectRoot filesystem.Filesystem
	Address     string
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
	logger := contexts.GetLogger(ctx)
	eng := engine.New(tmpl.NewGoFactory(), m.Compiler, m.System, args.ProjectRoot)
	srv, err := mcpserver.NewServer(eng)
	if err != nil {
		return err
	}

	if args.Address != "" {
		handler := mcp.NewStreamableHTTPHandler(func(request *http.Request) *mcp.Server {
			return srv.MCP
		}, nil)

		ctx, cancel := context.WithCancelCause(ctx)
		go func() {
			logger.Info("Running MCP HTTP server.", "address", args.Address)
			serv := &http.Server{Addr: args.Address, Handler: handler}
			defer func() {
				_ = serv.Shutdown(context.Background())
			}()
			err := serv.ListenAndServe()
			if err != nil {
				logger.Error("Failed to start MCP HTTP server", "address", args.Address, "err", err.Error())
			}
			cancel(err)
		}()

		<-ctx.Done()
		return context.Cause(ctx)
	}

	return srv.MCP.Run(ctx, &mcp.StdioTransport{})
}
