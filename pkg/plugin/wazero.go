package plugin

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"sync"

	"github.com/skiff-sh/api/go/skiff/plugin/v1alpha1"
	"github.com/skiff-sh/sdk-go/skiffwasm"
	"github.com/skiff-sh/skiff/pkg/bufferpool"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"google.golang.org/protobuf/proto"
)

type Compiler interface {
	Compile(ctx context.Context, b []byte, rootFs fs.FS) (Plugin, error)
}

func NewWazeroCompiler() Compiler {
	run := wazero.NewRuntime(context.Background())
	return &wazeroCompiler{Runtime: run}
}

type Plugin interface {
	WriteFile(ctx context.Context, req *v1alpha1.WriteFileRequest) (*v1alpha1.WriteFileResponse, error)
	Stdout() io.Writer
	Stderr() io.Writer
	Stdin() io.Reader
	Close() error
}

type wazeroCompiler struct {
	Runtime wazero.Runtime
}

func (w *wazeroCompiler) Compile(ctx context.Context, b []byte, rootFs fs.FS) (Plugin, error) {
	stdout, stdin, stderr := bufferpool.GetBytesBuffer(), bufferpool.GetBytesBuffer(), bufferpool.GetBytesBuffer()
	mounts := wazero.NewFSConfig().WithFSMount(rootFs, "root")
	modConfig := wazero.NewModuleConfig().
		WithFSConfig(mounts).
		WithEnv(skiffwasm.EnvVarProjectPath, "root").
		WithEnv(skiffwasm.EnvVarMessageDelimiter, "\n").
		WithStdin(stdin).
		WithStdout(stdout).
		WithStderr(stderr).
		WithStartFunctions("main")

	mod, err := w.Runtime.InstantiateWithConfig(ctx, b, modConfig)
	if err != nil {
		return nil, err
	}

	handleRequestFunc := mod.ExportedFunction(skiffwasm.WASMFuncHandleRequestName)
	if handleRequestFunc == nil {
		return nil, fmt.Errorf("func %s must be exported in your plugin", skiffwasm.WASMFuncHandleRequestName)
	}

	def := handleRequestFunc.Definition()
	if resultTypes := def.ResultTypes(); len(resultTypes) != 1 || resultTypes[0] != api.ValueTypeI64 {
		return nil, fmt.Errorf("func %s must return a single int64", skiffwasm.WASMFuncHandleRequestName)
	}

	if paramTypes := def.ParamTypes(); len(paramTypes) != 1 || paramTypes[0] != api.ValueTypeI64 {
		return nil, fmt.Errorf("func %s must take a single int64", skiffwasm.WASMFuncHandleRequestName)
	}

	return &wazeroPlugin{
		Module:    mod,
		ProjectFS: rootFs,
	}, nil
}

type wazeroPlugin struct {
	Module api.Module

	HandleRequestFunc api.Function

	StdoutBuffer *bytes.Buffer
	StderrBuffer *bytes.Buffer
	StdinBuffer  *bytes.Buffer
	MessageDelim byte
	ProjectFS    fs.FS
	Closer       sync.Once
}

func (w *wazeroPlugin) Close() error {
	var err error
	w.Closer.Do(func() {
		bufferpool.PutBytesBuffer(w.StdinBuffer)
		bufferpool.PutBytesBuffer(w.StdoutBuffer)
		bufferpool.PutBytesBuffer(w.StderrBuffer)
		err = w.Module.Close(context.Background())
	})
	return err
}

func (w *wazeroPlugin) WriteFile(ctx context.Context, req *v1alpha1.WriteFileRequest) (*v1alpha1.WriteFileResponse, error) {
	b, err := proto.Marshal(req)
	if err != nil {
		return nil, err
	}

	_, err = w.StdinBuffer.Write(append(b, w.MessageDelim))
	if err != nil {
		return nil, err
	}

	res, err := w.HandleRequestFunc.Call(ctx, uint64(skiffwasm.RequestTypeWriteFile))
	if err != nil {
		return nil, err
	}

	if code := skiffwasm.ExitCode(res[0]); code != skiffwasm.ExitCodeOK {
		return nil, errors.New(code.String())
	}

	raw, err := w.StdoutBuffer.ReadBytes(w.MessageDelim)
	if err != nil {
		return nil, err
	}

	resp := new(v1alpha1.WriteFileResponse)
	err = proto.Unmarshal(raw, resp)
	return resp, err
}

func (w *wazeroPlugin) Stdout() io.Writer {
	return w.StdoutBuffer
}

func (w *wazeroPlugin) Stderr() io.Writer {
	return w.StderrBuffer
}

func (w *wazeroPlugin) Stdin() io.Reader {
	return w.StdinBuffer
}
