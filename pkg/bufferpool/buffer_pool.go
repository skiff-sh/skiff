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
	buf := bytesBufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

func PutBytesBuffer(buf *bytes.Buffer) {
	bytesBufferPool.Put(buf)
}
