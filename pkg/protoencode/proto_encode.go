package protoencode

import "google.golang.org/protobuf/encoding/protojson"

var Unmarshaller = protojson.UnmarshalOptions{
	DiscardUnknown: true,
}
var Marshaller = protojson.MarshalOptions{}

var PrettyMarshaller = protojson.MarshalOptions{
	Multiline: true,
}
