package ctimer

import (
	"github.com/dlshle/gommon/async"
	"sync"
	"time"
)

var timerPool *sync.Pool

func init() {
	timerPool = &sync.Pool{
		New: func() interface{} {
			return new(cTimer)
		},
	}
}

const (
	StatusIdle = iota
	StatusWaiting
	StatusReset
	StatusCancelled
	StatusRunning
	StatusRepeatWaiting
	StatusRepeatRunning
)

type CTimer interface {
	Start()
	Reset()
	Cancel()
	Repeat()
	WithAsyncPool(pool async.AsyncPool)
}

type cTimer struct {
	job           func()
	startTime     time.Time
	resetInterval time.Duration
	interval      time.Duration
	status        uint8
	asyncPool     async.AsyncPool
}

func New(interval time.Duration, job func()) CTimer {
	timer := timerPool.Get().(*cTimer)
	timer.job = job
	timer.interval = interval
	timer.status = StatusIdle
	return &cTimer{
		job:      job,
		interval: interval,
		status:   StatusIdle,
	}
}

func (t *cTimer) WithAsyncPool(pool async.AsyncPool) {
	t.asyncPool = pool
}

func (t *cTimer) Start() {
	if t.status == StatusIdle {
		t.runTask(func() {
			t.waitAndRun(t.interval)
		})
	}
}

func (t *cTimer) Repeat() {
	if t.status == StatusIdle {
		t.runTask(func() {
			t.repeatWaitAndRun(t.interval)
		})
	}
}

func (t *cTimer) runTask(task func()) {
	if t.asyncPool != nil {
		t.asyncPool.Execute(task)
	} else {
		go task()
	}
}

func (t *cTimer) repeatWaitAndRun(interval time.Duration) {
	for t.status != StatusCancelled {
		t.waitAndRun(interval)
	}
}

func (t *cTimer) waitAndRun(interval time.Duration) {
	if t.status == StatusCancelled {
		return
	}
	t.resetInterval = 0
	t.startTime = time.Now()
	t.status = StatusWaiting
	time.Sleep(interval)
	if t.status == StatusCancelled {
		t.status = StatusIdle
		return
	}
	if t.status == StatusReset && t.resetInterval > 0 {
		t.waitAndRun(t.resetInterval)
		return
	}
	t.status = StatusRunning
	t.job()
	t.status = StatusIdle
}

func (t *cTimer) Reset() {
	if t.status == StatusWaiting || t.status == StatusReset {
		t.status = StatusReset
		previousTime := t.startTime
		t.startTime = time.Now()
		t.resetInterval = t.resetInterval + t.startTime.Sub(previousTime)
		return
	} else {
		t.Start()
	}
}

func (t *cTimer) Cancel() {
	if t.status == StatusWaiting || t.status == StatusRepeatWaiting || t.status == StatusRunning {
		t.status = StatusCancelled
	}
}
