package plugins

import (
	"bufio"
	"context"
	"strings"

	"github.com/skiff-sh/skiff/api/go/skiff/plugin/v1alpha1"
)

func WriteFile(ctx context.Context, req *v1alpha1.WriteFileRequest) (*v1alpha1.WriteFileResponse, error) {
	scanner := bufio.NewScanner(strings.NewReader(string(req.File.Content)))
	for scanner.Scan() {
		req.File.get
	}
	return nil, nil
}
