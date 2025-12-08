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

func (w *WazeroTestSuite) TestWriteFile() {
	type test struct {
		SourceName string
	}

	tests := map[string]test{
		"basic plugin": {
			SourceName: "basic_plugin",
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
			plug, err := compiler.Compile(ctx, b, fstest.MapFS{})
			if !w.NoError(err) {
				return
			}

			timer := time.Now()
			resp, err := plug.WriteFile(ctx, &v1alpha1.WriteFileRequest{})
			if !w.NoError(err) {
				fmt.Println(plug.Stderr().String())
				return
			}
			total := time.Since(timer)

			w.Less(total, 5*time.Millisecond)
			w.Equal("hi", string(resp.Contents))
		})
	}
}

func TestWazeroTestSuite(t *testing.T) {
	suite.Run(t, new(WazeroTestSuite))
}
