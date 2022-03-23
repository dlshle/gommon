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

type WRNotificationEmitter struct {
	listeners                map[string][]EventListener
	lock                     *sync.RWMutex
	maxNumOfMessageListeners int
}

type IWRNotificationEmitter interface {
	HasEvent(eventId string) bool
	MessageListenerCount(eventId string) int
	Notify(eventId string, payload interface{})
	On(eventId string, listener EventListener) (Disposable, error)
	Once(eventId string, listener EventListener) (Disposable, error)
	Off(eventId string, listener EventListener)
	OffAll(eventId string)
}

func New(maxMessageListenerCount int) IWRNotificationEmitter {
	if maxMessageListenerCount < 1 || maxMessageListenerCount > DefaultMaxListeners {
		maxMessageListenerCount = DefaultMaxListeners
	}
	return &WRNotificationEmitter{make(map[string][]EventListener), new(sync.RWMutex), maxMessageListenerCount}
}

func (e *WRNotificationEmitter) withWrite(cb func()) {
	e.lock.Lock()
	defer e.lock.Unlock()
	cb()
}

func (e *WRNotificationEmitter) getMessageListeners(eventId string) []EventListener {
	e.lock.RLock()
	defer e.lock.RUnlock()
	return e.listeners[eventId]
}

func (e *WRNotificationEmitter) addMessageListener(eventId string, listener EventListener) (err error) {
	e.withWrite(func() {
		listeners := e.listeners[eventId]
		if listeners == nil {
			listeners = make([]EventListener, 0, e.maxNumOfMessageListeners)
		} else if len(listeners) >= e.maxNumOfMessageListeners {
			err = fmt.Errorf("listener count exceeded maxMessageListenerCount for event " +
				eventId +
				", please use SetMaxMessageListenerCount to top maxMessageListenerCount.")
			return
		}
		e.listeners[eventId] = append(listeners, listener)
	})
	return
}

func (e *WRNotificationEmitter) indexOfMessageListener(eventId string, listener EventListener) int {
	listenerPtr := reflect.ValueOf(listener).Pointer()
	e.lock.RLock()
	defer e.lock.RUnlock()
	if e.listeners[eventId] == nil {
		return -1
	}
	for i, f := range e.listeners[eventId] {
		currPtr := reflect.ValueOf(f).Pointer()
		if listenerPtr == currPtr {
			return i
		}
	}
	return -1
}

func (e *WRNotificationEmitter) removeIthMessageListener(eventId string, listenerIdx int) {
	if listenerIdx == -1 || e.MessageListenerCount(eventId) == 0 {
		return
	}
	e.withWrite(func() {
		allMessageListeners := e.listeners[eventId]
		if len(allMessageListeners) == 0 {
			return
		}
		if len(allMessageListeners) == 1 {
			delete(e.listeners, eventId)
		} else {
			e.listeners[eventId] = append(allMessageListeners[:listenerIdx], allMessageListeners[listenerIdx+1:]...)
		}
	})
}

func (e *WRNotificationEmitter) HasEvent(eventId string) bool {
	e.lock.RLock()
	defer e.lock.RUnlock()
	return e.listeners[eventId] != nil
}

func (e *WRNotificationEmitter) Notify(eventId string, payload interface{}) {
	if !e.HasEvent(eventId) {
		return
	}
	e.lock.RLock()
	listeners := e.listeners[eventId]
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

func (e *WRNotificationEmitter) MessageListenerCount(eventId string) int {
	e.lock.RLock()
	defer e.lock.RUnlock()
	if e.listeners[eventId] == nil {
		return 0
	}
	return len(e.listeners[eventId])
}

func (e *WRNotificationEmitter) On(eventId string, listener EventListener) (Disposable, error) {
	err := e.addMessageListener(eventId, listener)
	if err != nil {
		return nil, err
	}
	return func() {
		e.Off(eventId, listener)
	}, nil
}

func (e *WRNotificationEmitter) Once(eventId string, listener EventListener) (Disposable, error) {
	hasFired := atomic.Value{}
	hasFired.Store(false)
	// need this to refer from the actualMessageListener
	var actualMessageListenerPtr func(interface{})
	actualMessageListener := func(param interface{}) {
		if hasFired.Load().(bool) {
			e.Off(eventId, actualMessageListenerPtr)
			return
		}
		listener(param)
		e.Off(eventId, actualMessageListenerPtr)
		hasFired.Store(true)
	}
	actualMessageListenerPtr = actualMessageListener
	err := e.addMessageListener(eventId, actualMessageListenerPtr)
	if err != nil {
		return nil, err
	}
	return func() {
		e.Off(eventId, actualMessageListenerPtr)
		// manually free two pointers
		actualMessageListenerPtr = nil
		actualMessageListener = nil
	}, nil
}

func (e *WRNotificationEmitter) Off(eventId string, listener EventListener) {
	if !e.HasEvent(eventId) {
		return
	}
	listenerIdx := e.indexOfMessageListener(eventId, listener)
	e.removeIthMessageListener(eventId, listenerIdx)
}

func (e *WRNotificationEmitter) OffAll(eventId string) {
	if !e.HasEvent(eventId) {
		return
	}
	e.withWrite(func() {
		e.listeners[eventId] = nil
	})
}
