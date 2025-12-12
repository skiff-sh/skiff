package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"sync"

	"github.com/skiff-sh/api/go/skiff/plugin/v1alpha1"
	"github.com/skiff-sh/sdk-go/skiff/pluginapi"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"

	"github.com/skiff-sh/skiff/pkg/bufferpool"
	"github.com/skiff-sh/skiff/pkg/execcmd"
)

type Compiler interface {
	Compile(ctx context.Context, b []byte, opts CompileOpts) (Plugin, error)
}

type CompileOpts struct {
	CWDPath string
	Mounts  []*Mount
}

type Mount struct {
	GuestPath string
	Dir       fs.FS
}

const (
	guestCWDPath = "/cwd"
)

func NewWazeroCompiler() (Compiler, error) {
	ctx := context.Background()
	run := wazero.NewRuntime(ctx)
	cl, err := wasi_snapshot_preview1.Instantiate(ctx, run)
	if err != nil {
		return nil, err
	}

	return &wazeroCompiler{Runtime: run, Closer: cl}, nil
}

type Response struct {
	Body   *v1alpha1.Response
	logs   *bytes.Buffer
	closer sync.Once
}

func (r *Response) Logs() []byte {
	if r == nil {
		return nil
	}
	return r.logs.Bytes()
}

func (r *Response) Close() {
	r.closer.Do(func() {
		bufferpool.PutBytesBuffers(r.logs)
	})
}

type Plugin interface {
	// SendRequest sends a request to the plugin. Cannot be called concurrently.
	SendRequest(ctx context.Context, req *v1alpha1.Request) (*Response, error)
	Close() error
}

type wazeroCompiler struct {
	Runtime wazero.Runtime
	Closer  api.Closer
}

func (w *wazeroCompiler) Compile(ctx context.Context, b []byte, opts CompileOpts) (Plugin, error) {
	buff := execcmd.NewBuffers()

	modConfig := wazero.NewModuleConfig().
		WithEnv(pluginapi.EnvVarMessageDelimiter, "\r").
		WithStdout(buff.Stdout).
		WithStdin(buff.Stdin).
		WithStderr(buff.Stderr).
		WithEnv(pluginapi.EnvVarCWD, guestCWDPath).
		WithEnv(pluginapi.EnvVarCWDHost, opts.CWDPath).
		WithStartFunctions("_initialize", "_start")

	mounts := wazero.NewFSConfig()
	if opts.CWDPath != "" {
		mounts = mounts.WithReadOnlyDirMount(opts.CWDPath, guestCWDPath)
	} else {
		mounts = mounts.WithFSMount(new(noOpFS), guestCWDPath)
	}

	for _, v := range opts.Mounts {
		mounts = mounts.WithFSMount(v.Dir, v.GuestPath)
	}

	modConfig = modConfig.WithFSConfig(mounts)

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

	return &wazeroPlugin{
		Module:            mod,
		Buffer:            buff,
		HandleRequestFunc: handleRequestFunc,
		MessageDelim:      '\r',
	}, nil
}

type wazeroPlugin struct {
	Module api.Module

	HandleRequestFunc api.Function

	Buffer       *execcmd.Buffers
	MessageDelim byte
	Closer       sync.Once
}

func (w *wazeroPlugin) Close() error {
	var err error
	w.Closer.Do(func() {
		w.Buffer.Close()
		err = w.Module.Close(context.Background())
	})
	return err
}

func (w *wazeroPlugin) SendRequest(ctx context.Context, req *v1alpha1.Request) (*Response, error) {
	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	b = append(b, w.MessageDelim)

	defer w.Buffer.Reset()
	_, err = w.Buffer.Stdin.Write(b)
	if err != nil {
		return nil, err
	}

	res, err := w.HandleRequestFunc.Call(ctx)
	if err != nil {
		return nil, err
	}

	if code := pluginapi.ExitCode(res[0]); code != pluginapi.ExitCodeOK {
		return nil, errors.New(code.String())
	}

	raw, err := w.Buffer.Stdout.ReadBytes(w.MessageDelim)
	if err != nil {
		return nil, err
	}

	if len(raw) == 0 {
		return nil, errors.New("plugin returned empty response")
	}

	resp := new(v1alpha1.Response)
	// Drop delimiter
	raw = raw[:len(raw)-1]
	err = json.Unmarshal(raw, resp)
	if err != nil {
		return nil, err
	}

	out := &Response{
		Body: resp,
		logs: bufferpool.GetBytesBuffer(),
	}

	_, _ = out.logs.Write(w.Buffer.Stderr.Bytes())

	return out, nil
}

var _ fs.FS = (*noOpFS)(nil)

type noOpFS struct {
}

func (n *noOpFS) Open(_ string) (fs.File, error) {
	return nil, fs.ErrNotExist
}
