package async

import (
	"sync"
	"sync/atomic"
)

type WaitLock struct {
	cond   *sync.Cond
	isOpen atomic.Bool
}

func (b *WaitLock) Open() {
	b.cond.L.Lock()
	if !b.isOpen.Load() {
		b.isOpen.Store(true)
		b.cond.Broadcast()
	}
	b.cond.L.Unlock()
}

func (b *WaitLock) Wait() {
	b.cond.L.Lock()
	for !b.isOpen.Load() {
		b.cond.Wait()
	}
	b.cond.L.Unlock()
}

func (b *WaitLock) IsOpen() bool {
	return b.isOpen.Load()
}

func (b *WaitLock) Lock() {
	b.cond.L.Lock()
	b.isOpen.Store(false)
	b.cond.L.Unlock()
}

func NewWaitLock() *WaitLock {
	b := &WaitLock{
		cond: sync.NewCond(&sync.Mutex{}),
	}
	b.isOpen.Store(false)
	return b
}

func NewOpenWaitLock() *WaitLock {
	wl := NewWaitLock()
	wl.isOpen.Store(true)
	return wl
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
