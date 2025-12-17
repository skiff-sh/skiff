package artifact

import (
	"context"
	"io"
	"net/http"
	"path"
	"runtime"
	"strings"

	"github.com/eddieowens/opts"

	"github.com/skiff-sh/skiff/pkg/filesystem"
)

type Artifact interface {
	URL(os, arch string) (string, error)
	Name() string
}

func WithOS(os string) opts.Opt[InstallOpts] {
	return func(i *InstallOpts) {
		i.OS = os
	}
}

func WithArch(arch string) opts.Opt[InstallOpts] {
	return func(i *InstallOpts) {
		i.Arch = arch
	}
}

type InstallOpts struct {
	OS        string
	Arch      string
	ExtractTo filesystem.Filesystem
}

func (i InstallOpts) DefaultOptions() InstallOpts {
	return InstallOpts{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}
}

type Destination func(ctx context.Context, artifactSrcPath string, r io.Reader) error

func WriterDestination(w io.Writer) Destination {
	return func(_ context.Context, _ string, r io.Reader) error {
		_, err := io.Copy(w, r)
		return err
	}
}

func UnarchiveDestination(to filesystem.Filesystem) Destination {
	return func(ctx context.Context, artifactSrcPath string, r io.Reader) error {
		var ext string
		if strings.HasSuffix(artifactSrcPath, ".tar.gz") {
			ext = ".tar.gz"
		} else {
			ext = path.Ext(artifactSrcPath)
		}
		return extractArchive(ctx, r, to, ext)
	}
}

type Installer interface {
	InstallHTTP(ctx context.Context, art Artifact, dest Destination, op ...opts.Opt[InstallOpts]) error
}

func NewInstaller(cl *http.Client) Installer {
	return &installer{Client: cl}
}

type installer struct {
	Client *http.Client
}

func (i *installer) InstallHTTP(
	ctx context.Context,
	art Artifact,
	dest Destination,
	op ...opts.Opt[InstallOpts],
) error {
	o := opts.DefaultApply(op...)

	u, err := art.URL(o.OS, o.Arch)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}

	resp, err := i.Client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	return dest(ctx, u, resp.Body)
}
