package async

import (
	"context"
	"fmt"
	"os"
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

const (
	IDLE        = 0
	RUNNING     = 1
	TERMINATING = 2
	TERMINATED  = 3
)

var statusStringMap = map[int32]string{
	IDLE:        "IDLE",
	RUNNING:     "RUNNING",
	TERMINATING: "TERMINATING",
	TERMINATED:  "TERMINATED",
}

type AsyncPoolOptions struct {
	MaxOutPolicy uint8
	PanicHandler func(interface{})
}

type AsyncPoolOpt func(*AsyncPoolOptions) *AsyncPoolOptions

func WithMaxOutPolicy(policy uint8) AsyncPoolOpt {
	return func(opts *AsyncPoolOptions) *AsyncPoolOptions {
		opts.MaxOutPolicy = policy
		return opts
	}
}

func WithPanicHandler(handler func(interface{})) AsyncPoolOpt {
	return func(opts *AsyncPoolOptions) *AsyncPoolOptions {
		opts.PanicHandler = handler
		return opts
	}
}

type asyncPool struct {
	id                    string
	ctx                   context.Context
	cancelFunc            func()
	stopWaitGroup         sync.WaitGroup
	lifecycleMu           sync.RWMutex
	tasks                 *taskQueue
	numMaxWorkers         int32
	numRunningWorkers     int32
	status                int32
	logger                logging.Logger
	maxPoolSize           int
	maxOutPolicy          uint8
	numWorkerInstantiated int32
	onPanicHandler        atomic.Value // func(interface{})
}

type AsyncPool interface {
	HasStarted() bool
	Stop()
	Execute(task AsyncTask)
	Schedule(task AsyncTask) Waitable
	ScheduleComputable(computableTask ComputableAsyncTask) WaitGettable
	NumMaxWorkers() int
	NumStartedWorkers() int
	NumPendingTasks() int
	Status() string
	IncreaseWorkerSizeTo(size int) bool
	SetPanicHandler(func(interface{})) AsyncPool
	NumGoroutineInitiated() int32
}

func NewPool(maxPoolSize, workerSize int) AsyncPool {
	return NewAsyncPool("default-"+utils.RandomStringWithSize(5), maxPoolSize, workerSize)
}

func NewPoolWithOptions(maxPoolSize, workerSize int, opts ...AsyncPoolOpt) AsyncPool {
	return NewAsyncPoolWithOptions("default-"+utils.RandomStringWithSize(5), maxPoolSize, workerSize, opts...)
}

func NewPoolCtx(ctx context.Context, maxPoolSize, workerSize int) AsyncPool {
	return NewAsyncPoolCtx(ctx, "default-"+utils.RandomStringWithSize(5), maxPoolSize, workerSize)
}

func NewPoolCtxWithOptions(ctx context.Context, maxPoolSize, workerSize int, opts ...AsyncPoolOpt) AsyncPool {
	return NewAsyncPoolCtxWithOptions(ctx, "default-"+utils.RandomStringWithSize(5), maxPoolSize, workerSize, opts...)
}

func NewAsyncPool(id string, maxPoolSize, workerSize int) AsyncPool {
	return NewAsyncPoolCtx(context.Background(), id, maxPoolSize, workerSize)
}

func NewAsyncPoolWithOptions(id string, maxPoolSize, workerSize int, opts ...AsyncPoolOpt) AsyncPool {
	return NewAsyncPoolCtxWithOptions(context.Background(), id, maxPoolSize, workerSize, opts...)
}

func NewAsyncPoolCtx(ctx context.Context, id string, maxPoolSize, workerSize int) AsyncPool {
	return newAsyncPool(ctx, id, maxPoolSize, workerSize, nil)
}

func NewAsyncPoolCtxWithOptions(ctx context.Context, id string, maxPoolSize, workerSize int, opts ...AsyncPoolOpt) AsyncPool {
	cfg := &AsyncPoolOptions{
		MaxOutPolicy: MaxOutPolicyWait,
	}
	for _, opt := range opts {
		cfg = opt(cfg)
	}
	return newAsyncPool(ctx, id, maxPoolSize, workerSize, cfg)
}

func newAsyncPool(ctx context.Context, id string, maxPoolSize, maxWorkerSize int, opts *AsyncPoolOptions) AsyncPool {
	if opts == nil {
		opts = &AsyncPoolOptions{
			MaxOutPolicy: MaxOutPolicyWait,
		}
	}
	ctx, cancel := context.WithCancel(ctx)
	pool := &asyncPool{
		id:            id,
		ctx:           ctx,
		cancelFunc:    cancel,
		stopWaitGroup: sync.WaitGroup{},
		tasks:         newTaskQueue(),
		numMaxWorkers: int32(getInRangeInt(maxWorkerSize, 1, cpuCount*1024)),
		logger:        logging.CreateDefaultLogger(logging.NewConsoleLogWriter(os.Stdout), "[AsyncPool"+id+"]", logging.ERROR),
		maxPoolSize:   maxPoolSize,
		maxOutPolicy:  opts.MaxOutPolicy,
	}
	if opts.PanicHandler != nil {
		pool.onPanicHandler.Store(opts.PanicHandler)
	}
	return pool
}

func NewSerialPool(id string, maxPoolSize int) AsyncPool {
	return NewAsyncPool(id, maxPoolSize, 1)
}

func NewSerialPoolWithOptions(id string, maxPoolSize int, opts ...AsyncPoolOpt) AsyncPool {
	return NewAsyncPoolWithOptions(id, maxPoolSize, 1, opts...)
}

func NewPoolByFactorOfCPUSpec(id string, poolSizeFactor, workerSizeFactor int) AsyncPool {
	return NewAsyncPool(id, cpuCount*poolSizeFactor, cpuCount*workerSizeFactor)
}

func NewPoolByFactorOfCPUSpecWithOptions(id string, poolSizeFactor, workerSizeFactor int, opts ...AsyncPoolOpt) AsyncPool {
	return NewAsyncPoolWithOptions(id, cpuCount*poolSizeFactor, cpuCount*workerSizeFactor, opts...)
}

func (p *asyncPool) getStatus() int32 {
	return atomic.LoadInt32(&p.status)
}

func (p *asyncPool) setStatus(status int32) {
	if status >= 0 && status < 4 {
		old := atomic.SwapInt32(&p.status, status)
		if old != status {
			p.logger.Info(p.ctx, "Pool status has transitioned to "+statusStringMap[status])
		}
	}
}

func (p *asyncPool) HasStarted() bool {
	return p.getStatus() > IDLE
}

func (p *asyncPool) runWorker(index int32) {
	atomic.AddInt32(&p.numWorkerInstantiated, 1)
	for p.ctx.Err() == nil {
		task := p.tasks.getTask()
		if task == nil {
			break
		}
		task()
	}
	p.decrementNumStartedWorkers()
	p.stopWaitGroup.Done()
}

func (p *asyncPool) tryAddAndRunWorker() {
	if p.getStatus() > RUNNING {
		p.logger.Warn(p.ctx, "status is terminating or terminated, can not add new worker")
		return
	}
	if p.NumPendingTasks() == 0 {
		return
	}
	for {
		started := atomic.LoadInt32(&p.numRunningWorkers)
		if started >= atomic.LoadInt32(&p.numMaxWorkers) {
			return
		}
		if atomic.CompareAndSwapInt32(&p.numRunningWorkers, started, started+1) {
			p.stopWaitGroup.Add(1)
			go p.runWorker(started + 1)
			return
		}
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
	p.lifecycleMu.Lock()
	defer p.lifecycleMu.Unlock()
	if !p.HasStarted() {
		p.logger.Warn(p.ctx, "pool has not started")
		return
	}
	p.cancelFunc()
	p.setStatus(TERMINATING)
	p.stopWaitGroup.Wait()
	p.setStatus(TERMINATED)
}

func (p *asyncPool) schedule(task AsyncTask) {
	p.lifecycleMu.RLock()
	defer p.lifecycleMu.RUnlock()
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
		p.tasks.addTask(task)
		p.addAndRunWorker()
	}
}

func (p *asyncPool) Execute(task AsyncTask) {
	p.schedule(func() {
		p.safeRunVoid(task)
	})
}

func (p *asyncPool) Schedule(task AsyncTask) Waitable {
	promise := NewWaitLock()
	p.schedule(func() {
		p.safeRunVoid(task)
		promise.Open()
	})
	return promise
}

func (p *asyncPool) ScheduleComputable(computableTask ComputableAsyncTask) WaitGettable {
	statefulBarrier := NewStatefulBarrier()
	p.schedule(func() {
		statefulBarrier.OpenWith(p.safeRunComputed(computableTask))
	})
	return statefulBarrier
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
	if s, ok := statusStringMap[p.getStatus()]; ok {
		return s
	}
	return "UNKNOWN"
}

func (p *asyncPool) IncreaseWorkerSizeTo(size int) bool {
	if size > p.NumMaxWorkers() {
		atomic.StoreInt32(&p.numMaxWorkers, int32(size))
		return true
	}
	return false
}

func (p *asyncPool) NumGoroutineInitiated() int32 {
	return atomic.LoadInt32(&p.numWorkerInstantiated) + 1
}

func (p *asyncPool) SetPanicHandler(handler func(interface{})) AsyncPool {
	if handler == nil {
		p.onPanicHandler.Store(func(interface{}) {})
	} else {
		p.onPanicHandler.Store(handler)
	}
	return p
}

func (p *asyncPool) safeRunVoid(task AsyncTask) {
	defer func() {
		if recovered := recover(); recovered != nil {
			p.logger.Errorf(p.ctx, "task failed due to: %v", recovered)
			if handler := p.onPanicHandler.Load(); handler != nil {
				handler.(func(interface{}))(recovered)
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
	}
	return value
}
