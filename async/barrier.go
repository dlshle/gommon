package async

import (
	"sync"
	"sync/atomic"
)

type Waitable interface {
	Wait()
	IsOpen() bool
}

type Gettable interface {
	Get() interface{}
}

type WaitGettable interface {
	Waitable
	Gettable
}

type WaitLock struct {
	cond   *sync.Cond
	isOpen atomic.Value
}

func (b *WaitLock) Open() {
	if !b.IsOpen() {
		b.isOpen.Store(true)
		b.cond.Broadcast()
	}
}

func (b *WaitLock) Wait() {
	if b.IsOpen() {
		return
	}
	b.cond.L.Lock()
	b.cond.Wait()
	b.cond.L.Unlock()
}

func (b *WaitLock) IsOpen() bool {
	return b.isOpen.Load().(bool)
}

func (b *WaitLock) Lock() {
	if b.IsOpen() {
		b.isOpen.Store(false)
	}
}

func NewWaitLock() *WaitLock {
	b := &WaitLock{
		sync.NewCond(&sync.Mutex{}),
		atomic.Value{},
	}
	b.isOpen.Store(false)
	return b
}

type StatefulBarrier struct {
	b     *WaitLock
	state atomic.Value
}

func (s *StatefulBarrier) IsOpen() bool {
	return s.b.IsOpen()
}

func (s *StatefulBarrier) OpenWith(state interface{}) {
	if s.b.IsOpen() {
		return
	}
	s.state.Store(state)
	s.b.Open()
}

func (s *StatefulBarrier) Wait() {
	s.b.Wait()
}

func (s *StatefulBarrier) Get() interface{} {
	s.Wait()
	return s.state.Load()
}

func NewStatefulBarrier() *StatefulBarrier {
	return &StatefulBarrier{
		b:     NewWaitLock(),
		state: atomic.Value{},
	}
}
