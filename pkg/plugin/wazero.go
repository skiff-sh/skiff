package plugin

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"sync"

	"github.com/skiff-sh/api/go/skiff/plugin/v1alpha1"
	"github.com/skiff-sh/sdk-go/pluginapi"
	"github.com/skiff-sh/skiff/pkg/bufferpool"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"google.golang.org/protobuf/proto"
)

type Compiler interface {
	Compile(ctx context.Context, b []byte, rootFs fs.FS) (Plugin, error)
}

func NewWazeroCompiler() (Compiler, error) {
	ctx := context.Background()
	run := wazero.NewRuntime(ctx)
	cl, err := wasi_snapshot_preview1.Instantiate(ctx, run)
	if err != nil {
		return nil, err
	}

	return &wazeroCompiler{Runtime: run, Closer: cl}, nil
}

type Plugin interface {
	WriteFile(ctx context.Context, req *v1alpha1.WriteFileRequest) (*v1alpha1.WriteFileResponse, error)
	Stdout() *bytes.Buffer
	Stderr() *bytes.Buffer
	Stdin() *bytes.Buffer
	Close() error
}

type wazeroCompiler struct {
	Runtime wazero.Runtime
	Closer  api.Closer
}

func (w *wazeroCompiler) Compile(ctx context.Context, b []byte, rootFs fs.FS) (Plugin, error) {
	stdout, stdin, stderr := bufferpool.GetBytesBuffer(), bufferpool.GetBytesBuffer(), bufferpool.GetBytesBuffer()
	mounts := wazero.NewFSConfig().WithFSMount(rootFs, "root")
	modConfig := wazero.NewModuleConfig().
		WithFSConfig(mounts).
		WithEnv(pluginapi.EnvVarProjectPath, "root").
		WithEnv(pluginapi.EnvVarMessageDelimiter, "\r").
		WithStdout(stdout).
		WithStdin(stdin).
		WithStderr(stderr).
		WithStartFunctions("_initialize", "_start")

	mod, err := w.Runtime.InstantiateWithConfig(ctx, b, modConfig)
	if err != nil {
		return nil, err
	}

	handleRequestFunc := mod.ExportedFunction(pluginapi.WASMFuncHandleRequestName)
	if handleRequestFunc == nil {
		return nil, fmt.Errorf("func %s must be exported in your plugin", pluginapi.WASMFuncHandleRequestName)
	}

	def := handleRequestFunc.Definition()
	if resultTypes := def.ResultTypes(); len(resultTypes) != 1 || resultTypes[0] != api.ValueTypeI64 {
		return nil, fmt.Errorf("func %s must return a single int64", pluginapi.WASMFuncHandleRequestName)
	}

	if paramTypes := def.ParamTypes(); len(paramTypes) != 1 || paramTypes[0] != api.ValueTypeI64 {
		return nil, fmt.Errorf("func %s must take a single int64", pluginapi.WASMFuncHandleRequestName)
	}

	return &wazeroPlugin{
		Module:            mod,
		HandleRequestFunc: handleRequestFunc,
		StdoutBuffer:      stdout,
		StderrBuffer:      stderr,
		StdinBuffer:       stdin,
		MessageDelim:      '\r',
		ProjectFS:         rootFs,
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
	b = append(b, w.MessageDelim)

	_, err = w.StdinBuffer.Write(b)
	if err != nil {
		return nil, err
	}

	res, err := w.HandleRequestFunc.Call(ctx, uint64(pluginapi.RequestTypeWriteFile))
	if err != nil {
		return nil, err
	}

	if code := pluginapi.ExitCode(res[0]); code != pluginapi.ExitCodeOK {
		return nil, errors.New(code.String())
	}

	raw, err := w.StdoutBuffer.ReadBytes(w.MessageDelim)
	if err != nil {
		return nil, err
	}

	resp := new(v1alpha1.WriteFileResponse)
	if len(raw) == 0 {
		return resp, nil
	}
	// Drop delimiter
	raw = raw[:len(raw)-1]
	err = proto.Unmarshal(raw, resp)
	return resp, err
}

func (w *wazeroPlugin) Stdout() *bytes.Buffer {
	return w.StdoutBuffer
}

func (w *wazeroPlugin) Stderr() *bytes.Buffer {
	return w.StderrBuffer
}

func (w *wazeroPlugin) Stdin() *bytes.Buffer {
	return w.StdinBuffer
}
