package mcpserver

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/skiff-sh/api/go/skiff/cmd/v1alpha1"
	v1alpha2 "github.com/skiff-sh/api/go/skiff/registry/v1alpha1"
	"github.com/skiff-sh/config/contexts"
	"google.golang.org/protobuf/proto"

	"github.com/skiff-sh/skiff/pkg/collection"

	"github.com/skiff-sh/skiff/pkg/schema"

	"github.com/skiff-sh/skiff/pkg/registry"

	"github.com/skiff-sh/skiff/pkg/embedded"

	"github.com/skiff-sh/skiff/pkg/protoencode"

	"github.com/skiff-sh/skiff/pkg/engine"

	"github.com/skiff-sh/skiff/pkg/vars"
)

type Server struct {
	MCP *mcp.Server
}

func NewServer(eng engine.Engine) (*Server, error) {
	srv := mcp.NewServer(&mcp.Implementation{
		Name:    vars.AppName,
		Title:   "Create and edit files from template packages.",
		Version: vars.Version,
	}, &mcp.ServerOptions{
		Logger:   slog.Default(),
		HasTools: true,
	})

	addPkg, err := embedded.LoadExchange(embedded.ExchangeNameAddPackage)
	if err != nil {
		return nil, fmt.Errorf("failed to load add package json schemas: %w", err)
	}

	listPkgs, err := embedded.LoadExchange(embedded.ExchangeNameListPackages)
	if err != nil {
		return nil, fmt.Errorf("failed to load list packages json schemas: %w", err)
	}

	viewPkg, err := embedded.LoadExchange(embedded.ExchangeNameViewPackages)
	if err != nil {
		return nil, fmt.Errorf("failed to load list packages json schemas: %w", err)
	}

	srv.AddTool(&mcp.Tool{
		Name:         "add_package",
		Description:  "Add/edit packages of files in the user's project. You **must** call 'list_packages' for each package you want to add to retrieve the data schemas under the 'schema' field.",
		InputSchema:  addPkg.RequestSchema,
		OutputSchema: addPkg.ResponseSchema,
	}, handleAddPackage(eng))

	srv.AddTool(&mcp.Tool{
		Name:         "list_packages",
		Description:  "Lists all available packages for a given registry. If no registry is specified, the user's current project packages are listed (if they exist). If none are found, an error is returned.",
		InputSchema:  listPkgs.RequestSchema,
		OutputSchema: listPkgs.ResponseSchema,
	}, handleListPackage(eng))

	srv.AddTool(&mcp.Tool{
		Name:         "view_package",
		Description:  "Get a detailed view of the package and all of its files.",
		InputSchema:  viewPkg.RequestSchema,
		OutputSchema: viewPkg.ResponseSchema,
	}, handleViewPackages(eng))

	return &Server{MCP: srv}, nil
}

func handleViewPackages(eng engine.Engine) mcp.ToolHandler {
	return func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logger := contexts.GetLogger(ctx)
		req := new(v1alpha1.ViewPackagesRequest)
		err := protoencode.Unmarshal(request.Params.Arguments, req)
		if err != nil {
			return newErrResult(err), nil
		}

		resp := &v1alpha1.ViewPackagesResponse{Packages: make([]*v1alpha2.Package, 0, len(req.GetPackages()))}
		for _, v := range req.GetPackages() {
			pkg, err := eng.ViewPackage(ctx, v)
			if err != nil {
				logger.Error("Invalid package.", "name", v, "err", err.Error())
				continue
			}

			resp.Packages = append(resp.Packages, pkg)
		}

		if len(resp.GetPackages()) == 0 {
			return newResult(
				fmt.Sprintf(
					"No packages found for: %s.\n\nMake sure the paths are correct and use the absolute path if referencing local files.",
					strings.Join(req.GetPackages(), ", "),
				),
			), nil
		}

		return handleResp(resp, nil)
	}
}

func handleAddPackage(eng engine.Engine) mcp.ToolHandler {
	return func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		req := new(v1alpha1.AddPackageRequest)
		err := protoencode.Unmarshal(request.Params.Arguments, req)
		if err != nil {
			return newErrResult(err), nil
		}

		pkg, err := registry.LoadPackage(ctx, req.GetPackage())
		if err != nil {
			return newErrResult(fmt.Errorf("%s is not a valid package: %w", req.GetPackage(), err)), nil
		}

		vals, err := schema.NewProtoMapValues(req.GetData())
		if err != nil {
			return newErrResult(err), nil
		}

		res, err := eng.AddPackage(ctx, pkg, schema.NewPackageSource(vals...))
		if err != nil {
			return newErrResult(err), nil
		}

		return handleResp(&v1alpha1.AddPackageResponse{
			UnifiedDiffs: collection.Map(res.Diffs, func(e *engine.Diff) string {
				return string(e.UnifiedDiff)
			}),
		}, nil)
	}
}

func handleListPackage(eng engine.Engine) mcp.ToolHandler {
	return func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logger := contexts.GetLogger(ctx)
		req := new(v1alpha1.ListPackagesRequest)
		err := protoencode.Unmarshal(request.Params.Arguments, req)
		if err != nil {
			return newErrResult(fmt.Errorf("invalid input %w", err)), nil
		}

		content := make([]*v1alpha1.ListPackagesResponse_PackagePreview, 0, len(req.GetRegistries()))
		for _, v := range req.GetRegistries() {
			pkgs, err := eng.ListPackages(ctx, v)
			if err != nil {
				logger.Error("Failed to get package for registry.", "name", v, "err", err.Error())
				continue
			}

			content = append(content, pkgs...)
		}

		if len(content) == 0 {
			return newResult(
				fmt.Sprintf(
					"No packages found in: %s.\n\nMake sure you list the proper URLs and use absolute paths for file paths.",
					strings.Join(req.GetRegistries(), ", "),
				),
			), nil
		}

		return handleResp(&v1alpha1.ListPackagesResponse{Packages: content}, nil)
	}
}

func handleResp(resp proto.Message, err error) (*mcp.CallToolResult, error) {
	if err != nil {
		return newErrResult(err), nil
	}

	b, err := protoencode.Marshal(resp)
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(b)}}}, nil
}

func newResult(txt string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: txt}},
	}
}

func newErrResult(err error) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{newErrContent(err)},
	}
}

func newErrContent(err error) *mcp.TextContent {
	return &mcp.TextContent{Text: err.Error()}
}
