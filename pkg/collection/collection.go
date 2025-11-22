package collection

import "github.com/skiff-sh/skiff/pkg/bufferpool"

func Map[S ~[]E, E, T any](s S, mapper func(e E) T) []T {
	out := make([]T, 0, len(s))
	for _, v := range s {
		out = append(out, mapper(v))
	}
	return out
}

func MapOrErr[S ~[]E, E, T any](s S, mapper func(e E) (T, error)) ([]T, error) {
	out := make([]T, 0, len(s))
	for _, v := range s {
		o, err := mapper(v)
		if err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, nil
}

func Keys[K comparable, V any](m map[K]V) []K {
	out := make([]K, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func Values[K comparable, V any](m map[K]V) []V {
	out := make([]V, 0, len(m))
	for k := range m {
		out = append(out, m[k])
	}
	return out
}

func Filter[S ~[]E, E any](s S, f func(e E) bool) S {
	out := make(S, 0, len(s))
	for i := range s {
		v := s[i]
		if f(v) {
			out = append(out, v)
		}
	}
	return out
}

func Suffix[T ~string](suffix string, t ...T) string {
	buf := bufferpool.GetBytesBuffer()
	defer bufferpool.PutBytesBuffer(buf)
	for _, v := range t {
		buf.WriteString(string(v))
		buf.WriteString(suffix)
	}
	return buf.String()
}
