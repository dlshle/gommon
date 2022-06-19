package data_structures

import "sync"

// quick add/removal of comparable

type InsertionList interface {
	Get(index int) IComparable
	Add(item IComparable)
	Remove(item IComparable) bool
	RemoveAt(index int) bool
	Find(item IComparable) int
	Has(item IComparable) bool
	AsSlice() []IComparable
	Size() int
	Clear()
}

type insertionList struct {
	list []IComparable
}

func NewInsertionList() InsertionList {
	return &insertionList{
		list: []IComparable{},
	}
}

func (l *insertionList) Get(index int) IComparable {
	if index < 0 || index > len(l.list) {
		return nil
	}
	return l.list[index]
}

func (l *insertionList) Add(item IComparable) {
	if len(l.list) == 0 {
		l.list = append(l.list, item)
		return
	}
	// find the first one that's >= item
	left, right := 0, len(l.list)
	for left < right {
		m := (left + right) / 2
		if l.list[m].Compare(item) >= 0 {
			right = m
		} else {
			left = m + 1
		}
	}
	if right == len(l.list) {
		l.list = append(l.list, item)
		return
	}
	l.list = append(l.list[:right], append([]IComparable{item}, l.list[right:]...)...)
}

func (l *insertionList) Remove(item IComparable) bool {
	index := l.Find(item)
	if index == -1 {
		return false
	}
	return l.RemoveAt(index)
}

func (l *insertionList) RemoveAt(index int) bool {
	if index < 0 || index > len(l.list) {
		return false
	}
	l.list = append(l.list[:index], l.list[index+1:]...)
	return true
}

func (l *insertionList) Clear() {
	l.list = nil
	l.list = make([]IComparable, 0)
}

func (l *insertionList) Find(item IComparable) int {
	if len(l.list) == 0 {
		return -1
	}
	left, right := 0, len(l.list)-1
	for left <= right {
		m := (left + right) / 2
		if l.list[m].Compare(item) == 0 {
			return m
		}
		if l.list[m].Compare(item) > 0 {
			right = m - 1
		} else {
			left = m + 1
		}
	}
	return -1
}

func (l *insertionList) AsSlice() []IComparable {
	slice := make([]IComparable, len(l.list), len(l.list))
	for i, item := range l.list {
		slice[i] = item
	}
	return slice
}

func (l *insertionList) Has(item IComparable) bool {
	return l.Find(item) > -1
}

func (l *insertionList) Size() int {
	return len(l.list)
}

type SafeInsertionList struct {
	list InsertionList
	lock sync.RWMutex
}

func NewSafeInsertionList() InsertionList {
	return SafeInsertionList{
		list: NewInsertionList(),
		lock: *new(sync.RWMutex),
	}
}

func (l SafeInsertionList) withRead(cb func()) {
	l.lock.RLock()
	defer l.lock.RUnlock()
	cb()
}

func (l SafeInsertionList) withWrite(cb func()) {
	l.lock.Lock()
	defer l.lock.Unlock()
	cb()
}

func (l SafeInsertionList) Get(index int) (res IComparable) {
	l.withRead(func() {
		res = l.list.Get(index)
	})
	return
}

func (l SafeInsertionList) Add(item IComparable) {
	l.withWrite(func() {
		l.list.Add(item)
	})
}

func (l SafeInsertionList) Remove(item IComparable) (res bool) {
	l.withWrite(func() {
		res = l.list.Remove(item)
	})
	return
}

func (l SafeInsertionList) RemoveAt(index int) (res bool) {
	l.withWrite(func() {
		res = l.list.RemoveAt(index)
	})
	return
}

func (l SafeInsertionList) Find(item IComparable) (res int) {
	l.withRead(func() {
		res = l.list.Find(item)
	})
	return
}

func (l SafeInsertionList) Has(item IComparable) (res bool) {
	l.withRead(func() {
		res = l.list.Has(item)
	})
	return
}

func (l SafeInsertionList) AsSlice() (res []IComparable) {
	l.withRead(func() {
		res = l.list.AsSlice()
	})
	return
}

func (l SafeInsertionList) Size() (res int) {
	l.withRead(func() {
		res = l.list.Size()
	})
	return
}

func (l SafeInsertionList) Clear() {
	l.withWrite(func() {
		l.list.Clear()
	})
}
