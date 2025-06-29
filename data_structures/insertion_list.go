package data_structures

import (
	"sort"
	"sync"

	"github.com/dlshle/gommon/slices"
)

type InsertionList[T comparable] interface {
	Container
	Get(index int) T
	Head() T
	Tail() T
	Add(item T)
	Filter(f func(T) bool) InsertionList[T]
	Remove(item T) bool
	RemoveAt(index int) bool
	Find(item T) int
	Has(item T) bool
	AsSlice() []T
	Clear()
}

type insertionList[T comparable] struct {
	list       []T
	comparator func(l T, r T) int
}

func NewInsertionList[T comparable](comparator func(l T, r T) int) InsertionList[T] {
	return &insertionList[T]{
		list:       make([]T, 0),
		comparator: comparator,
	}
}

func NewInsertionListOf[T comparable](l []T, comparator func(l T, r T) int) InsertionList[T] {
	copy := make([]T, len(l))
	for i := range l {
		copy[i] = l[i]
	}
	sort.Slice(copy, func(i, j int) bool {
		return comparator(l[i], l[j]) < 0
	})
	return &insertionList[T]{
		list:       copy,
		comparator: comparator,
	}
}

func (l *insertionList[T]) Get(index int) T {
	var zeroVal T
	if index < 0 || index > len(l.list) {
		return zeroVal
	}
	return l.list[index]
}

func (l *insertionList[T]) Head() T {
	return l.Get(0)
}

func (l *insertionList[T]) Tail() T {
	return l.Get(len(l.list) - 1)
}

func (l *insertionList[T]) Filter(f func(T) bool) InsertionList[T] {
	filtered := slices.Filter(l.list, f)
	return &insertionList[T]{
		comparator: l.comparator,
		list:       filtered,
	}
}

func (l *insertionList[T]) Add(item T) {
	if len(l.list) == 0 {
		l.list = append(l.list, item)
		return
	}
	// find the first one that's >= item
	left, right := 0, len(l.list)
	for left < right {
		m := (left + right) / 2
		if l.comparator(l.list[m], item) >= 0 {
			right = m
		} else {
			left = m + 1
		}
	}
	if right == len(l.list) {
		l.list = append(l.list, item)
		return
	}
	l.list = append(l.list[:right], append([]T{item}, l.list[right:]...)...)
}

func (l *insertionList[T]) Remove(item T) bool {
	index := l.Find(item)
	if index == -1 {
		return false
	}
	return l.RemoveAt(index)
}

func (l *insertionList[T]) RemoveAt(index int) bool {
	if index < 0 || index > len(l.list) {
		return false
	}
	l.list = append(l.list[:index], l.list[index+1:]...)
	return true
}

func (l *insertionList[T]) Clear() {
	l.list = nil
	l.list = make([]T, 0)
}

func (l *insertionList[T]) Find(item T) int {
	if len(l.list) == 0 {
		return -1
	}
	left, right := 0, len(l.list)-1
	for left <= right {
		m := (left + right) / 2
		if l.comparator(l.list[m], item) == 0 {
			return m
		}
		if l.comparator(l.list[m], item) > 0 {
			right = m - 1
		} else {
			left = m + 1
		}
	}
	return -1
}

func (l *insertionList[T]) AsSlice() []T {
	slice := make([]T, len(l.list), len(l.list))
	for i, item := range l.list {
		slice[i] = item
	}
	return slice
}

func (l *insertionList[T]) Has(item T) bool {
	return l.Find(item) > -1
}

func (l *insertionList[T]) Size() int {
	return len(l.list)
}

type safeInsertionList[T comparable] struct {
	list InsertionList[T]
	lock sync.RWMutex
}

func NewSafeInsertionList[T comparable](comparator func(l T, r T) int) InsertionList[T] {
	return &safeInsertionList[T]{
		list: NewInsertionList(comparator),
		lock: *new(sync.RWMutex),
	}
}

func (l *safeInsertionList[T]) withRead(cb func()) {
	l.lock.RLock()
	defer l.lock.RUnlock()
	cb()
}

func (l *safeInsertionList[T]) withWrite(cb func()) {
	l.lock.Lock()
	defer l.lock.Unlock()
	cb()
}

func (l *safeInsertionList[T]) Get(index int) (res T) {
	l.withRead(func() {
		res = l.list.Get(index)
	})
	return
}

func (l *safeInsertionList[T]) Head() (res T) {
	l.withRead(func() {
		res = l.list.Head()
	})
	return
}

func (l *safeInsertionList[T]) Tail() (res T) {
	l.withRead(func() {
		res = l.list.Tail()
	})
	return
}

func (l *safeInsertionList[T]) Filter(f func(T) bool) (res InsertionList[T]) {
	l.withRead(func() {
		res = l.list.Filter(f)
	})
	return
}

func (l *safeInsertionList[T]) Add(item T) {
	l.withWrite(func() {
		l.list.Add(item)
	})
}

func (l *safeInsertionList[T]) Remove(item T) (res bool) {
	l.withWrite(func() {
		res = l.list.Remove(item)
	})
	return
}

func (l *safeInsertionList[T]) RemoveAt(index int) (res bool) {
	l.withWrite(func() {
		res = l.list.RemoveAt(index)
	})
	return
}

func (l *safeInsertionList[T]) Find(item T) (res int) {
	l.withRead(func() {
		res = l.list.Find(item)
	})
	return
}

func (l *safeInsertionList[T]) Has(item T) (res bool) {
	l.withRead(func() {
		res = l.list.Has(item)
	})
	return
}

func (l *safeInsertionList[T]) AsSlice() (res []T) {
	l.withRead(func() {
		res = l.list.AsSlice()
	})
	return
}

func (l *safeInsertionList[T]) Size() (res int) {
	l.withRead(func() {
		res = l.list.Size()
	})
	return
}

func (l *safeInsertionList[T]) Clear() {
	l.withWrite(func() {
		l.list.Clear()
	})
}
