package skiff

import (
	"context"
	"io/fs"

	"github.com/skiff-sh/skiff/api/go/skiff/plugin/v1alpha1"
)

type Context struct {
	Ctx  context.Context
	Root fs.FS
}

type WriteFileFunc func(ctx *Context, req *v1alpha1.WriteFileRequest) (*v1alpha1.WriteFileResponse, error)
