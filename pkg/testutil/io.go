package testutil

import (
	"io"

	"github.com/skiff-sh/skiff/pkg/bufferpool"
)

func Dump(r io.Reader) string {
	buf := bufferpool.GetBytesBuffer()
	defer bufferpool.PutBytesBuffer(buf)
	_, _ = io.Copy(buf, r)
	return buf.String()
}
