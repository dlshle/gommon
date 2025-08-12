package async

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/dlshle/gommon/logging"
	"github.com/dlshle/gommon/utils"
)

const (
	MaxOutPolicyWait            = 0 // wait for next available worker
	MaxOutPolicyRunOnNewRoutine = 1 // run on new goroutine
	MaxOutPolicyPanic           = 2 // panic on max pool size exceeded
	MaxOutPolicyDiscard         = 3 // do not run this task
	MaxOutPolicyRunOnCaller     = 4 // run on "this" routine
)

var cpuCount = runtime.NumCPU()

var statusStringMap map[int32]string

func init() {
	statusStringMap = make(map[int32]string)
	statusStringMap[IDLE] = "IDLE"
	statusStringMap[RUNNING] = "RUNNING"
	statusStringMap[TERMINATING] = "TERMINATING"
	statusStringMap[TERMINATED] = "TERMINATED"
}

const (
	IDLE        = 0
	RUNNING     = 1
	TERMINATING = 2
	TERMINATED  = 3
)

type asyncPool struct {
	id                    string
	ctx                   context.Context
	cancelFunc            func()
	stopWaitGroup         sync.WaitGroup
	tasks                 *taskQueue
	numMaxWorkers         int32
	numRunningWorkers     int32
	status                int32
	logger                logging.Logger
	maxPoolSize           int
	maxOutPolicy          uint8
	numWorkerInstantiated int32
	onPanicHandler        func(interface{})
}

type AsyncPool interface {
	HasStarted() bool
	Stop()
	Execute(task AsyncTask)
	Schedule(task AsyncTask) Waitable
	ScheduleComputable(computableTask ComputableAsyncTask) WaitGettable
	Verbose(use bool)
	NumMaxWorkers() int
	NumStartedWorkers() int
	NumPendingTasks() int
	Status() string
	IncreaseWorkerSizeTo(size int) bool
	SetMaxOutPolicy(policy uint8) AsyncPool
	SetPanicHandler(func(interface{})) AsyncPool
	NumGoroutineInitiated() int32
}

func NewPool(maxPoolSize, workerSize int) AsyncPool {
	return NewAsyncPool("default-"+utils.RandomStringWithSize(5), maxPoolSize, workerSize)
}

func NewPoolCtx(ctx context.Context, maxPoolSize, workerSize int) AsyncPool {
	return NewAsyncPoolCtx(ctx, "default-"+utils.RandomStringWithSize(5), maxPoolSize, workerSize)
}

func NewAsyncPool(id string, maxPoolSize, workerSize int) AsyncPool {
	return NewAsyncPoolCtx(context.Background(), id, maxPoolSize, workerSize)
}

func NewAsyncPoolCtx(ctx context.Context, id string, maxPoolSize, workerSize int) AsyncPool {
	return newAsyncPool(ctx, id, maxPoolSize, workerSize)
}

func newAsyncPool(ctx context.Context, id string, maxPoolSize, maxWorkerSize int) AsyncPool {
	ctx, cancel := context.WithCancel(ctx)
	return &asyncPool{
		id,
		ctx,
		cancel,
		sync.WaitGroup{},
		newTaskQueue(),
		int32(getInRangeInt(maxWorkerSize, 2, cpuCount*1024)),
		0,
		0,
		logging.GlobalLogger.WithPrefix("[AsyncPool" + id + "]").WithWaterMark(logging.ERROR),
		maxPoolSize,
		MaxOutPolicyWait,
		0,
		nil,
	}
}

func NewSerialPool(id string, maxPoolSize int) AsyncPool {
	return NewAsyncPool(id, maxPoolSize, 1)
}

func NewPoolByFactorOfCPUSpec(id string, poolSizeFactor, workerSizeFactor int) AsyncPool {
	return NewAsyncPool(id, cpuCount*poolSizeFactor, cpuCount*workerSizeFactor)
}

func (p *asyncPool) getStatus() int32 {
	return atomic.LoadInt32(&p.status)
}

func (p *asyncPool) setStatus(status int32) {
	if status >= 0 && status < 4 {
		atomic.StoreInt32(&p.status, status)
		p.logger.Info(p.ctx, "Pool status has transitioned to "+statusStringMap[status])
	}
}

func (p *asyncPool) HasStarted() bool {
	return p.getStatus() > IDLE
}

