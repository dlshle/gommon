package async

import (
	"context"
	"fmt"
	"github.com/dlshle/gommon/logger"
	"github.com/dlshle/gommon/stringz"
	"os"
	"sync"
	"sync/atomic"
)

const (
	MaxOutPolicyWait            = 0 // wait for next available worker
	MaxOutPolicyRunOnNewRoutine = 1 // run on new goroutine
	MaxOutPolicyPanic           = 2 // panic on max pool size exceeded
	MaxOutPolicyDiscard         = 3 // do not run this task
	MaxOutPolicyRunOnCaller     = 4 // run on "this" routine
)

var statusStringMap map[byte]string

func init() {
	statusStringMap = make(map[byte]string)
	statusStringMap[IDLE] = "IDLE"
	statusStringMap[RUNNING] = "RUNNING"
	statusStringMap[TERMINATING] = "TERMINATING"
	statusStringMap[TERMINATED] = "TERMINATED"
}

type AsyncTask func()

type ComputableAsyncTask func() interface{}

const (
	IDLE        = 0
	RUNNING     = 1
	TERMINATING = 2
	TERMINATED  = 3
)

type asyncPool struct {
	id                    string
	context               context.Context
	cancelFunc            func()
	stopWaitGroup         sync.WaitGroup
	rwLock                *sync.RWMutex
	channel               chan AsyncTask
	numMaxWorkers         int32
	numStartedWorkers     int32
	status                byte
	logger                logger.Logger
	maxPoolSize           int
	maxOutPolicy          uint8
	numGoroutineInitiated int32
}

type AsyncPool interface {
	getStatus() byte
	setStatus(status byte)
	HasStarted() bool
	start()
	Stop()
	schedule(task AsyncTask)
	Execute(task AsyncTask)
	Schedule(task AsyncTask) Waitable
	ScheduleComputable(computableTask ComputableAsyncTask) WaitGettable
	Verbose(use bool)
	NumMaxWorkers() int
	NumStartedWorkers() int
	NumPendingTasks() int
	Status() string
	IncreaseWorkerSizeTo(size int) bool
	SetMaxOutPolicy(policy uint8)
	NumGoroutineInitiated() int32
}

func NewAsyncPool(id string, maxPoolSize, workerSize int) AsyncPool {
	ctx, cancel := context.WithCancel(context.Background())
	return &asyncPool{
		id,
		ctx,
		cancel,
		sync.WaitGroup{},
		new(sync.RWMutex),
		make(chan AsyncTask, getInRangeInt(maxPoolSize, 16, 2048)),
		int32(getInRangeInt(workerSize, 2, 1024)),
		0,
		0,
		// logger.New(os.Stdout, fmt.Sprintf("asyncPool[%s]", id), false),
		logger.CreateLevelLogger(logger.NewNoopWriter(), "[AsyncPool"+id+"]", -1),
		maxPoolSize,
		MaxOutPolicyWait,
		0,
	}
}

func (p *asyncPool) withWrite(cb func()) {
	p.rwLock.Lock()
	defer p.rwLock.Unlock()
	cb()
}

func (p *asyncPool) getStatus() byte {
	p.rwLock.RLock()
	defer p.rwLock.RUnlock()
	return p.status
}

func (p *asyncPool) setStatus(status byte) {
	p.rwLock.Lock()
	defer p.rwLock.Unlock()
	if status >= 0 && status < 4 {
		p.status = status
		p.logger.Info("Pool status has transitioned to " + statusStringMap[status])
	}
	return
}

func (p *asyncPool) HasStarted() bool {
	p.rwLock.RLock()
	defer p.rwLock.RUnlock()
	return p.status > IDLE
}

func (p *asyncPool) runWorker(index int32) {
	atomic.AddInt32(&p.numGoroutineInitiated, 1)
	// worker routine
	shouldContinue := true
	for shouldContinue {
		select {
		case task, isOpen := <-p.channel:
			// simply take task and work on it sequentially
			if isOpen {
				p.logger.Info(stringz.Builder().
					String("Worker ").
					Int32(index).
					String(" has acquired task ").
					Pointer(task).
					BuildL())
				task()
			} else {
				shouldContinue = false
				break
			}
			if p.NumPendingTasks() == 0 {
				shouldContinue = false
			}
		case <-p.context.Done():
			shouldContinue = false
		}
	}
	p.decrementNumStartedWorkers()
	p.logger.Info(stringz.Builder().String("Worker ").Int32(index).String(" terminated").BuildL())
	p.stopWaitGroup.Done()
}

