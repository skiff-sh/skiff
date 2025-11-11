package valid

import (
	"buf.build/go/protovalidate"
	"google.golang.org/protobuf/proto"
)

var Validator, _ = protovalidate.New()

func ValidateProto(p proto.Message) error {
	return Validator.Validate(p)
}
