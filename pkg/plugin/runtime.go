package plugin

import (
	"context"

	"github.com/skiff-sh/skiff/api/go/skiff/plugin/v1alpha1"
)

type Interpreter interface {
	WriteFile(ctx *Context, req *v1alpha1.WriteFileRequest) (*v1alpha1.WriteFileResponse, error)
}

type Context struct {
	Ctx context.Context
}
