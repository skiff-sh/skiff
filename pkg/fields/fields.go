package fields

import (
	"strconv"

	"golang.org/x/exp/constraints"
	"google.golang.org/protobuf/types/known/structpb"
)

func Cast[T any](v any) (out T) {
	out, _ = As[T](v)
	return out
}

func As[T any](v any) (out T, ok bool) {
	if v == nil {
		ok = true
		return out, ok
	}

	out, ok = v.(T)
	return out, ok
}

func FormatFloat[F constraints.Float](f F) string {
	return strconv.FormatFloat(float64(f), 'g', 3, 64)
}

func ParseFloat[F constraints.Float](s string) (F, error) {
	v, err := strconv.ParseFloat(s, 64)
	return F(v), err
}

// SubInt64 is basically constraints.Integer but excludes any types that don't fit into int64.
type SubInt64 interface {
	constraints.Signed |
		~uint | ~uint8 | ~uint16 | ~uint32
}

func Atoi[I SubInt64](i I) string {
	return strconv.FormatInt(int64(i), 10)
}

// Provider is a singleton wrapper that is only instantiated once. It's the same as sync.OnceValue but without the additional overhead
// of supporting concurrent calls.
type Provider[T any] func() T

// NewProvider constructs a new Provider.
func NewProvider[T any](t Provider[T]) Provider[T] {
	called := false
	var val T
	return func() T {
		if !called {
			val = t()
			called = true
		}
		return val
	}
}

func NewListValue(items ...any) *structpb.ListValue {
	v, _ := structpb.NewList(items)
	return v
}

func NewValue(a any) *structpb.Value {
	v, _ := structpb.NewValue(a)
	return v
}

type Number interface {
	constraints.Integer | constraints.Float
}

func ToFloat64[T Number](t T) float64 {
	return float64(t)
}
