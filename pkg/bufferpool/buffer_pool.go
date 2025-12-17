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

var bytesReaderPool = sync.Pool{
	New: func() any {
		return bytes.NewReader(nil)
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

func GetBytesReader(b []byte) *bytes.Reader {
	//nolint:errcheck // Pool contains one type.
	buf := bytesReaderPool.Get().(*bytes.Reader)
	buf.Reset(b)
	return buf
}

func PutBytesReader(buf *bytes.Reader) {
	bytesReaderPool.Put(buf)
}

func PutBytesReaders(buf ...*bytes.Reader) {
	for i := range buf {
		bytesReaderPool.Put(buf[i])
	}
}
