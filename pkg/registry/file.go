package registry

import (
	"fmt"

	"github.com/skiff-sh/skiff/api/go/skiff/registry/v1alpha1"
	"github.com/skiff-sh/skiff/pkg/bufferpool"
	"github.com/skiff-sh/skiff/pkg/filesystem"
	"github.com/skiff-sh/skiff/pkg/tmpl"
	"google.golang.org/protobuf/proto"
)

type FileGenerator struct {
	Proto *v1alpha1.File

	Target  tmpl.Template
	Content tmpl.Template
}

func (f *FileGenerator) Generate(d map[string]any) (*File, error) {
	out := proto.CloneOf(f.Proto)

	buf := bufferpool.GetBytesBuffer()
	defer bufferpool.PutBytesBuffer(buf)
	err := f.Target.Render(d, buf)
	if err != nil {
		return nil, fmt.Errorf("failed to render 'target' template: %w", err)
	}
	out.Target = buf.String()
	buf.Reset()

	err = f.Content.Render(d, buf)
	if err != nil {
		return nil, fmt.Errorf("failed to render 'contents' template: %w", err)
	}

	out.Content = make([]byte, buf.Len())
	copy(out.Content, buf.Bytes())
	return &File{Proto: out}, nil
}

func NewFileGenerator(t tmpl.Factory, p *v1alpha1.File) (*FileGenerator, error) {
	out := &FileGenerator{
		Proto: p,
	}

	var err error
	out.Target, err = t.NewTemplate([]byte(p.Target))
	if err != nil {
		return nil, fmt.Errorf("invalid 'target' template expression: %w", err)
	}

	out.Content, err = t.NewTemplate(p.Content)
	if err != nil {
		return nil, fmt.Errorf("invalid 'content' template expression: %w", err)
	}

	return out, nil
}

type File struct {
	Proto *v1alpha1.File
}

func (f *File) WriteTo(fsys filesystem.Filesystem) error {
	return fsys.WriteFile(f.Proto.Target, f.Proto.Content)
}
