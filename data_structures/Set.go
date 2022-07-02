package data_structures

import "sync"

type ImmutableSet[T comparable] interface {
	Container
	GetAll() []T
	ForEach(func(T))
	ForEachWithBreaker(cb func(T, func()))
}

type Set[T comparable] interface {
	Container
	ImmutableSet[T]
	Add(T) bool
	Delete(T) bool
	Clear()
}

type set[T comparable] struct {
	m map[T]bool
}

func NewSet[T comparable]() Set[T] {
	return &set[T]{make(map[T]bool)}
}

func NewSetOf[T comparable](m map[T]bool) Set[T] {
	return &set[T]{m}
}

func (s *set[T]) Add(data T) bool {
	if s.m[data] {
		return false
	}
	s.m[data] = true
	return true
}

func (s *set[T]) Delete(data T) bool {
	if s.m[data] {
		delete(s.m, data)
		return true
	}
	return false
}

func (s *set[T]) Clear() {
	for k := range s.m {
		delete(s.m, k)
	}
}

func (s *set[T]) GetAll() []T {
	var data []T
	for k := range s.m {
		data = append(data, k)
	}
	return data
}

func (s *set[T]) Size() int {
	return len(s.m)
}

func (s *set[T]) ForEach(cb func(T)) {
	for k := range s.m {
		cb(k)
	}
}

func (s *set[T]) ForEachWithBreaker(cb func(T, func())) {
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

type SafeSet[T comparable] struct {
	lock *sync.RWMutex
	s    Set[T]
}

func (s *SafeSet[T]) withRead(cb func()) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	cb()
}

func (s *SafeSet[T]) withWrite(cb func()) {
	s.lock.Lock()
	defer s.lock.Unlock()
	cb()
}

func (s *SafeSet[T]) Add(i T) (exist bool) {
	s.withWrite(func() {
		exist = s.s.Add(i)
	})
	return
}

func (s *SafeSet[T]) Delete(i T) (exist bool) {
	s.withWrite(func() {
		exist = s.s.Delete(i)
	})
	return
}

func (s *SafeSet[T]) Clear() {
	s.withWrite(func() {
		s.s.Clear()
	})
}

func (s *SafeSet[T]) GetAll() (elements []T) {
	s.withRead(func() {
		elements = s.s.GetAll()
	})
	return
}

func (s *SafeSet[T]) Size() (size int) {
	s.withRead(func() {
		size = s.s.Size()
	})
	return
}

func (s *SafeSet[T]) ForEach(cb func(T)) {
	s.ForEachWithBreaker(func(ele T, _ func()) {
		cb(ele)
	})
}

func (s *SafeSet[T]) ForEachWithBreaker(cb func(T, func())) {
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

func NewSafeSet[T comparable]() Set[T] {
	return &SafeSet[T]{
		lock: new(sync.RWMutex),
		s:    NewSet[T](),
	}
}

func NewSafeSetOf[T comparable](m map[T]bool) Set[T] {
	return &SafeSet[T]{
		lock: new(sync.RWMutex),
		s:    NewSetOf(m),
	}
}

func ImmutableSetOf[T comparable](data ...T) ImmutableSet[T] {
	m := make(map[T]bool)
	for _, d := range data {
		m[d] = true
	}
	return NewSetOf(m)
}
