package collection

func Map[S ~[]E, E, T any](s S, mapper func(e E) T) []T {
	out := make([]T, 0, len(s))
	for _, v := range s {
		out = append(out, mapper(v))
	}
	return out
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
