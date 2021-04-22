package async

import "sync/atomic"

type Barrier struct {
	c chan bool
	isOpen atomic.Value
}

func (b *Barrier) Open() {
	close(b.c)
	b.isOpen.Store(true)
}

func (b *Barrier) Wait() {
	if b.IsOpen() {
		return
	}
	<- b.c
}

func (b *Barrier) IsOpen() bool {
	return b.isOpen.Load().(bool)
}

func NewBarrier() *Barrier {
	return &Barrier{
		make(chan bool),
		atomic.Value{},
	}
}

type StatefulBarrier struct {
	b *Barrier
	state atomic.Value
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

func (s* StatefulBarrier) Get() interface{} {
	s.Wait()
	return s.state
}

func NewStatefulBarrier() *StatefulBarrier {
	return &StatefulBarrier{
		b:     NewBarrier(),
		state: atomic.Value{},
	}
}