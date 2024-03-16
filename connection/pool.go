package connection

import (
	"sync"

	"github.com/dlshle/gommon/logging"
)

const (
	DefaultFactoryRetryCount     = 0
	ConnectionPoolErrPoolClosed  = 1
	ConnectionPoolErrGetTimeout  = 2
	ConnectionPoolErrNumInUseIs0 = 3
	ConnectionPollErrInvalidConn = 4
)

type ConnectionPoolError struct {
	code uint8
	msg  string
}

func NewConnectionPoolError(code uint8, msg string) ConnectionPoolError {
	return ConnectionPoolError{code, msg}
}

func (e ConnectionPoolError) Error() string {
	return e.msg
}

func (e ConnectionPoolError) Code() uint8 {
	return e.code
}

type connectionPool struct {
	consumerPool chan Connection
	producerChan chan bool // works kinda like producer sem
	// getTimeoutInMS time.Duration // max timeout for waiting for an idle conn(create new conn after timeout).
	// idleTimeoutMs  time.Duration // how long should an idle connection be omitted from the pool
	numInUse    int
	numMaxSize  int
	rwLock      *sync.RWMutex
	connFactory func() (Connection, error)
	logger      logging.Logger
}

type ConnectionPool interface {
	Get() (Connection, error)
	Return(Connection) error
	IsClosed() bool
	Close()
}

func NewConnectionPool(loggerPrefix string, factory func() (Connection, error), initSize int, maxSize int) (ConnectionPool, error) {
	pool := &connectionPool{
		consumerPool: make(chan Connection, maxSize),
		producerChan: make(chan bool, maxSize),
		// getTimeoutInMS: timeoutInMs,
		numInUse:    0,
		numMaxSize:  maxSize,
		rwLock:      new(sync.RWMutex),
		connFactory: factory,
		logger:      logging.GlobalLogger.WithPrefix(loggerPrefix),
	}
	return pool, pool.init(initSize)
}

// initialize the pool with #initSize live connections
func (p *connectionPool) init(initSize int) error {
	for p.numProducedConnections() < initSize {
		if err := p.produceConnection(); err != nil {
			return err
		}
	}
	return nil
}

func (p *connectionPool) withWrite(cb func()) {
	p.rwLock.Lock()
	defer p.rwLock.RUnlock()
	cb()
}

func (p *connectionPool) withRead(cb func()) {
	p.rwLock.RLock()
	defer p.rwLock.RUnlock()
	cb()
}

func (p *connectionPool) createConnection(retry int, lastErr error) (conn Connection, err error) {
	if retry == 0 {
		return nil, lastErr
	}
	conn, err = p.connFactory()
	if err != nil {
		return p.createConnection(retry-1, err)
	}
	return
}

func (p *connectionPool) produceConnection() error {
	// will block when produced over #maxPoolSize connections
	<-p.producerChan
	c, e := p.createConnection(DefaultFactoryRetryCount, nil)
	if e != nil {
		if c != nil {
			c.Close()
		}
		return e
	}
	p.consumerPool <- c
	return nil
}

func (p *connectionPool) produceAndGetConnection() (Connection, error) {
	err := p.produceConnection()
	if err != nil {
		return nil, err
	}
	return p.safeGet()
}

func (p *connectionPool) Return(conn Connection) (err error) {
	if p.IsClosed() {
		return NewConnectionPoolError(ConnectionPoolErrPoolClosed, "pool already closed")
	}
	if conn == nil {
		return NewConnectionPoolError(ConnectionPollErrInvalidConn, "nil connection")
	}
	p.withWrite(func() {
		if p.numInUse == 0 {
			err = NewConnectionPoolError(ConnectionPoolErrNumInUseIs0, "number of in-use connection is 0")
			return
		}
		if conn.IsLive() {
			p.consumerPool <- conn
		} else {
			// only sem post to producer chan when a closed conn is returned because we try to keep all connections in
			// pool alive
			p.Close()
			p.producerChan <- true
		}
		p.numInUse--
	})
	return
}

func (p *connectionPool) safeGet() (conn Connection, err error) {
	for !p.IsClosed() {
		conn = <-p.consumerPool
		if conn.IsLive() {
			break
		} else {
			p.Return(conn)
			// if no conn in consumer pool, try to produce one
			if len(p.consumerPool) == 0 {
				if err = p.produceConnection(); err != nil {
					return nil, err
				}
			}
		}
	}
	p.withWrite(func() {
		p.numInUse++
	})
	return
}

/*
// TODO  we need to somehow return err on get timeout, Get should be something like doGet
func (p *connectionPool) doGet() (IConnection, error) {
	select {
	case conn := <-p.consumerPool:
		if conn.IsLive() {
			return conn, nil
		}
		conn.Close()
		p.Return(conn)
		return p.safeGet()
	case <-time.After(p.getTimeoutInMS):
		return nil, NewConnectionPoolError(ConnectionPoolErrGetTimeout, "get connection timeout")
	}
}
*/

func (p *connectionPool) Get() (Connection, error) {
	if p.IsClosed() {
		return nil, NewConnectionPoolError(ConnectionPoolErrPoolClosed, "pool already closed")
	}
	if numProduced := p.numProducedConnections(); p.numInUse >= numProduced && numProduced < p.numMaxSize {
		return p.produceAndGetConnection()
	} else {
		// if has spare conn, return; if no spare conn and exceeded max size, wait for the next available conn bitch
		return p.safeGet()
	}
}

func (p *connectionPool) Close() {
	if p.IsClosed() {
		return
	}
	p.withWrite(func() {
		for c := range p.consumerPool {
			c.Close()
		}
		close(p.consumerPool)
		p.consumerPool = nil
		p.connFactory = nil
		p.rwLock = nil
		p.logger = nil
	})
}

func (p *connectionPool) IsClosed() (closed bool) {
	p.withRead(func() {
		closed = p.consumerPool == nil
	})
	return closed
}

func (p *connectionPool) numProducedConnections() (num int) {
	p.withRead(func() {
		num = len(p.producerChan)
	})
	return
}
