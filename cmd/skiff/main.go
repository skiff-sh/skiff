package main

import (
	"context"
	"os"

	"github.com/skiff-sh/skiff/cmd/cmdinit"
	"github.com/skiff-sh/skiff/pkg/interact"
)

func main() {
	cmd, err := cmdinit.NewCommand()
	if err != nil {
		interact.Error(err.Error())
		os.Exit(1)
	}

	ctx := context.Background()

	err = cmd.Run(ctx, os.Args)
	if err != nil {
		interact.Error(err.Error())
		os.Exit(1)
	}
}
