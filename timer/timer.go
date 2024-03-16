package timer

import (
	"sync"
	"time"
)

type Timer interface {
}

type timer struct {
	fn       func()
	duration time.Duration
	timer    *time.Timer
	wg       sync.WaitGroup
}

// this is a more reliable timer
func New(duration time.Duration, fn func()) *timer {
	var wg sync.WaitGroup
	return &timer{
		duration: duration,
		fn:       fn,
		wg:       wg,
	}
}

func (t *timer) Start() bool {
	if t.timer != nil {
		// already started
		return false
	}
	t.wg.Add(1)
	t.timer = time.AfterFunc(t.duration, func() {
		t.run(false)
	})
	return true
}

func (t *timer) Repeat() bool {
	if t.timer != nil {
		// already started
		return false
	}
	t.wg.Add(1)
	t.timer = time.AfterFunc(t.duration, func() {
		t.run(true)
	})
	return true
}

func (t *timer) Wait() {
	if t.timer == nil {
		return
	}
	t.wg.Wait()
}

func (t *timer) run(repeat bool) {
	t.fn()
	t.wg.Done()
	if repeat {
		t.wg.Add(1)
		t.timer.Reset(t.duration)
	}
}

func (t *timer) Stop() bool {
	if t.timer == nil {
		return false
	}
	stopped := t.timer.Stop()
	if stopped {
		t.timer = nil
		t.wg.Done()
		return true
	}
	return false
}

func (t *timer) Reset() bool {
	if t.timer == nil {
		return false
	}
	return t.timer.Reset(t.duration)
}
