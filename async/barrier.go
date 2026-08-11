package async

import (
	"sync"
	"sync/atomic"
	"time"
)

type WaitLock struct {
	mu     sync.Mutex
	ch     chan struct{}
	isOpen atomic.Bool
}

func (b *WaitLock) Open() {
	b.mu.Lock()
	if !b.isOpen.Load() {
		b.isOpen.Store(true)
		close(b.ch)
	}
	b.mu.Unlock()
}

func (b *WaitLock) Wait() {
	for {
		if b.isOpen.Load() {
			return
		}
		b.mu.Lock()
		ch := b.ch
		b.mu.Unlock()
		<-ch
	}
}

// WaitWithTimeout waits for the lock to be opened or for the timeout to expire.
// It returns true if the lock was opened, false if the timeout expired.
func (b *WaitLock) WaitWithTimeout(duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	for {
		if b.isOpen.Load() {
			return true
		}
		b.mu.Lock()
		ch := b.ch
		b.mu.Unlock()
		select {
		case <-ch:
			// continue loop to re-check isOpen and refresh ch if needed
		case <-timer.C:
			return false
		}
	}
}

func (b *WaitLock) IsOpen() bool {
	return b.isOpen.Load()
}

func (b *WaitLock) Lock() {
	b.mu.Lock()
	if b.isOpen.CompareAndSwap(true, false) {
		b.ch = make(chan struct{})
	}
	b.mu.Unlock()
}

func NewWaitLock() *WaitLock {
	return &WaitLock{
		ch: make(chan struct{}),
	}
}

func NewOpenWaitLock() *WaitLock {
	b := &WaitLock{
		ch: make(chan struct{}),
	}
	b.isOpen.Store(true)
	close(b.ch)
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
