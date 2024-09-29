package slices

func ToMap[K comparable, T, V any](s []T, makeElem func(T) (K, V)) map[K]V {
	m := make(map[K]V, len(s))

	for _, elem := range s {
		k, v := makeElem(elem)
		m[k] = v
	}

	return m
}

func Map[T, V any](s []T, transformElem func(T) V) []V {
	m := make([]V, len(s))

	for i, elem := range s {
		v := transformElem(elem)
		m[i] = v
	}

	return m
}

func Reduce[A, T any](s []T, initial A, f func(acc A, value T) A) A {
	for _, val := range s {
		initial = f(initial, val)
	}
	return initial
}
