package plugin

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"testing"
	"testing/fstest"
	"time"

	"github.com/skiff-sh/api/go/skiff/plugin/v1alpha1"
	"github.com/stretchr/testify/suite"
)

type WazeroTestSuite struct {
	suite.Suite
}

//go:embed testdata/*
var testdata embed.FS

func (w *WazeroTestSuite) TestSendRequest() {
	type test struct {
		SourceName string
		Opts       CompileOpts
		Expected   *v1alpha1.Response
		Given      *v1alpha1.Request
		RunCount   int
	}

	tests := map[string]test{
		"basic plugin": {
			SourceName: "basic_plugin",
			RunCount:   5,
			Expected:   &v1alpha1.Response{WriteFile: &v1alpha1.WriteFileResponse{Contents: []byte("hi")}},
			Given:      &v1alpha1.Request{WriteFile: &v1alpha1.WriteFileRequest{}},
		},
		"file access": {
			SourceName: "file_access_plugin",
			Expected:   &v1alpha1.Response{WriteFile: &v1alpha1.WriteFileResponse{Contents: []byte("derp")}},
			Given:      &v1alpha1.Request{WriteFile: &v1alpha1.WriteFileRequest{}},
			Opts: CompileOpts{
				Mounts: []*Mount{
					{
						GuestPath: guestCWDPath,
						Dir:       fstest.MapFS{"derp.txt": {Data: []byte("derp")}},
					},
				},
			},
		},
	}

	for desc, v := range tests {
		w.Run(desc, func() {
			b, err := fs.ReadFile(testdata, path.Join("testdata", v.SourceName+".wasm"))
			if !w.NoError(err) {
				return
			}

			compiler, err := NewWazeroCompiler()
			if !w.NoError(err) {
				return
			}

			ctx := w.T().Context()
			plug, err := compiler.Compile(ctx, b, v.Opts)
			if !w.NoError(err) {
				return
			}

			runCount := v.RunCount
			if runCount == 0 {
				runCount = 1
			}

			timer := time.Now()
			for range runCount {
				resp, err := plug.SendRequest(ctx, v.Given)
				if !w.NoError(err) {
					fmt.Println(string(resp.Logs()))
					return
				}
				if !w.Equal(v.Expected, resp.Body) {
					fmt.Println(string(resp.Logs()))
				}
			}
			total := time.Since(timer)
			w.Less(total, time.Duration(runCount+1)*time.Millisecond)
		})
	}
}

func TestWazeroTestSuite(t *testing.T) {
	suite.Run(t, new(WazeroTestSuite))
}
