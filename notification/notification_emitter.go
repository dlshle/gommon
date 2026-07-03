package notification

import (
	"fmt"
	"reflect"
	"sync"

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
	Notify(eventID string, payload T) bool
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

func (e *notificationEmitter[T]) addMessageListener(eventID string, listener EventListener[T]) (err error) {
	e.withWrite(func() {
		listeners := e.listeners[eventID]
		if listeners == nil {
			listeners = make([]EventListener[T], 0, e.maxNumOfMessageListeners)
		} else if len(listeners) >= e.maxNumOfMessageListeners {
			err = fmt.Errorf("listener count exceeded maxMessageListenerCount(%d) for event %s", e.maxNumOfMessageListeners, eventID)
			return
		}
		e.listeners[eventID] = append(listeners, listener)
	})
	return
}

// copyListeners returns a snapshot of the current listeners for the given event.
// The snapshot is made under the read lock so callers can iterate safely after
// the lock is released.
func (e *notificationEmitter[T]) copyListeners(eventID string) []EventListener[T] {
	e.lock.RLock()
	defer e.lock.RUnlock()
	listeners := e.listeners[eventID]
	if len(listeners) == 0 {
		return nil
	}
	cpy := make([]EventListener[T], len(listeners))
	copy(cpy, listeners)
	return cpy
}

func (e *notificationEmitter[T]) HasEvent(eventID string) bool {
	e.lock.RLock()
	defer e.lock.RUnlock()
	return len(e.listeners[eventID]) > 0
}

func (e *notificationEmitter[T]) Notify(eventID string, payload T) bool {
	listeners := e.copyListeners(eventID)
	if len(listeners) == 0 {
		return false
	}
	var wg sync.WaitGroup
	for _, f := range listeners {
		if f != nil {
			wg.Add(1)
			go func(listener EventListener[T]) {
				defer wg.Done()
				defer func() {
					_ = recover()
				}()
				listener(payload)
			}(f)
		}
	}
	wg.Wait()
	return true
}

func (e *notificationEmitter[T]) NotifyAsync(eventID string, payload T, executor async.Executor) {
	listeners := e.copyListeners(eventID)
	if len(listeners) == 0 {
		return
	}
	for _, f := range listeners {
		if f != nil {
			listener := f
			executor.Execute(func() {
				listener(payload)
			})
		}
	}
}

func (e *notificationEmitter[T]) MessageListenerCount(eventID string) int {
	e.lock.RLock()
	defer e.lock.RUnlock()
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
	var once sync.Once
	var actualMessageListener EventListener[T]
	actualMessageListener = func(param T) {
		once.Do(func() {
			listener(param)
			e.Off(eventID, actualMessageListener)
		})
	}
	err := e.addMessageListener(eventID, actualMessageListener)
	if err != nil {
		return nil, err
	}
	return func() {
		e.Off(eventID, actualMessageListener)
	}, nil
}

func (e *notificationEmitter[T]) Off(eventID string, listener EventListener[T]) {
	e.removeMessageListener(eventID, listener)
}

func (e *notificationEmitter[T]) removeMessageListener(eventID string, listener EventListener[T]) {
	if listener == nil {
		return
	}
	e.withWrite(func() {
		listeners := e.listeners[eventID]
		if len(listeners) == 0 {
			return
		}
		targetPtr := reflect.ValueOf(listener).Pointer()
		for i, f := range listeners {
			if f == nil {
				continue
			}
			if reflect.ValueOf(f).Pointer() == targetPtr {
				if len(listeners) == 1 {
					delete(e.listeners, eventID)
				} else {
					e.listeners[eventID] = append(listeners[:i], listeners[i+1:]...)
				}
				return
			}
		}
	})
}

func (e *notificationEmitter[T]) OffAll(eventID string) {
	e.withWrite(func() {
		delete(e.listeners, eventID)
	})
}
