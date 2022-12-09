package gr_context

import (
	"strings"
	"sync"

	"github.com/petermattis/goid"
)

// a goroutine local context maintainer

// no need to use lock on inner map(goroutine local map) as operations are on the same goroutine
var context = map[int64]map[string]interface{}{}
var mutex = new(sync.Mutex)

func withLock(cb func()) {
	mutex.Lock()
	defer mutex.Unlock()
	cb()
}

func getGRContextMap() (cm map[string]interface{}) {
	withLock(func() {
		id := goid.Get()
		cm = context[id]
		if cm == nil {
			cm = make(map[string]interface{})
			context[id] = cm
		}
	})
	return
}

func unsafeGetGRContextMap() (m map[string]interface{}) {
	withLock(func() {
		m = context[goid.Get()]
	})
	return
}

func Put(key string, v interface{}) {
	getGRContextMap()[key] = v
}

func Get(key string) interface{} {
	m := unsafeGetGRContextMap()
	if m == nil {
		return nil
	}
	return m[key]
}

func GetByPrefix(prefix string) (result map[string]interface{}) {
	m := unsafeGetGRContextMap()
	result = make(map[string]interface{})
	if m == nil {
		return
	}
	for k := range m {
		if strings.HasPrefix(k, prefix) {
			result[k] = m[k]
		}
	}
	return
}

func Delete(key string) {
	m := unsafeGetGRContextMap()
	if m == nil {
		return
	}
	delete(m, key)
}

func Clear() {
	withLock(func() {
		delete(context, goid.Get())
	})
}

func ClearByPrefix(prefix string) {
	m := unsafeGetGRContextMap()
	if m == nil {
		return
	}
	for k := range m {
		if strings.HasPrefix(k, prefix) {
			delete(m, k)
		}
	}
}
