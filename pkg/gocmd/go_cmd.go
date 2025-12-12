package gocmd

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/skiff-sh/skiff/pkg/except"
	"github.com/skiff-sh/skiff/pkg/execcmd"
	"github.com/skiff-sh/skiff/pkg/fileutil"
)

type CLI interface {
	Path() string
	Version(ctx context.Context) (*Version, error)
	Build(ctx context.Context, args BuildArgs) (*execcmd.Buffers, error)
}

type BuildArgs struct {
	BuildMode  BuildMode
	OutputPath string
	Packages   []string
	GoOS       OS
	GoArch     Arch
	Env        []string
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

func (v *Version) String() string {
	return strings.Join([]string{strconv.Itoa(v.Major), strconv.Itoa(v.Minor), strconv.Itoa(v.Patch)}, ".")
}

var goVersionRegex = regexp.MustCompile(`.+go([0-9]+)\.([0-9]+)\.([0-9]+)?.+`)

func New() (CLI, error) {
	out := &goCLI{}

	cmd := exec.CommandContext(context.Background(), "go")
	if cmd.Err != nil {
		return nil, cmd.Err
	}

	out.path = cmd.Path

	return out, nil
}

type goCLI struct {
	path string
}

func (g *goCLI) Path() string {
	return g.path
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

func (g *goCLI) Build(ctx context.Context, build BuildArgs) (*execcmd.Buffers, error) {
	if len(build.Packages) == 0 {
		return nil, errors.New("at least one go file required to build")
	}

	args := []string{"build"}
	if build.BuildMode != "" {
		args = append(args, "-buildmode="+string(build.BuildMode))
	}

	if build.OutputPath != "" {
		args = append(args, "-o", build.OutputPath)
	}

	sib, err := fileutil.FindSibling(build.Packages[0], "go.mod")
	if err != nil {
		return nil, fmt.Errorf("go.mod file required for %s", build.Packages[0])
	}
	args = append(args, build.Packages...)

	cmd, err := execcmd.NewCmd(ctx, "go", args...)
	if err != nil {
		return nil, err
	}

	cmd.Cmd.Dir = filepath.Dir(sib)
	cmd.Cmd.Env = build.Env

	if build.GoOS != "" {
		cmd.Cmd.Env = append(cmd.Cmd.Env, "GOOS="+string(build.GoOS))
	}

	if build.GoArch != "" {
		cmd.Cmd.Env = append(cmd.Cmd.Env, "GOARCH="+string(build.GoArch))
	}

	err = execcmd.Run(cmd)
	return cmd.Buffers, err
}