func (p *asyncPool) runWorker(index int32) {
	atomic.AddInt32(&p.numWorkerInstantiated, 1)
	// worker routine
	shouldContinue := true
	for shouldContinue {
		select {
		case <-p.ctx.Done():
			shouldContinue = false
		default:
			task := p.tasks.getTask()
			// simply take task and work on it sequentially
			if task != nil {
				task()
			} else {
				shouldContinue = false
				break
			}
			if p.NumPendingTasks() == 0 {
				shouldContinue = false
			}
		}
	}
	p.decrementNumStartedWorkers()
	p.stopWaitGroup.Done()
}

func (p *asyncPool) tryAddAndRunWorker() {
	if p.getStatus() > RUNNING {
		p.logger.Warn(p.ctx, "status is terminating or terminated, can not add new worker")
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
}

func (p *asyncPool) start() {
	if p.getStatus() > IDLE {
		return
	}
	p.setStatus(RUNNING)
}

func (p *asyncPool) Stop() {
	if !p.HasStarted() {
		p.logger.Warn(p.ctx, "pool has not started")
		return
	}
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
		panic("pool has already been stopped, unable to run further tasks")
	}
	if p.isPoolSizeExceeded() && p.maxOutPolicy != MaxOutPolicyWait {
		p.handlePoolSizeExceeded(task)
		return
	}
	p.tasks.addTask(task)
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
		p.logger.Warnf(p.ctx, "task %p is discarded", task)
		return
	case MaxOutPolicyRunOnCaller:
		task()
		return
	default:
		// by default, add a new worker temporarily to handle the extra tasks
		p.addAndRunWorker()
		p.tasks.addTask(task)
		break
	}
}

func (p *asyncPool) Execute(task AsyncTask) {
	p.schedule(func() {
		p.safeRunVoid(task)
	})
}

// will block on channel buffer size exceeded
func (p *asyncPool) Schedule(task AsyncTask) Waitable {
	promise := NewWaitLock()
	p.schedule(func() {
		p.safeRunVoid(task)
		promise.Open()
	})
	return promise
}

// will block on channel buffer size exceeded
func (p *asyncPool) ScheduleComputable(computableTask ComputableAsyncTask) WaitGettable {
	statefulBarrier := NewStatefulBarrier()
	p.schedule(func() {
		statefulBarrier.OpenWith(p.safeRunComputed(computableTask))
	})
	return statefulBarrier
}

func (p *asyncPool) Verbose(use bool) {
	if use {
		p.logger.SetWaterMark(logging.DEBUG)
	} else {
		p.logger.SetWaterMark(logging.FATAL)
	}
}

func (p *asyncPool) NumMaxWorkers() int {
	return int(atomic.LoadInt32(&p.numMaxWorkers))
}

func (p *asyncPool) NumPendingTasks() int {
	if p.getStatus() == RUNNING {
		return p.tasks.numTasks()
	}
	return 0
}

func (p *asyncPool) isPoolSizeExceeded() bool {
	return p.NumPendingTasks() >= p.maxPoolSize
}

func (p *asyncPool) NumStartedWorkers() int {
	return int(atomic.LoadInt32(&p.numRunningWorkers))
}

func (p *asyncPool) incrementAndGetNumStartedWorkers() int32 {
	return atomic.AddInt32(&p.numRunningWorkers, 1)
}

func (p *asyncPool) decrementNumStartedWorkers() {
	atomic.AddInt32(&p.numRunningWorkers, -1)
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

func (p *asyncPool) SetMaxOutPolicy(policy uint8) AsyncPool {
	if policy >= MaxOutPolicyWait && policy <= MaxOutPolicyRunOnCaller {
		p.maxOutPolicy = policy
	}
	return p
}

func (p *asyncPool) NumGoroutineInitiated() int32 {
	return atomic.LoadInt32(&p.numWorkerInstantiated) + 1
}

func (p *asyncPool) SetPanicHandler(handler func(interface{})) AsyncPool {
	p.onPanicHandler = handler
	return p
}

func (p *asyncPool) safeRunVoid(task AsyncTask) {
	defer func() {
		if recovered := recover(); recovered != nil {
			p.logger.Errorf(p.ctx, "task failed due to: %v", recovered)
			if p.onPanicHandler != nil {
				p.onPanicHandler(recovered)
			}
		}
	}()
	task()
}

func (p *asyncPool) safeRunComputed(computedTask ComputableAsyncTask) interface{} {
	var val interface{}
	p.safeRunVoid(func() {
		val = computedTask()
	})
	return val
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
