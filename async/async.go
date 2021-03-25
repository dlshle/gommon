package async

import (
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"
)

type AsyncError struct {
	msg string
}

func (e *AsyncError) Error() string {
	return e.msg
}

func NewAsyncError(msg string) error {
	return &AsyncError{msg}
}

type Promise struct {
	cond     *sync.Cond
	isClosed atomic.Value
}

type IPromise interface {
	IsClosed() bool
	Wait()
	resolve() error
}

func NewPromise() *Promise {
	var value atomic.Value
	value.Store(false)
	return &Promise{sync.NewCond(&sync.Mutex{}), value}
}

func (p *Promise) IsClosed() bool {
	return p.isClosed.Load().(bool)
}

func (p *Promise) Wait() {
	if p.IsClosed() {
		return
	}
	p.cond.L.Lock()
	p.cond.Wait()
	p.cond.L.Unlock()
}

func (p *Promise) resolve() error {
	if p.IsClosed() {
		return NewAsyncError("Promise has already been resolved.")
	}
	p.cond.L.Lock()
	p.cond.Broadcast()
	p.isClosed.Store(true)
	p.cond.L.Unlock()
	return nil
}

type Future struct {
	*Promise
	value interface{}
}

type IFuture interface {
	Get() interface{}
	Wait()
	resolve(value interface{}) error
}

func NewFuture() *Future {
	return &Future{NewPromise(), nil}
}

func (f *Future) Get() interface{} {
	f.Wait()
	return f.value
}

func (f *Future) Wait() {
	f.Promise.Wait()
}

func (f *Future) resolve(value interface{}) error {
	if f.Promise.IsClosed() {
		return NewAsyncError("Future has already been resolved.")
	}
	f.Promise.resolve()
	f.value = value
	return nil
}

type AsyncTask func()

type ComputableAsyncTask func() interface{}

const (
	IDLE        = 0
	RUNNING     = 1
	TERMINATING = 2
	TERMINATED  = 3
)

type AsyncPool struct {
	id         string
	rwLock     *sync.RWMutex
	channel    chan AsyncTask
	numWorkers int
	status     int
	logger     *log.Logger
}

type IAsyncPool interface {
	getStatus() int
	setStatus(status int)
	HasStarted() bool
	isRunning() bool
	Start()
	Stop()
	schedule(task AsyncTask)
	Schedule(task AsyncTask) IPromise
	ScheduleComputable(computableTask ComputableAsyncTask) IFuture
}

func NewAsyncPool(id string, maxPoolSize, workerSize int) *AsyncPool {
	return &AsyncPool{
		id,
		new(sync.RWMutex),
		make(chan AsyncTask, getInRangeInt(maxPoolSize, 16, 2048)),
		getInRangeInt(workerSize, 2, 1024),
		0,
		log.New(os.Stdout, fmt.Sprintf("AsyncPool[pool-%s]", id), log.Ldate|log.Ltime|log.Lshortfile),
	}
}

func (p *AsyncPool) getStatus() int {
	p.rwLock.RLock()
	defer p.rwLock.RUnlock()
	return p.status
}

func (p *AsyncPool) setStatus(status int) {
	p.rwLock.Lock()
	defer p.rwLock.Unlock()
	if status > -1 && status < 4 {
		p.status = status
		p.logger.Printf("Pool status has transited to %d\n", status)
	}
	return
}

func (p *AsyncPool) HasStarted() bool {
	p.rwLock.RLock()
	defer p.rwLock.RUnlock()
	return p.status > IDLE
}

func (p *AsyncPool) isRunning() bool {
	p.rwLock.RLock()
	defer p.rwLock.RUnlock()
	return p.status == RUNNING
}

func (p *AsyncPool) Start() {
	if p.getStatus() > IDLE {
		return
	}
	go func() {
		// worker manager routine
		var wg sync.WaitGroup
		for i := 0; i < p.numWorkers; i++ {
			wg.Add(1)
			go func(wi int) {
				// worker routine
				for p.isRunning() {
					// simply take task and work on it sequentially
					task, isOpen := <-p.channel
					if isOpen {
						p.logger.Printf("Worker %d has acquired task %p\n", wi, task)
						task()
					}
				}
				p.logger.Printf("Worker %d terminated\n", wi)
				wg.Done()
			}(i)
		}
		// wait till all workers terminated
		wg.Wait()
		p.setStatus(TERMINATED)
		p.logger.Printf("All worker has been terminated\n")
	}()
	p.setStatus(RUNNING)
}

func (p *AsyncPool) Stop() {
	if !p.HasStarted() {
		p.logger.Printf("Warn pool has not started\n")
		return
	}
	close(p.channel)
	p.setStatus(TERMINATING)
	for p.getStatus() != TERMINATED {
	}
}

func (p *AsyncPool) schedule(task AsyncTask) {
	if !p.HasStarted() {
		p.Start()
	}
	p.channel <- task
	p.logger.Printf("Task %p has been scheduled\n", task)
}

// will block on channel buffer size exceeded
func (p *AsyncPool) Schedule(task AsyncTask) IPromise {
	promise := NewPromise()
	p.schedule(func() {
		task()
		promise.resolve()
	})
	return promise
}

// will block on channel buffer size exceeded
func (p *AsyncPool) ScheduleComputable(computableTask ComputableAsyncTask) IFuture {
	future := NewFuture()
	p.schedule(func() {
		future.resolve(computableTask())
	})
	return future
}

// utils
func getInRangeInt(value, min, max int) int {
	if value < min {
		return min
	} else if value > max {
		return max
	} else {
		return value
	}
}
