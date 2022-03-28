package async

import "sync"

type callState struct {
	res      interface{}
	err      error
	waitable Waitable
}

func (s *callState) waitAndGet() (interface{}, error) {
	s.waitable.Wait()
	return s.res, s.err
}

type SingleRequest interface {
	Do(key string, fn func() (interface{}, error)) (interface{}, error)
}

type singleRequestGroup struct {
	calls  map[string]*callState
	rwLock *sync.RWMutex
}

func NewRequestGroup() SingleRequest {
	return singleRequestGroup{
		calls:  make(map[string]*callState),
		rwLock: new(sync.RWMutex),
	}
}

func (g singleRequestGroup) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	call := g.get(key)
	if call == nil {
		call = g.create(key, fn)
	}
	return call.waitAndGet()
}

func (g singleRequestGroup) create(key string, fn func() (interface{}, error)) (cs *callState) {
	var (
		callStateExists bool
		waitLock        *WaitLock
	)
	g.withWrite(func() {
		oldCS := g.calls[key]
		if oldCS != nil && !oldCS.waitable.IsOpen() {
			cs = oldCS
			callStateExists = true
			return
		}
		waitLock = NewWaitLock()
		cs = &callState{nil, nil, waitLock}
		g.calls[key] = cs
	})
	if callStateExists {
		return
	}
	cs.res, cs.err = fn()
	g.withWrite(func() {
		waitLock.Open()
		delete(g.calls, key)
	})
	return
}

func (g singleRequestGroup) get(key string) (cs *callState) {
	g.withRead(func() {
		cs = g.calls[key]
	})
	return
}

func (g singleRequestGroup) withRead(cb func()) {
	g.rwLock.RLock()
	defer g.rwLock.RUnlock()
	cb()
}

func (g singleRequestGroup) withWrite(cb func()) {
	g.rwLock.Lock()
	defer g.rwLock.Unlock()
	cb()
}
