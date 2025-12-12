package main

import (
	"io/fs"

	"github.com/skiff-sh/api/go/skiff/plugin/v1alpha1"
	"github.com/skiff-sh/sdk-go/skiff"
)

var _ skiff.Plugin = (*FileAccessPlugin)(nil)

type FileAccessPlugin struct{}

func (p *FileAccessPlugin) WriteFile(c *skiff.Context, _ *v1alpha1.WriteFileRequest) (*v1alpha1.WriteFileResponse, error) {
	b, err := fs.ReadFile(c.CWD.FS, "derp.txt")
	if err != nil {
		return nil, err
	}
	return &v1alpha1.WriteFileResponse{Contents: b}, nil
}

func init() {
	skiff.Register(new(FileAccessPlugin))
}

func main() {
}
