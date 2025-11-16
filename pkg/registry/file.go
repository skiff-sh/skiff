package registry

import (
	"fmt"

	"github.com/skiff-sh/config/ptr"
	"github.com/skiff-sh/skiff/api/go/skiff/registry/v1alpha1"
	"github.com/skiff-sh/skiff/pkg/bufferpool"
	"github.com/skiff-sh/skiff/pkg/filesystem"
	"github.com/skiff-sh/skiff/pkg/tmpl"
)

type FileGenerator struct {
	Proto *v1alpha1.File

	Target  tmpl.Template
	Content tmpl.Template
}

func (f *FileGenerator) Generate(d map[string]any) (*File, error) {
	out := &v1alpha1.File{
		Path: f.Proto.Path,
		Raw:  f.Proto.Raw,
		Type: f.Proto.Type,
	}

	buf := bufferpool.GetBytesBuffer()
	defer bufferpool.PutBytesBuffer(buf)
	err := f.Target.Render(d, buf)
	if err != nil {
		return nil, fmt.Errorf("failed to render 'target' template: %w", err)
	}
	out.Target = buf.String()
	buf.Reset()

	if f.Content != nil {
		err = f.Content.Render(d, buf)
		if err != nil {
			return nil, fmt.Errorf("failed to render 'contents' template: %w", err)
		}
		out.Content = ptr.Ptr(buf.String())
	}

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

	if p.Content != nil {
		content, err := t.NewTemplate([]byte(*p.Content))
		if err != nil {
			return nil, fmt.Errorf("invalid 'content' template expression: %w", err)
		}

		out.Content = content
	}

	return out, nil
}

type File struct {
	Proto *v1alpha1.File
}

func (f *File) WriteTo(fsys filesystem.Filesystem) error {
	if f.Proto.Content != nil {
		return fsys.WriteFile(f.Proto.Target, []byte(f.Proto.GetContent()))
	} else {
		return fsys.WriteFile(f.Proto.Target, f.Proto.Raw)
	}
}
