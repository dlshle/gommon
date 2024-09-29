package slices

func Filter[T any](s []T, keep func(T) bool) []T {
	m := make([]T, 0, len(s))

	for _, elem := range s {
		if keep(elem) {
			m = append(m, elem)
		}
	}

	return m
}
