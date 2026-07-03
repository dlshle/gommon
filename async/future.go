package async

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/dlshle/gommon/errors"
)

type FutureGetter interface {
	Wait()
	Get() (interface{}, error)
	MustGet() interface{} // panic on error
	WaitWithTimeout(duration time.Duration) error
	GetWithTimeout(duration time.Duration) (interface{}, error)
}

type directExecutor uint8

func (e directExecutor) Execute(task AsyncTask) {
	task()
}

type newGoRoutineExecutor uint8

func (e newGoRoutineExecutor) Execute(task AsyncTask) {
	go task()
}

const (
	DirectExecutor       directExecutor       = 0
	NewGoRoutineExecutor newGoRoutineExecutor = 0
)

const (
	CanceledMsg = "future_canceled"
	TimeoutMsg  = "future_timeout"
)

var canceledError error

func init() {
	canceledError = errors.Error(CanceledMsg)
}

type Future interface {
	FutureGetter
	// try to cancel the task before its execution
	Cancel()
	IsDone() bool
	Then(onComplete func(interface{}) (interface{}, error)) Future
	ThenAsync(onComplete func(interface{}) (Future, error)) Future
	ThenAsyncWithExecutor(onComplete func(interface{}) (Future, error), executor Executor) Future
	ThenWithExecutor(onComplete func(interface{}) (interface{}, error), executor Executor) Future
	ThenWithFuture(future Future) Future
	OnPanic(onPanic func(interface{})) Future
	OnError(onError func(error)) Future
	MapError(mappingFn func(error) interface{}) Future
	MapPanic(mappingFn func(interface{}) interface{}) Future
}

type OptionalParamOperation func(interface{}) interface{}

type future struct {
	mu             sync.Mutex
	executor       Executor
	waitLock       *WaitLock
	task           ComputableAsyncTaskWithError
	result         interface{}
	panicEntity    interface{}
	errEntity      error
	isRunning      atomic.Bool
	prevFuture     *future
	nextFutures    []*future
	onPanic        func(interface{})
	propogatePanic bool
}

func newAsyncTaskFuture(task AsyncTask, executor Executor) *future {
	computedTask := func() (interface{}, error) {
		task()
		return nil, nil
	}
	return newFuture(computedTask, executor, nil)
}

func newComputedFuture(task ComputableAsyncTask, executor Executor) *future {
	taskWithMaybeError := func() (interface{}, error) {
		return task(), nil
	}
	return newFuture(taskWithMaybeError, executor, nil)
}

func newComputedWithErrorFuture(task ComputableAsyncTaskWithError, executor Executor) *future {
	return newFuture(task, executor, nil)
}

func newFuture(task ComputableAsyncTaskWithError, executor Executor, prevFuture *future) *future {
	return &future{
		prevFuture:     prevFuture,
		executor:       executor,
		waitLock:       NewWaitLock(),
		task:           task,
		propogatePanic: true,
	}
}

func (f *future) start() *future {
	if f.prevFuture != nil {
		return f.prevFuture.start()
	}
	return f.run()
}

func (f *future) Cancel() {
	f.mu.Lock()
	if f.task == nil || f.isRunning.Load() || f.waitLock.IsOpen() {
		next := append([]*future(nil), f.nextFutures...)
		f.mu.Unlock()
		for _, nf := range next {
			nf.Cancel()
		}
		return
	}
	f.executor = DirectExecutor
	f.task = func() (interface{}, error) {
		panic(canceledError)
	}
	f.mu.Unlock()
}

func (f *future) Wait() {
	f.start()
	f.waitLock.Wait()
}

func (f *future) WaitWithTimeout(duration time.Duration) error {
	f.start()
	if f.waitLock.IsOpen() {
		return nil
	}
	return RaceTimeoutWithOperation(duration, f.Wait)
}

func (f *future) Get() (interface{}, error) {
	f.start()
	f.waitLock.Wait()
	f.mu.Lock()
	panicEntity := f.panicEntity
	result := f.result
	err := f.errEntity
	f.mu.Unlock()
	if panicEntity != nil {
		panic(panicEntity)
	}
	return result, err
}

func (f *future) MustGet() interface{} {
	res, err := f.Get()
	if err != nil {
		panic(err)
	}
	return res
}

func (f *future) GetWithTimeout(duration time.Duration) (result interface{}, err error) {
	if f.waitLock.IsOpen() {
		f.mu.Lock()
		result = f.result
		err = f.errEntity
		panicEntity := f.panicEntity
		f.mu.Unlock()
		if panicEntity != nil {
			panic(panicEntity)
		}
		return
	}
	err = RaceTimeoutWithOperation(duration, func() {
		result, err = f.Get()
	})
	return
}

