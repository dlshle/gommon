package notification

import (
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"

	"github.com/dlshle/gommon/async"
)

const DefaultMaxListeners = 256

type EventListener[T comparable] func(T)
type Disposable func()

type notificationEmitter[T comparable] struct {
	listeners                map[string][]EventListener[T]
	lock                     *sync.RWMutex
	maxNumOfMessageListeners int
}

type WRNotificationEmitter[T comparable] interface {
	HasEvent(eventID string) bool
	MessageListenerCount(eventID string) int
	Notify(eventID string, payload T)
	NotifyAsync(eventID string, payload T, executor async.Executor)
	On(eventID string, listener EventListener[T]) (Disposable, error)
	Once(eventID string, listener EventListener[T]) (Disposable, error)
	Off(eventID string, listener EventListener[T])
	OffAll(eventID string)
}

func New[T comparable](maxMessageListenerCount int) WRNotificationEmitter[T] {
	if maxMessageListenerCount < 1 || maxMessageListenerCount > DefaultMaxListeners {
		maxMessageListenerCount = DefaultMaxListeners
	}
	return &notificationEmitter[T]{make(map[string][]EventListener[T]), new(sync.RWMutex), maxMessageListenerCount}
}

func (e *notificationEmitter[T]) withWrite(cb func()) {
	e.lock.Lock()
	defer e.lock.Unlock()
	cb()
}

func (e *notificationEmitter[T]) getMessageListeners(eventID string) []EventListener[T] {
	e.lock.RLock()
	defer e.lock.RUnlock()
	return e.listeners[eventID]
}

func (e *notificationEmitter[T]) addMessageListener(eventID string, listener EventListener[T]) (err error) {
	e.withWrite(func() {
		listeners := e.listeners[eventID]
		if listeners == nil {
			listeners = make([]EventListener[T], 0, e.maxNumOfMessageListeners)
		} else if len(listeners) >= e.maxNumOfMessageListeners {
			err = fmt.Errorf("listener count exceeded maxMessageListenerCount for event " +
				eventID +
				", please use SetMaxMessageListenerCount to top maxMessageListenerCount.")
			return
		}
		e.listeners[eventID] = append(listeners, listener)
	})
	return
}

func (e *notificationEmitter[T]) indexOfMessageListener(eventID string, listener EventListener[T]) int {
	listenerPtr := reflect.ValueOf(listener).Pointer()
	e.lock.RLock()
	defer e.lock.RUnlock()
	if e.listeners[eventID] == nil {
		return -1
	}
	for i, f := range e.listeners[eventID] {
		currPtr := reflect.ValueOf(f).Pointer()
		if listenerPtr == currPtr {
			return i
		}
	}
	return -1
}

func (e *notificationEmitter[T]) removeIthMessageListener(eventID string, listenerIdx int) {
	if listenerIdx == -1 || e.MessageListenerCount(eventID) == 0 {
		return
	}
	e.withWrite(func() {
		allMessageListeners := e.listeners[eventID]
		if len(allMessageListeners) == 0 {
			return
		}
		if len(allMessageListeners) == 1 {
			delete(e.listeners, eventID)
		} else {
			e.listeners[eventID] = append(allMessageListeners[:listenerIdx], allMessageListeners[listenerIdx+1:]...)
		}
	})
}

func (e *notificationEmitter[T]) HasEvent(eventID string) bool {
	e.lock.RLock()
	defer e.lock.RUnlock()
	return e.listeners[eventID] != nil
}

func (e *notificationEmitter[T]) Notify(eventID string, payload T) {
	if !e.HasEvent(eventID) {
		return
	}
	e.lock.RLock()
	listeners := e.listeners[eventID]
	e.lock.RUnlock()
	var wg sync.WaitGroup
	for _, f := range listeners {
		if f != nil {
			wg.Add(1)
			go func(listener EventListener[T]) {
				listener(payload)
				wg.Done()
			}(f)
		}
	}
	wg.Wait()
}

func (e *notificationEmitter[T]) NotifyAsync(eventID string, payload T, executor async.Executor) {
	if !e.HasEvent(eventID) {
		return
	}
	e.lock.RLock()
	listeners := e.listeners[eventID]
	e.lock.RUnlock()
	for _, f := range listeners {
		if f != nil {
			executor.Execute(func() {
				f(payload)
			})
		}
	}
}

func (e *notificationEmitter[T]) MessageListenerCount(eventID string) int {
	e.lock.RLock()
	defer e.lock.RUnlock()
	if e.listeners[eventID] == nil {
		return 0
	}
	return len(e.listeners[eventID])
}

func (e *notificationEmitter[T]) On(eventID string, listener EventListener[T]) (Disposable, error) {
	err := e.addMessageListener(eventID, listener)
	if err != nil {
		return nil, err
	}
	return func() {
		e.Off(eventID, listener)
	}, nil
}

func (e *notificationEmitter[T]) Once(eventID string, listener EventListener[T]) (Disposable, error) {
	hasFired := atomic.Value{}
	hasFired.Store(false)
	// need this to refer from the actualMessageListener
	var actualMessageListenerPtr func(T)
	actualMessageListener := func(param T) {
		if hasFired.Load().(bool) {
			e.Off(eventID, actualMessageListenerPtr)
			return
		}
		listener(param)
		e.Off(eventID, actualMessageListenerPtr)
		hasFired.Store(true)
	}
	actualMessageListenerPtr = actualMessageListener
	err := e.addMessageListener(eventID, actualMessageListenerPtr)
	if err != nil {
		return nil, err
	}
	return func() {
		e.Off(eventID, actualMessageListenerPtr)
		// manually free two pointers
		actualMessageListenerPtr = nil
		actualMessageListener = nil
	}, nil
}

func (e *notificationEmitter[T]) Off(eventID string, listener EventListener[T]) {
	if !e.HasEvent(eventID) {
		return
	}
	listenerIdx := e.indexOfMessageListener(eventID, listener)
	e.removeIthMessageListener(eventID, listenerIdx)
}

func (e *notificationEmitter[T]) OffAll(eventID string) {
	if !e.HasEvent(eventID) {
		return
	}
	e.withWrite(func() {
		e.listeners[eventID] = nil
	})
}
