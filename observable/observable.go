package observable

import (
	"fmt"
	"sync"
)

type Observable struct {
	value       interface{}
	observerMap map[string]func(interface{})
}

func NewObservable() *Observable {
	return &Observable{nil, make(map[string]func(interface{}))}
}

func NewObservableWith(v interface{}) *Observable {
	return &Observable{v, make(map[string]func(interface{}))}
}

type SafeObservable struct {
	o      *Observable
	rwLock *sync.RWMutex
}

func NewSafeObservable() *SafeObservable {
	return &SafeObservable{NewObservable(), new(sync.RWMutex)}
}

func NewSafeObservableWith(v interface{}) *SafeObservable {
	return &SafeObservable{NewObservableWith(v), new(sync.RWMutex)}
}

type IObservable interface {
	Get() interface{}
	Set(interface{})
	On(func(interface{})) func()   //returns disposer function
	Once(func(interface{})) func() //returns disposer function
	Off(string) bool
	Dispose()
	deleteIfExist(id string)
}

func (o *Observable) deleteIfExist(id string) {
	if o.observerMap[id] != nil {
		delete(o.observerMap, id)
	}
}

func (o *Observable) Get() interface{} {
	return o.value
}

func (o *Observable) Set(v interface{}) {
	o.value = v
	for _, fun := range o.observerMap {
		fun(v)
	}
}

func (o *Observable) On(observer func(interface{})) func() {
	id := fmt.Sprintf("%p", &observer)
	o.observerMap[id] = observer
	return func() { o.deleteIfExist(id) }
}

func (o *Observable) Once(observer func(interface{})) func() {
	id := fmt.Sprintf("%p", &observer)
	actual := func(v interface{}) {
		observer(v)
		o.deleteIfExist(id)
	}
	o.observerMap[id] = actual
	return func() { o.deleteIfExist(id) }
}

func (o *Observable) Off(id string) bool {
	if o.observerMap[id] == nil {
		return false
	}
	o.deleteIfExist(id)
	return true
}

func (o *Observable) Dispose() {
	for k, _ := range o.observerMap {
		delete(o.observerMap, k)
	}
}

func (o *SafeObservable) Get() interface{} {
	o.rwLock.RLock()
	defer o.rwLock.RUnlock()
	return o.o.Get()
}

func (o *SafeObservable) Set(v interface{}) {
	o.rwLock.Lock()
	o.o.value = v
	o.rwLock.Unlock()
	o.rwLock.RLock()
	defer o.rwLock.RUnlock()
	for _, fun := range o.o.observerMap {
		fun(v)
	}
}

func (o *SafeObservable) On(observer func(interface{})) func() {
	o.rwLock.Lock()
	defer o.rwLock.Unlock()
	disposer := o.On(observer)
	return func() {
		o.rwLock.Lock()
		defer o.rwLock.Unlock()
		disposer()
	}
}

func (o *SafeObservable) Once(observer func(interface{})) func() {
	o.rwLock.Lock()
	defer o.rwLock.Unlock()
	disposer := o.On(observer)
	return func() {
		o.rwLock.Lock()
		defer o.rwLock.Unlock()
		disposer()
	}
}

func (o *SafeObservable) Off(id string) bool {
	o.rwLock.Lock()
	defer o.rwLock.Unlock()
	success := o.o.Off(id)
	return success
}

func (o *SafeObservable) Dispose() {
	o.rwLock.Lock()
	defer o.rwLock.Unlock()
	o.o.Dispose()
}
