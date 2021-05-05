package notification

import (
	"errors"
	"reflect"
	"sync"
	"sync/atomic"
)

const DefaultMaxListeners = 1024

var globalEmitter INotificationEmitter

func init() {
	globalEmitter = New(DefaultMaxListeners)
}

type Listener func(interface{})
type Disposable func()

type NotificationEmitter struct {
	notificationMap map[string][]Listener
	lock *sync.RWMutex
	maxNumOfListeners int
}

type INotificationEmitter interface {
	HasEvent(eventId string) bool
	ListenerCount(eventId string) int
	Notify(eventId string, payload interface{})
	On(eventId string, listener Listener) (Disposable, error)
	Once(eventId string, listener Listener) (Disposable, error)
	Off(eventId string, listener Listener)
	OffAll(eventId string)
}

func New(maxListenerCount int) INotificationEmitter {
	if maxListenerCount < 1 || maxListenerCount > DefaultMaxListeners {
		maxListenerCount = DefaultMaxListeners
	}
	return &NotificationEmitter{make(map[string][]Listener), new(sync.RWMutex), maxListenerCount}
}

func (e *NotificationEmitter) withWrite(cb func()) {
	e.lock.Lock()
	defer e.lock.Unlock()
	cb()
}

func (e *NotificationEmitter) getListeners(eventId string) []Listener {
	e.lock.RLock()
	defer e.lock.RUnlock()
	return e.notificationMap[eventId]
}

func (e *NotificationEmitter) addListener(eventId string, listener Listener) (err error) {
	e.withWrite(func() {
		listeners := e.notificationMap[eventId]
		if listeners == nil {
			listeners = make([]Listener, 0, e.maxNumOfListeners)
		} else if len(listeners) >= e.maxNumOfListeners {
				err = errors.New("listener count exceeded maxListenerCount for event " +
					eventId +
					", please use SetMaxListenerCount to top maxListenerCount.")
				return
		}
		e.notificationMap[eventId] = append(listeners, listener)
	})
	return
}

func (e *NotificationEmitter) indexOfListener(eventId string, listener Listener) int {
	listenerPtr := reflect.ValueOf(listener).Pointer()
	e.lock.RLock()
	defer e.lock.RUnlock()
	if e.notificationMap[eventId] == nil {
		return -1
	}
	for i, f := range e.notificationMap[eventId] {
		currPtr := reflect.ValueOf(f).Pointer()
		if listenerPtr == currPtr {
			return i
		}
	}
	return -1
}

func (e *NotificationEmitter) removeIthListener(eventId string, listenerIdx int) {
	if e.ListenerCount(eventId) < listenerIdx {
		return
	}
	e.withWrite(func() {
		allListeners := e.notificationMap[eventId]
		e.notificationMap[eventId] = append(allListeners[:listenerIdx], allListeners[listenerIdx+1:]...)
	})
}

func (e *NotificationEmitter) HasEvent(eventId string) bool {
	e.lock.RLock()
	defer e.lock.RUnlock()
	return e.notificationMap[eventId] != nil
}

func (e *NotificationEmitter) Notify(eventId string, payload interface{}) {
	if !e.HasEvent(eventId) {
		return
	}
	e.lock.RLock()
	listeners := e.notificationMap[eventId]
	e.lock.RUnlock()
	// defer e.lock.RUnlock()
	var wg sync.WaitGroup
	for _, f := range listeners {
		if f != nil {
			wg.Add(1)
			go func(listener Listener) {
				listener(payload)
				wg.Done()
			}(f)
		}
	}
	wg.Wait()
}

func (e *NotificationEmitter) ListenerCount(eventId string) int {
	e.lock.RLock()
	defer e.lock.RUnlock()
	if e.notificationMap[eventId] == nil {
		return 0
	}
	return len(e.notificationMap[eventId])
}

func (e *NotificationEmitter) On(eventId string, listener Listener) (Disposable, error) {
	err := e.addListener(eventId, listener)
	if err != nil {
		return nil, err
	}
	return func() {
		e.Off(eventId, listener)
	}, nil
}

func (e *NotificationEmitter) Once(eventId string, listener Listener) (Disposable, error) {
	hasFired := atomic.Value{}
	hasFired.Store(false)
	// need this to refer from the actualListener
	var actualListenerPtr func(interface{})
	actualListener := func(param interface{}) {
		if hasFired.Load().(bool) {
			e.Off(eventId, actualListenerPtr)
			return
		}
		listener(param)
		e.Off(eventId, actualListenerPtr)
		hasFired.Store(true)
	}
	actualListenerPtr = actualListener
	err := e.addListener(eventId, actualListenerPtr)
	if err != nil {
		return nil, err
	}
	return func() {
		e.Off(eventId, actualListenerPtr)
		// manually free two pointers
		actualListenerPtr = nil
		actualListener = nil
	}, nil
}

func (e *NotificationEmitter) Off(eventId string, listener Listener) {
	if !e.HasEvent(eventId) {
		return
	}
	listenerIdx := e.indexOfListener(eventId, listener)
	e.removeIthListener(eventId, listenerIdx)
}

func (e *NotificationEmitter) OffAll(eventId string) {
	if !e.HasEvent(eventId) {
		return
	}
	e.withWrite(func() {
		e.notificationMap[eventId] = nil
	})
}

func HasEvent(eventId string) bool {
	return globalEmitter.HasEvent(eventId)
}

func Notify(eventId string, payload interface{}) {
	globalEmitter.Notify(eventId, payload)
}

func ListenerCount(eventId string) int {
	return globalEmitter.ListenerCount(eventId)
}

func On(eventId string, listener Listener) (Disposable, error) {
	return globalEmitter.On(eventId, listener)
}

func Once(eventId string, listener Listener) (Disposable, error) {
	return globalEmitter.Once(eventId, listener)
}

func Off(eventId string, listener Listener) {
	globalEmitter.Off(eventId, listener)
}

func OffAll(eventId string) {
	globalEmitter.OffAll(eventId)
}