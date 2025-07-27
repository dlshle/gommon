package connection

type ConnectionState int

type Connection interface {
	ConnectionType() uint8
	Close() error
	Read() ([]byte, error)
	OnMessage(func([]byte))
	Write([]byte) error
	Address() string
	OnError(func(error))
	OnClose(func(error))
	State() ConnectionState
	ReadLoop()
	String() string
	IsLive() bool
}

const (
	StateIdle         ConnectionState = 0
	StateReading      ConnectionState = 1
	StateStopping     ConnectionState = 2
	StateStopped      ConnectionState = 3
	StateClosing      ConnectionState = 4
	StateDisconnected ConnectionState = 5
)
