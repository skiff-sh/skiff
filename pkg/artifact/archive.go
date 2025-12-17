package artifact

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/skiff-sh/skiff/pkg/bufferpool"
	"github.com/skiff-sh/skiff/pkg/filesystem"
	"github.com/skiff-sh/skiff/pkg/fileutil"
)

// extractArchive handles extraction of both .zip (Windows) and .tar.gz (Unix) files.
func extractArchive(ctx context.Context, from io.Reader, to filesystem.Filesystem, ext string) error {
	v, ok := Archivers[ext]
	if !ok {
		return fmt.Errorf("unsupported archive format: %s", ext)
	}

	return v.Extract(ctx, from, to)
}

var Archivers = map[string]Archiver{
	".zip":    new(ZipArchiver),
	".tar.gz": new(TarGZArchiver),
}

type Archiver interface {
	Extract(ctx context.Context, from io.Reader, to filesystem.Filesystem) error
}

var _ Archiver = (*TarGZArchiver)(nil)

type TarGZArchiver struct {
}

func (t *TarGZArchiver) Extract(ctx context.Context, from io.Reader, to filesystem.Filesystem) error {
	gz, err := gzip.NewReader(from)
	if err != nil {
		return fmt.Errorf("creating gzip reader: %w", err)
	}
	defer func() {
		_ = gz.Close()
	}()

	tr := tar.NewReader(gz)

	for {
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		default:
		}

		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			// End of archive.
			break
		}
		if err != nil {
			return fmt.Errorf("reading tar header: %w", err)
		}

		if isExcludedHeaderName(hdr.Name) {
			// Some malformed archives might have empty names; skip them.
			continue
		}
		name := filepath.Clean(hdr.Name)

		// Clean the path and ensure it is within destDir to
		// prevent path traversal attacks.
		destPath := name

		// Tar headers can contain absolute paths; make them relative.
		if filepath.IsAbs(destPath) {
			destPath = destPath[1:]
		}

		// Tar headers sometimes start with ./.
		destPath = strings.TrimPrefix(destPath, "./")
		if destPath == "" { // e.g. header for root directory
			continue
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := mkdirWithMode(destPath, to, hdr.FileInfo().Mode()); err != nil {
				return fmt.Errorf("creating directory %q: %w", destPath, err)
			}
			_ = preserveTimes(to, destPath, hdr.AccessTime, hdr.ModTime)

		case tar.TypeReg:
			if err := writeFileFromTar(tr, destPath, to, hdr.FileInfo().Mode(), hdr.ModTime); err != nil {
				return fmt.Errorf("writing file %q: %w", destPath, err)
			}

		case tar.TypeSymlink:
			// Ensure parent dir exists.
			if err := to.MkdirAll(filepath.Dir(destPath), fileutil.DefaultDirMode); err != nil {
				return fmt.Errorf("creating parent directory for symlink %q: %w", destPath, err)
			}
			// Remove existing path before creating symlink.
			_ = to.Remove(destPath)
			if err := to.Symlink(hdr.Linkname, destPath); err != nil {
				return fmt.Errorf("creating symlink %q -> %q: %w", destPath, hdr.Linkname, err)
			}

		case tar.TypeLink:
			// Hard link. We'll best-effort create it if the target exists inside
			// the extraction root. If it doesn't, skip.
			targetPath, err := to.Abs(hdr.Linkname)
			if err != nil {
				return fmt.Errorf(
					"creating hardlink %q -> %q: %w",
					destPath, targetPath, err,
				)
			}
			if _, err := to.Stat(targetPath); err == nil {
				// Remove if something already exists at destPath.
				_ = to.Remove(destPath)
				if err := to.Link(targetPath, destPath); err != nil {
					// Non-fatal: on some systems or FSs this may fail.
					// Fall back to copying the file contents instead.
					if copyErr := copyFile(to, targetPath, destPath); copyErr != nil {
						return fmt.Errorf(
							"creating hardlink %q -> %q (and copy fallback failed): %w / %w",
							destPath, targetPath, err, copyErr,
						)
					}
				}
			}

		default:
			// Skip other types (sockets, devices, etc.).
			continue
		}
	}

	return nil
}

var _ Archiver = (*ZipArchiver)(nil)

type ZipArchiver struct {
}

