package notification

import (
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
)

const DefaultMaxListeners = 256

type EventListener func(interface{})
type Disposable func()

type notificationEmitter struct {
	listeners                map[string][]EventListener
	lock                     *sync.RWMutex
	maxNumOfMessageListeners int
}

type WRNotificationEmitter interface {
	HasEvent(eventID string) bool
	MessageListenerCount(eventID string) int
	Notify(eventID string, payload interface{})
	On(eventID string, listener EventListener) (Disposable, error)
	Once(eventID string, listener EventListener) (Disposable, error)
	Off(eventID string, listener EventListener)
	OffAll(eventID string)
}

func New(maxMessageListenerCount int) WRNotificationEmitter {
	if maxMessageListenerCount < 1 || maxMessageListenerCount > DefaultMaxListeners {
		maxMessageListenerCount = DefaultMaxListeners
	}
	return &notificationEmitter{make(map[string][]EventListener), new(sync.RWMutex), maxMessageListenerCount}
}

func (e *notificationEmitter) withWrite(cb func()) {
	e.lock.Lock()
	defer e.lock.Unlock()
	cb()
}

func (e *notificationEmitter) getMessageListeners(eventID string) []EventListener {
	e.lock.RLock()
	defer e.lock.RUnlock()
	return e.listeners[eventID]
}

func (e *notificationEmitter) addMessageListener(eventID string, listener EventListener) (err error) {
	e.withWrite(func() {
		listeners := e.listeners[eventID]
		if listeners == nil {
			listeners = make([]EventListener, 0, e.maxNumOfMessageListeners)
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

func (e *notificationEmitter) indexOfMessageListener(eventID string, listener EventListener) int {
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

func (e *notificationEmitter) removeIthMessageListener(eventID string, listenerIdx int) {
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

func (e *notificationEmitter) HasEvent(eventID string) bool {
	e.lock.RLock()
	defer e.lock.RUnlock()
	return e.listeners[eventID] != nil
}

func (e *notificationEmitter) Notify(eventID string, payload interface{}) {
	if !e.HasEvent(eventID) {
		return
	}
	e.lock.RLock()
	listeners := e.listeners[eventID]
	e.lock.RUnlock()
	// defer e.lock.RUnlock()
	var wg sync.WaitGroup
	for _, f := range listeners {
		if f != nil {
			wg.Add(1)
			go func(listener EventListener) {
				listener(payload)
				wg.Done()
			}(f)
		}
	}
	wg.Wait()
}

func (e *notificationEmitter) MessageListenerCount(eventID string) int {
	e.lock.RLock()
	defer e.lock.RUnlock()
	if e.listeners[eventID] == nil {
		return 0
	}
	return len(e.listeners[eventID])
}

func (e *notificationEmitter) On(eventID string, listener EventListener) (Disposable, error) {
	err := e.addMessageListener(eventID, listener)
	if err != nil {
		return nil, err
	}
	return func() {
		e.Off(eventID, listener)
	}, nil
}

func (e *notificationEmitter) Once(eventID string, listener EventListener) (Disposable, error) {
	hasFired := atomic.Value{}
	hasFired.Store(false)
	// need this to refer from the actualMessageListener
	var actualMessageListenerPtr func(interface{})
	actualMessageListener := func(param interface{}) {
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

func (e *notificationEmitter) Off(eventID string, listener EventListener) {
	if !e.HasEvent(eventID) {
		return
	}
	listenerIdx := e.indexOfMessageListener(eventID, listener)
	e.removeIthMessageListener(eventID, listenerIdx)
}

func (e *notificationEmitter) OffAll(eventID string) {
	if !e.HasEvent(eventID) {
		return
	}
	e.withWrite(func() {
		e.listeners[eventID] = nil
	})
}
