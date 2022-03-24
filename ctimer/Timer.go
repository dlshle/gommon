package ctimer

// TODO one goroutine runs timer tick, another goroutine or async pool to run callbacks

import (
	"context"
	"time"
	"github.com/dlshle/gommon/async"
)

type task struct {
	id        string
	cb        func()
	duration  time.Duration
	timeoutAt time.Time
	repeat    bool
}

type Timer interface {
  Timeout(duration time.Duration, cb func()) string
  Interval(duration time.Duration, cb func()) string
  Reset(id string) bool
  Cancel(id string) bool
  Stop()
}

type timer struct {
	asyncPool  async.AsyncPool
	tasks      map[string]task
	isRunning  bool
	ctx        context.Context
	cancelFunc func()
	ticker     *time.Ticker
}

func (t *timer) Timeout(duration time.Duration, callback func()) string {
	panic("implement this")
}

func (t *timer) Interval(duration time.Duration, callback func()) string {
	panic("implement this")
}

func (t *timer) Reset(id string) bool {
	panic("implement this")
}

func (t *timer) Cancel(id string) bool {
	panic("implement this")
}

func (t *timer) Stop() {
	t.cancelFunc()
}

func (t *timer) loop() {
	select {
	case <-t.ticker.C:
		t.tickAction()
	case <-t.ctx.Done():
		break
	}
}

func (t *timer) tickAction() {
	now := time.Now()
	for _, v := range t.tasks {
		if now.After(v.timeoutAt) {
			t.asyncPool.Execute(func() {
				t.executeTask(v)
			})
		}
	}
}

func (t *timer) executeTask(v task) {
	v.cb()
	if v.repeat {
		v.timeoutAt = time.Now().Add(v.duration)
	}
}