func (z *ZipArchiver) Extract(ctx context.Context, from io.Reader, to filesystem.Filesystem) error {
	buff := bufferpool.GetBytesBuffer()
	defer bufferpool.PutBytesBuffers(buff)

	_, err := io.Copy(buff, from)
	if err != nil {
		return err
	}

	reader := bufferpool.GetBytesReader(buff.Bytes())
	defer bufferpool.PutBytesReader(reader)

	zr, err := zip.NewReader(reader, reader.Size())
	if err != nil {
		return fmt.Errorf("opening zip file: %w", err)
	}

	for _, f := range zr.File {
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		default:
		}

		if isExcludedHeaderName(f.Name) {
			continue
		}

		destPath := filepath.Clean(f.Name)
		if filepath.IsAbs(destPath) {
			destPath = destPath[1:]
		}
		destPath = strings.TrimPrefix(destPath, "./")
		if destPath == "" {
			continue
		}

		mode := f.Mode()

		// Directory entry.
		if f.FileInfo().IsDir() {
			if err := mkdirWithMode(destPath, to, mode); err != nil {
				return fmt.Errorf("creating directory %q: %w", destPath, err)
			}
			_ = preserveZipTimes(to, destPath, f.Modified)
			continue
		}

		// Symlink (zip has no dedicated type, but mode can indicate it).
		if mode&os.ModeSymlink != 0 {
			if err := extractZipSymlink(to, f, destPath); err != nil {
				return fmt.Errorf("creating symlink for %q: %w", destPath, err)
			}
			continue
		}

		// Regular file (best-effort detection).
		if err := extractZipFile(to, f, destPath); err != nil {
			return fmt.Errorf("writing file %q: %w", destPath, err)
		}

		_ = to.Chmod(destPath, mode.Perm())
		_ = preserveZipTimes(to, destPath, f.Modified)
	}

	return nil
}

func mkdirWithMode(path string, fsys filesystem.Filesystem, mode os.FileMode) error {
	if mode == 0 {
		mode = fileutil.DefaultDirMode
	}

	if err := fsys.MkdirAll(path, mode.Perm()); err != nil {
		return err
	}
	// Best-effort chmod after creation, to catch umask differences.
	_ = fsys.Chmod(path, mode.Perm())
	return nil
}

func writeFileFromTar(
	r io.Reader,
	destPath string,
	fsys filesystem.Filesystem,
	mode os.FileMode,
	modTime time.Time,
) error {
	// Ensure parent dir exists.
	if err := fsys.MkdirAll(filepath.Dir(destPath), fileutil.DefaultDirMode); err != nil {
		return err
	}

	// Create/truncate file.
	f, err := fsys.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode.Perm())
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()

	if _, err := io.Copy(f, r); err != nil {
		return err
	}

	_ = f.Sync()

	// Preserve mode & timestamps best-effort.
	_ = fsys.Chmod(destPath, mode.Perm())
	_ = fsys.Chtimes(destPath, modTime, modTime)

	return nil
}

func isExcludedHeaderName(headerName string) bool {
	return headerName == "" || strings.HasPrefix(filepath.Base(headerName), "._") ||
		strings.HasPrefix(headerName, ".DS_Store") ||
		strings.Contains(headerName, "/.DS_Store")
}

func copyFile(fsys filesystem.Filesystem, src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		_ = in.Close()
	}()

	if err := fsys.MkdirAll(filepath.Dir(dst), fileutil.DefaultDirMode); err != nil {
		return err
	}

	out, err := fsys.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		_ = out.Close()
	}()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

func extractZipFile(fsys filesystem.Filesystem, f *zip.File, destPath string) error {
	// Ensure parent dir exists.
	if err := fsys.MkdirAll(filepath.Dir(destPath), fileutil.DefaultDirMode); err != nil {
		return err
	}

	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer func() {
		_ = rc.Close()
	}()

	// Create/truncate file; fileutil.DefaultFileMode default; we'll chmod later.
	out, err := fsys.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, fileutil.DefaultFileMode)
	if err != nil {
		return err
	}
	defer func() {
		_ = out.Close()
	}()

	if _, err := io.Copy(out, rc); err != nil {
		return err
	}

	return out.Sync()
}

func extractZipSymlink(fsys filesystem.Filesystem, f *zip.File, destPath string) error {
	// Ensure parent dir exists.
	if err := fsys.MkdirAll(filepath.Dir(destPath), fileutil.DefaultDirMode); err != nil {
		return err
	}

	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer func() {
		_ = rc.Close()
	}()

	linkTargetBytes, err := io.ReadAll(rc)
	if err != nil {
		return err
	}
	linkTarget := string(linkTargetBytes)

	// Remove any existing path where we want to place the symlink.
	_ = fsys.Remove(destPath)
	return fsys.Symlink(linkTarget, destPath)
}

func preserveTimes(fsys filesystem.Filesystem, path string, atime, mtime time.Time) error {
	// Some tar creators set AccessTime to zero; fall back if needed.
	if atime.IsZero() {
		atime = mtime
	}
	if mtime.IsZero() {
		return nil
	}
	return fsys.Chtimes(path, atime, mtime)
}

func preserveZipTimes(fsys filesystem.Filesystem, path string, mtime time.Time) error {
	if mtime.IsZero() {
		return nil
	}
	return fsys.Chtimes(path, mtime, mtime)
}