func (f *future) IsDone() bool {
	return f.waitLock.IsOpen()
}

func (f *future) ThenWithFuture(nextFuture Future) Future {
	return f.then(nextFuture.(*future))
}

func (f *future) ThenWithExecutor(onComplete func(interface{}) (interface{}, error), executor Executor) Future {
	nextTask := f.assembleNextTask(onComplete)
	return f.then(newFuture(nextTask, executor, f))
}

func (f *future) ThenAsync(onComplete func(interface{}) (Future, error)) Future {
	return f.ThenAsyncWithExecutor(onComplete, DirectExecutor)
}

func (f *future) ThenAsyncWithExecutor(onComplete func(interface{}) (Future, error), executor Executor) Future {
	return f.then(newPromisedFuture(func(res ResultAcceptor, rej ErrorAcceptor) {
		input, _ := f.Get()
		nextFuture, err := onComplete(input)
		if err != nil {
			rej(err)
			return
		}

		nextFutureInternal, ok := nextFuture.(*future)
		if !ok {
			// if casting fails, use traditional chainning approach
			nextFuture.Then(func(nextInput interface{}) (interface{}, error) {
				res(nextInput)
				return nil, nil
			}).OnError(func(err error) {
				rej(err)
			})
		} else {
			// otherwise, use internal chainning approach for simplicity and performance
			nextFutureInternal.then(newFuture(func() (interface{}, error) {
				r, e := nextFuture.Get()
				if e != nil {
					rej(e)
				} else {
					res(r)
				}
				return nil, nil
			}, executor, nextFutureInternal))
		}
	}, executor, f, false))
}

func (f *future) Then(onSuccess func(interface{}) (interface{}, error)) Future {
	nextTask := f.assembleNextTask(onSuccess)
	return f.then(newFuture(nextTask, f.executor, f))
}

func (f *future) OnError(onError func(error)) Future {
	return f.then(newFuture(func() (interface{}, error) {
		res, err := f.Get()
		if err != nil {
			onError(err)
		}
		return res, err
	}, DirectExecutor, f))
}

func (f *future) OnPanic(onPanic func(interface{})) Future {
	f.mu.Lock()
	f.onPanic = onPanic
	panicEntity := f.panicEntity
	isDone := f.waitLock.IsOpen()
	f.mu.Unlock()
	if isDone && panicEntity != nil {
		f.handlePanic(panicEntity)
	}
	return f
}

func (f *future) MapError(mappingFn func(error) interface{}) Future {
	return f.then(newFuture(func() (interface{}, error) {
		res, err := f.Get()
		if err != nil {
			return mappingFn(err), nil
		}
		return res, err
	}, DirectExecutor, f))
}

func (f *future) MapPanic(mappingFn func(interface{}) interface{}) Future {
	f.mu.Lock()
	f.propogatePanic = false
	f.mu.Unlock()
	return f.OnPanic(func(recovered interface{}) {
		f.acceptResult(mappingFn(recovered))
	})
}

func (f *future) assembleNextTask(onSuccess func(interface{}) (interface{}, error)) func() (interface{}, error) {
	return func() (interface{}, error) {
		res, err := f.Get()
		if err != nil {
			return nil, err
		}
		return onSuccess(res)
	}
}

func (f *future) then(nextFuture *future) Future {
	f.mu.Lock()
	f.nextFutures = append(f.nextFutures, nextFuture)
	nextFuture.prevFuture = f
	isDone := f.waitLock.IsOpen()
	hasPanic := f.panicEntity != nil
	panicEntity := f.panicEntity
	isRunning := f.isRunning.Load()
	f.mu.Unlock()

	if !isRunning && !isDone {
		f.start()
		return nextFuture
	}
	if isDone {
		if hasPanic {
			nextFuture.handlePanic(panicEntity)
		} else {
			nextFuture.run()
		}
	}
	return nextFuture
}

func (f *future) run() *future {
	f.mu.Lock()
	if f.isRunning.Load() || f.waitLock.IsOpen() {
		f.mu.Unlock()
		return f
	}
	f.isRunning.Store(true)
	executor := f.executor
	f.mu.Unlock()
	executor.Execute(f.execute)
	return f
}

func (f *future) execute() {
	defer func() {
		if recovered := recover(); recovered != nil {
			f.acceptPanic(recovered)
		}
	}()
	f.mu.Lock()
	task := f.task
	f.mu.Unlock()
	if task != nil {
		result, err := task()
		if err != nil {
			f.acceptError(err)
		} else if result != nil {
			f.acceptResult(result)
		}
	}
}

