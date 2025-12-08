package gocmd

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"

	"github.com/skiff-sh/skiff/pkg/except"
	"github.com/skiff-sh/skiff/pkg/execcmd"
)

type GoCLI interface {
	Version(ctx context.Context) (*Version, error)
	Build(ctx context.Context, args BuildArgs) error
}

type BuildArgs struct {
	BuildMode  BuildMode
	OutputPath string
	Packages   []string
	GoOS       OS
	GoArch     Arch
}

type Arch string

const (
	ArchWASM Arch = "wasm"
)

type OS string

const (
	OSWASIP1 OS = "wasip1"
)

type BuildMode string

const (
	BuildModeCShared BuildMode = "c-shared"
)

type Version struct {
	Major int
	Minor int
	Patch int
}

var goVersionRegex = regexp.MustCompile(".+go([0-9]+)\\.([0-9]+)\\.([0-9]+)?.+")

func New() (GoCLI, error) {
	out := &goCLI{}

	cmd := exec.Command("go")
	if cmd.Err != nil {
		return nil, cmd.Err
	}

	return out, nil
}

type goCLI struct {
}

func (g *goCLI) Version(ctx context.Context) (*Version, error) {
	cmd, err := execcmd.NewCmd(ctx, "go", "version")
	if err != nil {
		return nil, err
	}
	defer cmd.Close()

	err = execcmd.Run(cmd)
	if err != nil {
		return nil, err
	}
	stdout := cmd.Buffers.Stdout.String()

	matches := goVersionRegex.FindAllStringSubmatch(stdout, -1)
	if len(matches) != 1 || len(matches[0]) != 4 {
		return nil, fmt.Errorf("%w: version: %s", except.ErrInvalid, stdout)
	}
	semver := matches[0][1:]

	major, err := strconv.Atoi(semver[0])
	if err != nil {
		return nil, fmt.Errorf("%w: invalid major version: %s", except.ErrInvalid, stdout)
	}

	minor, err := strconv.Atoi(semver[1])
	if err != nil {
		return nil, fmt.Errorf("%w: invalid minor version: %s", except.ErrInvalid, stdout)
	}

	patch, err := strconv.Atoi(semver[2])
	if err != nil {
		return nil, fmt.Errorf("%w: invalid patch version: %s", except.ErrInvalid, stdout)
	}

	return &Version{
		Major: major,
		Minor: minor,
		Patch: patch,
	}, err
}

func (g *goCLI) Build(ctx context.Context, build BuildArgs) error {
	args := []string{"build"}
	if build.BuildMode != "" {
		args = append(args, "-buildmode="+string(build.BuildMode))
	}

	if build.OutputPath != "" {
		args = append(args, "-o", build.OutputPath)
	}

	if len(build.Packages) > 0 {
		args = append(args, build.Packages...)
	}

	cmd, err := execcmd.NewCmd(ctx, "go", args...)
	if err != nil {
		return err
	}

	if build.GoOS != "" {
		cmd.Cmd.Env = append(cmd.Cmd.Env, "GOOS="+string(build.GoOS))
	}

	if build.GoArch != "" {
		cmd.Cmd.Env = append(cmd.Cmd.Env, "GOARCH="+string(build.GoArch))
	}

	return execcmd.Run(cmd)
}
