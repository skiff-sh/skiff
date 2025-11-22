package testutil

import (
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
)

// DiffProto returns the diff of two messages and an empty string if there
// is no difference.
func DiffProto(expected, actual proto.Message) string {
	return cmp.Diff(expected, actual, protocmp.Transform())
}