func (f *future) acceptResult(result interface{}) {
	if result == nil {
		return
	}
	f.mu.Lock()
	if f.result != nil {
		f.mu.Unlock()
		return
	}
	f.result = result
	next := append([]*future(nil), f.nextFutures...)
	f.mu.Unlock()
	f.openWaitLockAndStopRunning()
	f.runNextFutures(next)
}

func (f *future) acceptError(err error) {
	if err == nil {
		return
	}
	f.mu.Lock()
	if f.errEntity != nil {
		f.mu.Unlock()
		return
	}
	f.errEntity = err
	next := append([]*future(nil), f.nextFutures...)
	f.mu.Unlock()
	f.openWaitLockAndStopRunning()
	f.runNextFutures(next)
}

func (f *future) acceptPanic(recovered interface{}) {
	if recovered == nil {
		return
	}
	f.mu.Lock()
	if f.panicEntity != nil {
		f.mu.Unlock()
		return
	}
	f.panicEntity = recovered
	f.mu.Unlock()
	f.handlePanic(recovered)
}

func (f *future) handlePanic(recovered interface{}) {
	f.mu.Lock()
	onPanic := f.onPanic
	propagate := f.propogatePanic
	next := append([]*future(nil), f.nextFutures...)
	f.mu.Unlock()
	if onPanic != nil {
		onPanic(recovered)
	}
	f.openWaitLockAndStopRunning()
	if propagate {
		for _, nf := range next {
			if !nf.isRunning.Load() {
				nf.handlePanic(recovered)
			}
		}
	}
}

func (f *future) runNextFutures(next []*future) {
	for _, nf := range next {
		if !nf.isRunning.Load() {
			nf.run()
		}
	}
}

func (f *future) openWaitLockAndStopRunning() {
	f.waitLock.Open()
	f.isRunning.Store(false)
}

// public utility functions

func Run(task AsyncTask, executor Executor) Future {
	f := newAsyncTaskFuture(task, executor)
	return f
}

func NewComputedFuture(task ComputableAsyncTask, executor Executor) Future {
	f := newComputedFuture(task, executor)
	return f
}

func NewComputedErrorReturningFuture(task ComputableAsyncTaskWithError, executor Executor) Future {
	f := newComputedWithErrorFuture(task, executor)
	return f
}

func newPromisedFuture(resolver func(ResultAcceptor, ErrorAcceptor), executor Executor, prevFuture *future, immediateRun bool) *future {
	f := newFuture(nil, executor, prevFuture)
	f.mu.Lock()
	f.task = func() (_ interface{}, _ error) {
		resolver(func(computedResult interface{}) {
			f.acceptResult(computedResult)
		}, func(catchedErr error) {
			f.acceptError(catchedErr)
		})
		return
	}
	f.mu.Unlock()
	// promised future should automatically start on creation
	if immediateRun {
		return f.run()
	}
	return f
}

type ResultAcceptor = func(interface{})
type ErrorAcceptor = func(error)

// From creates a new Future that settles through a callback.
// NOTE: do not run resolve with the direct executor(i.e. run on a different goroutine)
func From(resolver func(ResultAcceptor, ErrorAcceptor)) Future {
	return newPromisedFuture(resolver, DirectExecutor, nil, true)
}

func FromWithExecutor(resolver func(ResultAcceptor, ErrorAcceptor), executor Executor) Future {
	return newPromisedFuture(resolver, executor, nil, true)
}

func IsCanceled(f Future) bool {
	if !f.IsDone() {
		return false
	}
	rawFuture := f.(*future)
	rawFuture.mu.Lock()
	panicEntity := rawFuture.panicEntity
	rawFuture.mu.Unlock()
	if panicEntity == nil {
		return false
	}
	return panicEntity == canceledError
}

func WhenAllCompleted(futures ...Future) Future {
	return whenAllCompleted(futures)
}

func ImmediateFuture(val interface{}) Future {
	return From(func(ra ResultAcceptor, ea ErrorAcceptor) {
		ra(val)
	})
}

func ImmediateErrorFuture(err error) Future {
	return From(func(ra ResultAcceptor, ea ErrorAcceptor) {
		ea(err)
	})
}

func whenAllCompleted(futures []Future) *future {
	return newFuture(func() (interface{}, error) {
		for _, f := range futures {
			f.Wait()
		}
		return nil, nil
	}, DirectExecutor, nil)
}
