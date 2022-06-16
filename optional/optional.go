// Package Optional Deprecated
package Optional

import (
	"github.com/dlshle/gommon/errors"
)

type Optional[T comparable] struct {
	val      T
	emptyVal T
}

func (o Optional[T]) GetOrError() (T, error) {
	if o.val == o.emptyVal {
		return o.emptyVal, errors.Error("empty Optional value")
	}
	return o.val, nil
}

func (o Optional[T]) GetOrPanic() T {
	val, err := o.GetOrError()
	if err != nil {
		panic(err)
	}
	return val
}

func (o Optional[T]) IsPresent() bool {
	return o.val != o.emptyVal
}

func (o Optional[T]) IfPresent(thenFunc func(T)) {
	if o.val == o.emptyVal {
		return
	}
	thenFunc(o.val)
}

func (o Optional[T]) Filter(filterFunc func(T) bool) Optional[T] {
	if o.val == o.emptyVal {
		return Of(o.emptyVal)
	}
	if filterFunc(o.val) {
		return o
	}
	return Of(o.emptyVal)
}

func (o Optional[T]) Map(mappingFunc func(val T) T) Optional[T] {
	if o.val == o.emptyVal {
		return Of(o.emptyVal)
	}
	return Of(mappingFunc(o.val))
}

func (o Optional[T]) OrElse(val T) T {
	if o.val == o.emptyVal {
		return val
	}
	return o.val
}

func (o Optional[T]) OrElseGet(getFunc func() T) T {
	if o.val == o.emptyVal {
		return getFunc()
	}
	return o.val
}

func (o Optional[T]) OrElsePanic(panicVal interface{}) T {
	if o.val == o.emptyVal {
		panic(panicVal)
	}
	return o.val
}

func Of[T comparable](val T) Optional[T] {
	return Optional[T]{
		val: val,
	}
}
