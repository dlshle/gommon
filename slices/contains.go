package slices

import "slices"

func Contains[E comparable](s []E, v E) bool {
	return slices.Contains(s, v)
}
