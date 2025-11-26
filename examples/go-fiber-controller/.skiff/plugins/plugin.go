package main

import (
	"github.com/skiff-sh/api/go/skiff/plugin/v1alpha1"
	"github.com/skiff-sh/sdk-go/skiff"
)

var _ skiff.Plugin = (*Plugin)(nil)

type Plugin struct {
}

func (p *Plugin) WriteFile(ctx *skiff.Context, req *v1alpha1.WriteFileRequest) (*v1alpha1.WriteFileResponse, error) {
	return &v1alpha1.WriteFileResponse{Contents: []byte("hi")}, nil
}

func init() {
	skiff.Register(new(Plugin))
}

func main() {

}