func (p *asyncPool) tryAddAndRunWorker() {
	if p.getStatus() > RUNNING {
		p.logger.Info("status is terminating or terminated, can not add new worker")
		return
	}
	if p.NumPendingTasks() > 0 && p.NumStartedWorkers() < p.NumMaxWorkers() {
		p.addAndRunWorker()
	}
}

func (p *asyncPool) addAndRunWorker() {
	// need to increment the waitGroup before worker goroutine runs
	p.stopWaitGroup.Add(1)
	// of course worker runs on its own goroutine
	go p.runWorker(p.incrementAndGetNumStartedWorkers())
	p.logger.Infof("worker %d has been started", p.numStartedWorkers)
}

func (p *asyncPool) start() {
	if p.getStatus() > IDLE {
		return
	}
	p.setStatus(RUNNING)
}

func (p *asyncPool) Stop() {
	if !p.HasStarted() {
		p.logger.Info("Warn: pool has not started")
		return
	}
	close(p.channel)
	p.cancelFunc()
	p.setStatus(TERMINATING)
	p.stopWaitGroup.Wait()
}

func (p *asyncPool) schedule(task AsyncTask) {
	status := p.getStatus()
	switch {
	case status == IDLE:
		p.start()
	case status > RUNNING:
		return
	}
	if p.isPoolSizeExceeded() && p.maxOutPolicy != MaxOutPolicyWait {
		p.handlePoolSizeExceeded(task)
		return
	}
	p.channel <- task
	p.logger.Infof("Task %p has been scheduled", task)
	p.tryAddAndRunWorker()
}

func (p *asyncPool) handlePoolSizeExceeded(task AsyncTask) {
	switch p.maxOutPolicy {
	case MaxOutPolicyRunOnNewRoutine:
		go task()
		break
	case MaxOutPolicyPanic:
		panic(fmt.Sprintf("max pool size(%d) exceeded", p.maxPoolSize))
	case MaxOutPolicyDiscard:
		p.logger.Infof("task %p is discarded", task)
		return
	case MaxOutPolicyRunOnCaller:
		task()
		return
	default:
		p.channel <- task
		break
	}
}

func (p *asyncPool) Execute(task AsyncTask) {
	p.schedule(task)
}

// will block on channel buffer size exceeded
func (p *asyncPool) Schedule(task AsyncTask) Waitable {
	promise := NewWaitLock()
	p.schedule(func() {
		task()
		promise.Open()
	})
	return promise
}

// will block on channel buffer size exceeded
func (p *asyncPool) ScheduleComputable(computableTask ComputableAsyncTask) WaitGettable {
	statefulBarrier := NewStatefulBarrier()
	p.schedule(func() {
		statefulBarrier.OpenWith(computableTask())
	})
	return statefulBarrier
}

func (p *asyncPool) Verbose(use bool) {
	if use {
		p.logger.Writer(logger.NewConsoleLogWriter(os.Stdout))
	} else {
		p.logger.Writer(logger.NewNoopWriter())
	}
}

func (p *asyncPool) NumMaxWorkers() int {
	return int(atomic.LoadInt32(&p.numMaxWorkers))
}

func (p *asyncPool) NumPendingTasks() int {
	if p.getStatus() == RUNNING {
		return len(p.channel)
	}
	return 0
}

func (p *asyncPool) isPoolSizeExceeded() bool {
	return p.NumPendingTasks() >= p.maxPoolSize
}

func (p *asyncPool) NumStartedWorkers() int {
	return int(atomic.LoadInt32(&p.numStartedWorkers))
}

func (p *asyncPool) incrementAndGetNumStartedWorkers() int32 {
	return atomic.AddInt32(&p.numStartedWorkers, 1)
}

func (p *asyncPool) decrementNumStartedWorkers() {
	atomic.AddInt32(&p.numStartedWorkers, -1)
}

func (p *asyncPool) Status() string {
	return statusStringMap[p.getStatus()]
}

func (p *asyncPool) IncreaseWorkerSizeTo(size int) bool {
	if size > p.NumMaxWorkers() {
		atomic.StoreInt32(&p.numMaxWorkers, int32(size))
		return true
	}
	return false
}

func (p *asyncPool) SetMaxOutPolicy(policy uint8) {
	if policy >= MaxOutPolicyWait && policy <= MaxOutPolicyRunOnCaller {
		p.maxOutPolicy = policy
	}
}

func (p *asyncPool) NumGoroutineInitiated() int32 {
	return atomic.LoadInt32(&p.numGoroutineInitiated)
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
