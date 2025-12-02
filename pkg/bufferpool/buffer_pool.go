package bufferpool

import (
	"bytes"
	"sync"
)

var bytesBufferPool = sync.Pool{
	New: func() any {
		return bytes.NewBuffer(nil)
	},
}

func GetBytesBuffer() *bytes.Buffer {
	//nolint:errcheck // Pool contains one type.
	buf := bytesBufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

func PutBytesBuffer(buf *bytes.Buffer) {
	bytesBufferPool.Put(buf)
}

func PutBytesBuffers(buf ...*bytes.Buffer) {
	for i := range buf {
		bytesBufferPool.Put(buf[i])
	}
}
