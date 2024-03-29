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

func (o Optional[T]) Transform(mappingFunc func(val T) T) Optional[T] {
	if o.val == o.emptyVal {
		return Of(o.emptyVal)
	}
	return Of(mappingFunc(o.val))
}

func Map[T, K comparable](optional Optional[T], mapper func(T) K) Optional[K] {
	var zeroK K
	if optional.IsPresent() {
		return Of(mapper(optional.val))
	}
	return Of(zeroK)
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

func compose(fns ...func(any) any) func(any) any {
	if fns == nil {
		panic("composing function shuold take at least 1 input function")
	}
	return func(input any) (mapped any) {
		var res any
		for i, fn := range fns {
			if i == 0 {
				res = fn(input)
			} else {
				res = fn(res)
			}
		}
		return res
	}
}
