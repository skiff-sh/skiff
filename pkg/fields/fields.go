package fields

import (
	"strconv"

	"golang.org/x/exp/constraints"
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

// SubInt64 is basically constraints.Integer but excludes any types that don't fit into int64
type SubInt64 interface {
	constraints.Signed |
		~uint | ~uint8 | ~uint16 | ~uint32
}

func Atoi[I SubInt64](i I) string {
	return strconv.FormatInt(int64(i), 10)
}
