package timer

import (
	"time"

	"github.com/dlshle/gommon/async"
)

type Timer interface {
	Start() bool
	Repeat() bool
	Reset() bool
	Wait()
	Stop() bool
}

type timer struct {
	fn       func()
	duration time.Duration
	timer    *time.Timer
	b        async.WaitLock
}

// this is a more reliable timer
func New(duration time.Duration, fn func()) Timer {
	return &timer{
		duration: duration,
		fn:       fn,
		b:        *async.NewOpenWaitLock(),
	}
}

func (t *timer) Start() bool {
	if t.timer != nil {
		// already started
		return false
	}
	t.b.Lock()
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
	t.b.Lock()
	t.timer = time.AfterFunc(t.duration, func() {
		t.run(true)
	})
	return true
}

func (t *timer) Wait() {
	if t.timer == nil {
		return
	}
	t.b.Wait()
}

func (t *timer) run(repeat bool) {
	t.fn()
	t.b.Open()
	if repeat {
		t.b.Lock()
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
		t.b.Open()
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
