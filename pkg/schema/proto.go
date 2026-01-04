package schema

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"
)

func NewProtoMapValues(val map[string]*structpb.Value) ([]Entry, error) {
	out := make([]Entry, 0, len(val))
	for k, v := range val {
		va, err := NewVal(v.AsInterface())
		if err != nil {
			return nil, fmt.Errorf("%s contains an invalid value: %w", k, err)
		}

		out = append(out, &protoValues{
			Val:  va,
			Name: k,
		})
	}

	return out, nil
}

type protoValues struct {
	Val  Value
	Name string
}

func (p *protoValues) Value() Value {
	return p.Val
}

func (p *protoValues) FieldName() string {
	return p.Name
}
