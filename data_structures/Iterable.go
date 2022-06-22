package data_structures

type Iterable[T comparable] interface {
	ForEach(cb func(item T, index int))
	Map(cb func(item T, index int) T) Iterable[T]
	// left to right
	Reduce(cb func(accu T, curr T) T, initialVal T) T
}
