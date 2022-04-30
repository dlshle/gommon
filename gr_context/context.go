package gr_context

import (
	"github.com/petermattis/goid"
	"strings"
	"sync"
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

func Put(key string, v interface{}) {
	getGRContextMap()[key] = v
}

func Get(key string) interface{} {
	return getGRContextMap()[key]
}

func GetByPrefix(prefix string) map[string]interface{} {
	m := getGRContextMap()
	subSet := make(map[string]interface{})
	for k := range m {
		if strings.HasPrefix(k, prefix) {
			subSet[k] = m[k]
		}
	}
	return subSet
}

func Delete(key string) {
	m := getGRContextMap()
	delete(m, key)
}

func Clear() {
	withLock(func() {
		delete(context, goid.Get())
	})
}

func ClearByPrefix(prefix string) {
	withLock(func() {
		m := context[goid.Get()]
		if m == nil {
			return
		}
		for k := range m {
			if strings.HasPrefix(k, prefix) {
				delete(m, k)
			}
		}
	})
}
