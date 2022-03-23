package data_structures

type Iterable interface {
	ForEach(cb func(item interface{}, index int))
	Map(cb func(item interface{}, index int) interface{}) Iterable
	// ReduceLeft(cb func(accu interface{}, curr interface{}) interface{}) interface{}
	// ReduceRight(cb func(accu interface{}, curr interface{}) interface{}) interface{}
}
