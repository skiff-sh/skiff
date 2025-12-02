package execcmd

import (
	"bytes"
	"context"
	"os/exec"

	"github.com/skiff-sh/skiff/pkg/bufferpool"
)

// Runner runs an exec.Cmd. Useful for tracking and testing command runs.
type Runner interface {
	Run(cmd *Cmd) error
}

type Cmd struct {
	Cmd     *exec.Cmd
	Buffers *Buffers
}

func (c *Cmd) Close() {
	c.Buffers.Close()
}

func NewCmd(ctx context.Context, name string, args ...string) (*Cmd, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if cmd.Err != nil {
		return nil, cmd.Err
	}

	buffs := NewBuffers()
	buffs.Attach(cmd)

	return &Cmd{
		Cmd:     cmd,
		Buffers: buffs,
	}, nil
}

func Run(cmd *Cmd) error {
	return DefaultRunner.Run(cmd)
}

// DefaultRunner is the default Runner for the package.
var DefaultRunner Runner = RunnerFunc(func(cmd *Cmd) error {
	return cmd.Cmd.Run()
})

var _ Runner = (RunnerFunc)(nil)

type RunnerFunc func(cmd *Cmd) error

func (r RunnerFunc) Run(cmd *Cmd) error {
	return r(cmd)
}

func NewBuffers() *Buffers {
	stdout, stderr, stdin := bufferpool.GetBytesBuffer(), bufferpool.GetBytesBuffer(), bufferpool.GetBytesBuffer()
	return &Buffers{
		Stdout: stdout,
		Stdin:  stdin,
		Stderr: stderr,
	}
}

type Buffers struct {
	Stdout *bytes.Buffer
	Stdin  *bytes.Buffer
	Stderr *bytes.Buffer
}

func (b *Buffers) Attach(cmd *exec.Cmd) {
	cmd.Stdout, cmd.Stderr, cmd.Stdin = b.Stdout, b.Stderr, b.Stdin
}

func (b *Buffers) Close() {
	bufferpool.PutBytesBuffers(b.Stderr, b.Stdin, b.Stdout)
}
