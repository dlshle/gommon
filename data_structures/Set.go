package data_structures

import "sync"

type ISet interface {
	Add(interface{}) bool
	Delete(interface{}) bool
	GetAll() []interface{}
	Clear()
	Size() int
	ForEach(func(interface{}))
	ForEachWithBreaker(cb func(interface{}, func()))
}

type Set struct {
	m map[interface{}]bool
}

func NewSet() ISet {
	return &Set{make(map[interface{}]bool)}
}

func (s *Set) Add(data interface{}) bool {
	if s.m[data] {
		return false
	}
	s.m[data] = true
	return true
}

func (s *Set) Delete(data interface{}) bool {
	if s.m[data] {
		delete(s.m, data)
		return true
	}
	return false
}

func (s *Set) Clear() {
	for k := range s.m {
		delete(s.m, k)
	}
}

func (s *Set) GetAll() []interface{} {
	var data []interface{}
	for k, _ := range s.m {
		data = append(data, k)
	}
	return data
}

func (s *Set) Size() int {
	return len(s.m)
}

func (s *Set) ForEach(cb func(interface{})) {
	for k := range s.m {
		cb(k)
	}
}

func (s *Set) ForEachWithBreaker(cb func(interface{}, func())) {
	shouldStop := false
	stopper := func() {
		shouldStop = true
	}
	for k := range s.m {
		if shouldStop {
			break
		}
		cb(k, stopper)
	}
}

type SafeSet struct {
	lock *sync.RWMutex
	s    ISet
}

func (s *SafeSet) withRead(cb func()) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	cb()
}

func (s *SafeSet) withWrite(cb func()) {
	s.lock.Lock()
	defer s.lock.Unlock()
	cb()
}

func (s *SafeSet) Add(i interface{}) (exist bool) {
	s.withWrite(func() {
		exist = s.s.Add(i)
	})
	return
}

func (s *SafeSet) Delete(i interface{}) (exist bool) {
	s.withWrite(func() {
		exist = s.s.Delete(i)
	})
	return
}

func (s *SafeSet) Clear() {
	s.withWrite(func() {
		s.s.Clear()
	})
}

func (s *SafeSet) GetAll() (elements []interface{}) {
	s.withRead(func() {
		elements = s.s.GetAll()
	})
	return
}

func (s *SafeSet) Size() (size int) {
	s.withRead(func() {
		size = s.s.Size()
	})
	return
}

func (s *SafeSet) ForEach(cb func(interface{})) {
	s.ForEachWithBreaker(func(ele interface{}, _ func()) {
		cb(ele)
	})
}

func (s *SafeSet) ForEachWithBreaker(cb func(interface{}, func())) {
	// do it this way to avoid writing in the for each loop
	shouldStop := false
	breakFunc := func() {
		shouldStop = true
	}
	elements := s.GetAll()
	for _, ele := range elements {
		if shouldStop {
			return
		}
		cb(ele, breakFunc)
	}
}

func NewSafeSet() ISet {
	return &SafeSet{
		lock: new(sync.RWMutex),
		s:    NewSet(),
	}
}
