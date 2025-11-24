package commands

import "github.com/urfave/cli/v3"

var BuildOutputDirectoryFlag = &cli.StringFlag{
	Name:  "output",
	Usage: "Destination directory for registry json files.",
	Value: "./public/r",
}

var BuildRegistryPathArg = &cli.StringArg{
	Name:      "registry-path",
	UsageText: "registry file path",
	Value:     "./.skiff/registry.json",
}
