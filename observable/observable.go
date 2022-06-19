package observable

import (
	"fmt"
	"sync"
)

type Observable[T any] struct {
	value       T
	observerMap map[string]func(T)
}

func NewObservable[T any]() *Observable[T] {
	var zeroVal T
	return &Observable[T]{zeroVal, make(map[string]func(T))}
}

func NewObservableWith[T any](v T) *Observable[T] {
	return &Observable[T]{v, make(map[string]func(T))}
}

func (o *Observable[T]) deleteIfExist(id string) {
	if o.observerMap[id] != nil {
		delete(o.observerMap, id)
	}
}

func (o *Observable[T]) Get() T {
	return o.value
}

func (o *Observable[T]) Set(v T) {
	o.value = v
	for _, fun := range o.observerMap {
		fun(v)
	}
}

func (o *Observable[T]) On(observer func(T)) func() {
	id := fmt.Sprintf("%p", &observer)
	o.observerMap[id] = observer
	return func() { o.deleteIfExist(id) }
}

func (o *Observable[T]) Once(observer func(T)) func() {
	id := fmt.Sprintf("%p", &observer)
	actual := func(v T) {
		observer(v)
		o.deleteIfExist(id)
	}
	o.observerMap[id] = actual
	return func() { o.deleteIfExist(id) }
}

func (o *Observable[T]) Off(id string) bool {
	if o.observerMap[id] == nil {
		return false
	}
	o.deleteIfExist(id)
	return true
}

func (o *Observable[T]) Dispose() {
	for k := range o.observerMap {
		delete(o.observerMap, k)
	}
}

type SafeObservable[T any] struct {
	o      *Observable[T]
	rwLock *sync.RWMutex
}

func NewSafeObservable[T any]() *SafeObservable[T] {
	return &SafeObservable[T]{NewObservable[T](), new(sync.RWMutex)}
}

func NewSafeObservableWith[T any](v T) *SafeObservable[T] {
	return &SafeObservable[T]{NewObservableWith[T](v), new(sync.RWMutex)}
}

func (o *SafeObservable[T]) Get() T {
	o.rwLock.RLock()
	defer o.rwLock.RUnlock()
	return o.o.Get()
}

func (o *SafeObservable[T]) Set(v T) {
	o.rwLock.Lock()
	o.o.value = v
	o.rwLock.Unlock()
	o.rwLock.RLock()
	defer o.rwLock.RUnlock()
	for _, fun := range o.o.observerMap {
		fun(v)
	}
}

func (o *SafeObservable[T]) On(observer func(T)) func() {
	o.rwLock.Lock()
	defer o.rwLock.Unlock()
	disposer := o.o.On(observer)
	return func() {
		o.rwLock.Lock()
		defer o.rwLock.Unlock()
		disposer()
	}
}

func (o *SafeObservable[T]) Once(observer func(T)) func() {
	o.rwLock.Lock()
	defer o.rwLock.Unlock()
	disposer := o.On(observer)
	return func() {
		o.rwLock.Lock()
		defer o.rwLock.Unlock()
		disposer()
	}
}

func (o *SafeObservable[T]) Off(id string) bool {
	o.rwLock.Lock()
	defer o.rwLock.Unlock()
	success := o.o.Off(id)
	return success
}

func (o *SafeObservable[T]) Dispose() {
	o.rwLock.Lock()
	defer o.rwLock.Unlock()
	o.o.Dispose()
}
