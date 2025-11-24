package plugin

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/skiff-sh/skiff/api/go/skiff/plugin/v1alpha1"
	"github.com/skiff-sh/skiff/pkg/fileutil"
)

type YaegiTestSuite struct {
	suite.Suite
}

func (y *YaegiTestSuite) Test() {
	ctx := y.T().Context()
	fileutil.CallerPath(1)
	pl := &Plugin{
		Content: []byte(`package main

import (
	"context"

	"github.com/skiff-sh/skiff/api/go/skiff/plugin/v1alpha1"
)

func WriteFile(ctx context.Context, req *v1alpha1.WriteFileRequest) (*v1alpha1.WriteFileResponse, error) {
	return &v1alpha1.WriteFileResponse{Contents: []byte("hi")}, nil
}
`),
	}

	r, err := NewYaegiInterpreter(pl)
	if !y.NoError(err) {
		return
	}

	resp, err := r.WriteFile(&Context{Ctx: ctx}, &v1alpha1.WriteFileRequest{})
	if !y.NoError(err) {
		return
	}

	y.Equal("hi", string(resp.GetContents()))
}

func TestYaegiTestSuite(t *testing.T) {
	suite.Run(t, new(YaegiTestSuite))
}
