package main

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io"
	"io/fs"
	"path"

	"github.com/skiff-sh/api/go/skiff/plugin/v1alpha1"
	"github.com/skiff-sh/sdk-go/skiff"
	"github.com/skiff-sh/sdk-go/skiff/issue"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var _ skiff.Plugin = (*Plugin)(nil)

type Plugin struct{}

var caser = cases.Title(language.English)

func (p *Plugin) WriteFile(ctx *skiff.Context, _ *v1alpha1.WriteFileRequest) (*v1alpha1.WriteFileResponse, error) {
	b, err := fs.ReadFile(ctx.CWD.FS, path.Join("controller", "controller.go"))
	if err != nil {
		return nil, issue.Errorf("Must be run from the root of the repository: %s", err.Error())
	}

	if ctx.Data["name"] == nil || ctx.Data["name"].String == nil {
		return nil, issue.Error("Missing name")
	}

	name := *ctx.Data["name"].String
	capName := caser.String(name[:1]) + name[1:]
	buff := bytes.NewBuffer(nil)
	err = addController(capName+"Controller", b, buff)
	if err != nil {
		return nil, issue.Errorf("Failed to add new controller: %s", err.Error())
	}

	return &v1alpha1.WriteFileResponse{Contents: buff.Bytes()}, nil
}

// Injects a new(<controller name>) entry into controller.go
func addController(name string, content []byte, into io.Writer) error {
	fset := token.NewFileSet()
	fi, err := parser.ParseFile(fset, "controller.go", content, parser.ParseComments)
	if err != nil {
		return err
	}

	ast.Inspect(fi, func(n ast.Node) bool {
		gen, ok := n.(*ast.GenDecl)
		if !ok || gen.Tok != token.VAR {
			return true
		}

		for _, spec := range gen.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}

			if len(vs.Names) == 0 || vs.Names[0].Name != "Controllers" {
				continue
			}

			// Expect: Controllers = []Controller{ ... }
			cl, ok := vs.Values[0].(*ast.CompositeLit)
			if !ok {
				return true
			}

			// Append: new(<name>)
			newCall := &ast.CallExpr{
				Fun: &ast.Ident{Name: "new"},
				Args: []ast.Expr{
					&ast.Ident{Name: name},
				},
			}

			cl.Elts = append(cl.Elts, newCall)

			return true
		}
		return true
	})

	return printer.Fprint(into, fset, fi)
}

func init() {
	skiff.Register(new(Plugin))
}

func main() {
}
