package plugin

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/skiff-sh/config/contexts"

	"github.com/skiff-sh/skiff/pkg/fileutil"
	"github.com/skiff-sh/skiff/pkg/gocmd"
	"github.com/skiff-sh/skiff/pkg/settings"
)

const (
	managedGoModCacheDirName = "modcache"
	minMinorGoVersion        = 24
	targetGoVersion          = "1.25.5"
	goModCacheEnvVarKey      = "GOMODCACHE"
)

type Builder interface {
	Build(ctx context.Context, from string, to io.Writer) error
}

func InstallGoCLI(ctx context.Context) (gocmd.CLI, error) {
	dir, err := settings.BuildDir()
	if err != nil {
		return nil, err
	}

	cli, err := gocmd.Download(ctx, targetGoVersion, dir)
	if err != nil {
		return nil, err
	}

	return cli, nil
}

type InstallHooks struct {
	OnDownload         func()
	OnDownloadComplete func()
}

func CreateOrInstallGoBuilder(ctx context.Context, hooks *InstallHooks) (Builder, error) {
	logger := contexts.GetLogger(ctx)
	var goModCache string
	cli, err := gocmd.New("")
	if err != nil {
		logger.DebugContext(ctx, "Go not detected on path. Checking installation directory.")
		cli, err = getOrInstallGoCLI(ctx, hooks)
		if err != nil {
			return nil, err
		}
		buildDir, _ := settings.BuildDir()
		goModCache = filepath.Join(buildDir, managedGoModCacheDirName)
		if !fileutil.Exists(goModCache) {
			_ = os.Mkdir(goModCache, fileutil.DefaultDirMode)
		}
	}

	ver, err := cli.Version(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to get Go version", "err", err)
		return nil, err
	}

	if ver.Minor < minMinorGoVersion {
		return nil, fmt.Errorf(
			"go CLI (at %s) is version %s. please upgrade to at least 1.%d to properly build plugins",
			cli.Path(),
			ver.String(),
			minMinorGoVersion,
		)
	}
	return &goBuilder{CLI: cli, GoModCacheDir: goModCache}, nil
}

func getOrInstallGoCLI(ctx context.Context, hooks *InstallHooks) (gocmd.CLI, error) {
	logger := contexts.GetLogger(ctx)
	local, err := goInstallPath()
	if err != nil {
		logger.ErrorContext(ctx, "Failed to get go install path.", "err", err.Error())
		return nil, err
	}

	cli, err := gocmd.New(local)
	if err != nil {
		logger.DebugContext(ctx, "CLI doesn't exist locally. Downloading.", "err", err.Error())
		if hooks.OnDownload != nil {
			hooks.OnDownload()
		}
		cli, err = InstallGoCLI(ctx)
		if err != nil {
			return nil, err
		}
		if hooks.OnDownloadComplete != nil {
			hooks.OnDownloadComplete()
		}
	}

	return cli, nil
}

func goInstallPath() (string, error) {
	dir, err := settings.BuildDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, "go", "bin", gocmd.GoFilename()), nil
}

type goBuilder struct {
	CLI gocmd.CLI
	// If set, overrides the current env var with the value
	GoModCacheDir string
}

func (g *goBuilder) Build(ctx context.Context, from string, to io.Writer) error {
	tmp, err := os.MkdirTemp(os.TempDir(), "*")
	if err != nil {
		return fmt.Errorf("failed to create temp build dir: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(tmp)
	}()

	args := gocmd.BuildArgs{
		BuildMode:  gocmd.BuildModeCShared,
		OutputPath: filepath.Join(tmp, "plugin.wasm"),
		Packages:   []string{from},
		GoOS:       gocmd.OSWASIP1,
		GoArch:     gocmd.ArchWASM,
		Env:        os.Environ(),
	}

	if g.GoModCacheDir != "" {
		args.Env = append(args.Env, goModCacheEnvVarKey+"="+g.GoModCacheDir)
	}

	buffers, err := g.CLI.Build(ctx, args)
	if err != nil {
		var stderr string
		if buffers != nil {
			stderr = buffers.Stderr.String()
		}
		return fmt.Errorf("failed to build plugin %s: %w: %s", from, err, stderr)
	}

	f, err := os.Open(args.OutputPath)
	if err != nil {
		return fmt.Errorf("failed to open plugin build output: %w", err)
	}
	defer func() {
		_ = f.Close()
	}()

	_, err = io.Copy(to, f)
	return err
}
