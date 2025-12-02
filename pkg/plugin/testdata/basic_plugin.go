package main

import (
	"encoding/json"

	"github.com/skiff-sh/sdk-go/skiff"
)

var _ skiff.Plugin = (*Plugin)(nil)

type Plugin struct{}

type req struct {
	Path   string
	Target string
}

func (p *Plugin) WriteFile(ctx *skiff.Context) error {
	req := new(req)

	_, _ = json.Marshal(req)
	return nil
}

func init() {
	skiff.Register(new(Plugin))
}

func main() {
}
