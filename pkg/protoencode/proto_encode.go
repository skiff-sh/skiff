package protoencode

import (
	"io"
	"os"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/skiff-sh/skiff/pkg/bufferpool"
)

var Unmarshaller = protojson.UnmarshalOptions{
	DiscardUnknown: true,
}
var Marshaller = protojson.MarshalOptions{}

var PrettyMarshaller = protojson.MarshalOptions{
	Multiline: true,
}

func Load(r io.Reader, p proto.Message) error {
	buf := bufferpool.GetBytesBuffer()
	defer bufferpool.PutBytesBuffer(buf)
	_, err := io.Copy(buf, r)
	if err != nil {
		return err
	}

	err = Unmarshaller.Unmarshal(buf.Bytes(), p)
	return err
}

func LoadFile(path string, p proto.Message) error {
	fi, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() {
		_ = fi.Close()
	}()

	return Load(fi, p)
}

func Marshal(msg proto.Message) ([]byte, error) {
	return Marshaller.Marshal(msg)
}

func Unmarshal(b []byte, msg proto.Message) error {
	return Unmarshaller.Unmarshal(b, msg)
}
