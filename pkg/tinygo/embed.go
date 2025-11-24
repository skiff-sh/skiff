package tinygo

import (
	"io"
	"io/fs"
)

type Compiler interface {
	CompileInto(f fs.FS, path string, w io.Writer) error
}

func NewCompiler() (Compiler, error) {
	out := &tinygoCompiler{}

	return out, nil
}

type tinygoCompiler struct {
}
