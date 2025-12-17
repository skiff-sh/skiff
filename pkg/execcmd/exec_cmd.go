package execcmd

import (
	"bytes"
	"context"
	"log/slog"
	"os/exec"
	"strings"

	"github.com/skiff-sh/skiff/pkg/bufferpool"
	"github.com/skiff-sh/skiff/pkg/system"
)

var (
	// DefaultRunner is the default Runner for the package.
	DefaultRunner Runner = RunnerFunc(func(cmd *Cmd) error {
		return cmd.Cmd.Run()
	})

	// DefaultPathLooker the default PathLooker for the package.
	DefaultPathLooker PathLooker = PathLookerFunc(exec.LookPath)
)

// Runner runs an exec.Cmd. Useful for tracking and testing command runs.
type Runner interface {
	Run(cmd *Cmd) error
}

// PathLooker an abstraction around os.LookPath.
type PathLooker interface {
	LookPath(fi string) (string, error)
}

type Cmd struct {
	Cmd     *exec.Cmd
	Buffers *Buffers
}

func NewCmd(ctx context.Context, name string, args ...string) (*Cmd, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if cmd.Err != nil {
		return nil, cmd.Err
	}

	var err error
	cmd.Dir, err = system.Getwd()
	if err != nil {
		return nil, err
	}

	buffs := NewBuffers()
	buffs.Attach(cmd)

	return &Cmd{
		Cmd:     cmd,
		Buffers: buffs,
	}, nil
}

func (c *Cmd) Close() {
	c.Buffers.Close()
}

func Run(cmd *Cmd) error {
	slog.Debug("Running command.", "args", cmd.Cmd.Args, "dir", cmd.Cmd.Dir)
	return DefaultRunner.Run(cmd)
}

func LookPath(fi string) (string, error) {
	return DefaultPathLooker.LookPath(fi)
}

var _ PathLooker = (PathLookerFunc)(nil)

type PathLookerFunc func(fi string) (string, error)

func (p PathLookerFunc) LookPath(fi string) (string, error) {
	return p(fi)
}

var _ Runner = (RunnerFunc)(nil)

type RunnerFunc func(cmd *Cmd) error

func (r RunnerFunc) Run(cmd *Cmd) error {
	return r(cmd)
}

type Buffers struct {
	Stdout *bytes.Buffer
	Stdin  *bytes.Buffer
	Stderr *bytes.Buffer
}

func NewBuffers() *Buffers {
	stdout, stderr, stdin := bufferpool.GetBytesBuffer(), bufferpool.GetBytesBuffer(), bufferpool.GetBytesBuffer()
	return &Buffers{
		Stdout: stdout,
		Stdin:  stdin,
		Stderr: stderr,
	}
}

func (b *Buffers) Copy() *Buffers {
	buff := NewBuffers()
	buff.Stdin.Write(b.Stdout.Bytes())
	buff.Stdin.Write(b.Stdin.Bytes())
	buff.Stderr.Write(b.Stderr.Bytes())
	return buff
}

func (b *Buffers) Attach(cmd *exec.Cmd) {
	cmd.Stdout, cmd.Stderr, cmd.Stdin = b.Stdout, b.Stderr, b.Stdin
}

func (b *Buffers) Close() {
	bufferpool.PutBytesBuffers(b.Stderr, b.Stdin, b.Stdout)
}

func (b *Buffers) Reset() {
	b.Stdout.Reset()
	b.Stderr.Reset()
	b.Stdin.Reset()
}

func EnvVarsToMap(evs []string) map[string]string {
	out := make(map[string]string, len(evs))
	for _, ev := range evs {
		idx := strings.Index(ev, "=")
		out[ev[:idx]] = ev[idx+1:]
	}

	return out
}
