package async

import (
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
	executor       Executor
	waitLock       *WaitLock
	task           ComputableAsyncTaskWithError
	result         interface{}
	panicEntity    interface{}
	errEntity      error
	isRunning      *atomic.Value
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
	isRunning := new(atomic.Value)
	isRunning.Store(false)
	f := &future{
		prevFuture:     prevFuture,
		executor:       executor,
		waitLock:       NewWaitLock(),
		task:           task,
		isRunning:      isRunning,
		propogatePanic: true,
	}
	f.isRunning.Store(false)
	return f
}

func (f *future) start() *future {
	if f.prevFuture != nil {
		return f.prevFuture.start()
	}
	return f.run()
}

func (f *future) Cancel() {
	if f.task == nil || f.isRunning.Load().(bool) || f.waitLock.IsOpen() {
		// try to cancel later futures
		f.withNextFutures(func(nextFuture *future) {
			nextFuture.Cancel()
		})
		return
	}
	f.executor = DirectExecutor
	f.task = func() (interface{}, error) {
		panic(canceledError)
	}
}

func (f *future) Wait() {
	// if f.executor == DirectExecutor {
	// return
	// }
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
	if f.panicEntity != nil {
		panic(f.panicEntity)
	}
	return f.result, f.errEntity
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
		return f.result, nil
	}
	err = RaceTimeoutWithOperation(duration, func() {
		result, err = f.Get()
	})
	return
}

func (f *future) IsDone() bool {
	return f.waitLock.IsOpen()
}

func (f *future) ThenWithFuture(future Future) Future {
	return f.then(f)
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
	f.onPanic = onPanic
	if f.IsDone() && f.panicEntity != nil {
		f.handlePanic(f.panicEntity)
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
	f.propogatePanic = false
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
	f.appendFuture(nextFuture)
	nextFuture.prevFuture = f
	// if current future isn't started, start it
	if !f.isRunning.Load().(bool) && !f.IsDone() {
		f.start()
		return nextFuture
	}
	if f.IsDone() {
		if f.panicEntity != nil {
			f.notifyAndPropagatePanicChain(f.panicEntity)
		} else {
			f.notifyAndRunNext()
		}
	}
	return nextFuture
}

func (f *future) appendFuture(nextFuture *future) {
	f.nextFutures = append(f.nextFutures, nextFuture)
}

func (f *future) run() *future {
	if f.isRunning.Load().(bool) || f.IsDone() {
		return f
	}
	f.isRunning.Store(true)
	f.executor.Execute(f.execute)
	return f
}

func (f *future) execute() {
	defer func() {
		if recovered := recover(); recovered != nil {
			f.acceptPanic(recovered)
		}
	}()
	if !(f.task == nil && f.result == nil && f.panicEntity == nil && !f.isRunning.Load().(bool)) {
		result, err := f.task()
		if err != nil {
			f.acceptError(err)
		} else if result != nil {
			f.acceptResult(result)
		}
	}
}

func (f *future) acceptResult(result interface{}) {
	if result == nil || f.result != nil {
		return
	}
	f.result = result
	f.notifyAndRunNext()
}

func (f *future) acceptError(err error) {
	if err == nil || f.errEntity != nil {
		return
	}
	f.errEntity = err
	f.notifyAndRunNext()
}

func (f *future) acceptPanic(recovered interface{}) {
	if recovered == nil || f.panicEntity != nil {
		return
	}
	f.panicEntity = recovered
	f.handlePanic(recovered)
}

func (f *future) handlePanic(recovered interface{}) {
	if f.onPanic != nil {
		f.onPanic(recovered)
	}
	f.notifyAndPropagatePanicChain(recovered)
}

func (f *future) withNextFutures(cb func(f *future)) {
	if len(f.nextFutures) == 0 {
		return
	}
	for _, nf := range f.nextFutures {
		cb(nf)
	}
}

func (f *future) notifyAndRunNext() {
	f.openWaitLockAndStopRunning()
	f.withNextFutures(func(nextFuture *future) {
		if !nextFuture.isRunning.Load().(bool) {
			nextFuture.run()
		}
	})
}

func (f *future) notifyAndPropagatePanicChain(recovered interface{}) {
	f.openWaitLockAndStopRunning()
	if f.propogatePanic {
		f.withNextFutures(func(nextFuture *future) {
			if !nextFuture.isRunning.Load().(bool) {
				nextFuture.handlePanic(recovered)
			}
		})
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
	f.task = func() (_ interface{}, _ error) {
		resolver(func(computedResult interface{}) {
			f.acceptResult(computedResult)
		}, func(catchedErr error) {
			f.acceptError(catchedErr)
		})
		return
	}
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
	if rawFuture.panicEntity == nil {
		return false
	}
	return rawFuture.panicEntity == canceledError
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
