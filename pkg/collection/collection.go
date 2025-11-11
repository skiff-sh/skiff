package collection

func Map[S ~[]E, E, T any](s S, mapper func(e E) T) []T {
	out := make([]T, 0, len(s))
	for _, v := range s {
		out = append(out, mapper(v))
	}
	return out
}
