package async

import (
	"fmt"
	"reflect"
	"sync/atomic"
	"time"
)

type FutureGetter interface {
	Wait()
	Get() interface{}
}

type Executor interface {
	Execute(task AsyncTask)
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
	canceledError = fmt.Errorf(CanceledMsg)
}

type Future interface {
	FutureGetter
	Run() FutureGetter
	// try to cancel the task before its execution
	Cancel()
	WaitWithTimeout(duration time.Duration) error
	GetWithTimeout(duration time.Duration) (interface{}, error)
	IsDone() bool
	Then(onSuccess func(interface{}) interface{}) Future
	OnPanic(onPanic func(interface{})) Future
}

type OptionalParamOperation func(interface{}) interface{}

type future struct {
	executor    Executor
	waitLock    *WaitLock
	task        ComputableAsyncTask
	result      interface{}
	panicEntity interface{}
	isRunning   atomic.Value
	prevFuture  *future
	nextFuture  *future
	onPanic     func(interface{})
}

func newAsyncTaskFuture(task AsyncTask, executor Executor) *future {
	computedTask := func() interface{} {
		task()
		return nil
	}
	return newFuture(computedTask, executor, nil)
}

func newComputedFuture(task ComputableAsyncTask, executor Executor) *future {
	return newFuture(task, executor, nil)
}

func newFuture(task ComputableAsyncTask, executor Executor, prevFuture *future) *future {
	f := &future{
		prevFuture: prevFuture,
		executor:   executor,
		waitLock:   NewWaitLock(),
		task:       task,
		isRunning:  atomic.Value{},
	}
	f.isRunning.Store(false)
	return f
}

func (f *future) Run() FutureGetter {
	if f.prevFuture != nil {
		f.prevFuture.Run()
	} else {
		f.run()
	}
	return f
}

func (f *future) Cancel() {
	if f.task == nil || f.isRunning.Load().(bool) || f.waitLock.IsOpen() {
		// try to cancel later futures
		f.withNextFuture(func(nextFuture *future) {
			nextFuture.Cancel()
		})
		return
	}
	f.executor = DirectExecutor
	f.task = func() interface{} {
		panic(canceledError)
	}
}

func (f *future) Wait() {
	if f.executor == DirectExecutor {
		return
	}
	f.waitLock.Wait()
}

func (f *future) WaitWithTimeout(duration time.Duration) error {
	if f.waitLock.IsOpen() {
		return nil
	}
	return raceTimeoutWithOperation(duration, f.Wait)
}

func (f *future) Get() interface{} {
	f.waitLock.Wait()
	return f.result
}

func (f *future) GetWithTimeout(duration time.Duration) (result interface{}, err error) {
	if f.waitLock.IsOpen() {
		return f.result, nil
	}
	err = raceTimeoutWithOperation(duration, func() {
		result = f.Get()
	})
	return
}

func (f *future) IsDone() bool {
	return f.waitLock.IsOpen()
}

func (f *future) Then(onSuccess func(interface{}) interface{}) Future {
	nextTask := func() interface{} {
		result, panicEntity := f.waitAndGetResultAndPanicEntity()
		if panicEntity != nil {
			f.handlePanic(panicEntity)
		}
		return onSuccess(result)
	}
	f.nextFuture = newFuture(nextTask, f.executor, f)
	if f.IsDone() {
		if f.panicEntity != nil {
			f.notifyAndPropagatePanicChain(f.panicEntity)
		} else {
			f.notifyAndRunNext()
		}
	}
	return f.nextFuture
}

func (f *future) OnPanic(onPanic func(interface{})) Future {
	f.onPanic = onPanic
	if f.IsDone() && f.panicEntity != nil {
		f.handlePanic(f.panicEntity)
	}
	return f
}

func (f *future) run() {
	if f.isRunning.Load().(bool) || f.IsDone() {
		return
	}
	f.isRunning.Store(true)
	f.executor.Execute(f.execute)
}

func (f *future) execute() {
	defer func() {
		if recovered := recover(); recovered != nil {
			f.handlePanic(recovered)
		}
	}()
	if !(f.task == nil && f.result == nil && f.panicEntity == nil && !f.isRunning.Load().(bool)) {
		f.result = f.task()
	}
	f.notifyAndRunNext()
}

func (f *future) handlePanic(recovered interface{}) {
	if f.onPanic != nil {
		f.onPanic(recovered)
	}
	f.panicEntity = recovered
	f.notifyAndPropagatePanicChain(recovered)
}

func (f *future) withNextFuture(cb func(f *future)) {
	if f.nextFuture == nil {
		return
	}
	cb(f.nextFuture)
}

func (f *future) notifyAndRunNext() {
	f.waitLock.Open()
	f.withNextFuture(func(nextFuture *future) {
		if !nextFuture.isRunning.Load().(bool) {
			nextFuture.run()
		}
	})
}

func (f *future) notifyAndPropagatePanicChain(recovered interface{}) {
	f.waitLock.Open()
	f.withNextFuture(func(nextFuture *future) {
		if !nextFuture.isRunning.Load().(bool) {
			nextFuture.handlePanic(recovered)
		}
	})
}

func (f *future) waitAndGetResultAndPanicEntity() (interface{}, interface{}) {
	f.Wait()
	return f.result, f.panicEntity
}

func Run(task AsyncTask, executor Executor) Future {
	f := newAsyncTaskFuture(task, executor)
	return f
}

func NewComputedFuture(task ComputableAsyncTask, executor Executor) Future {
	f := newComputedFuture(task, executor)
	return f
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

func raceTimeoutWithOperation(duration time.Duration, op func()) error {
	timer := time.NewTimer(duration)
	opChan := make(chan bool)
	go func() {
		op()
		close(opChan)
	}()
	select {
	case <-timer.C:
		return fmt.Errorf(TimeoutMsg)
	case <-opChan:
		return nil
	}
}

func race(channels ...chan interface{}) {
	if channels == nil || len(channels) == 0 {
		return
	}
	cases := make([]reflect.SelectCase, len(channels), len(channels))
	for i, channel := range channels {
		cases[i] = reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(channel),
		}
	}
	reflect.Select(cases)
}
