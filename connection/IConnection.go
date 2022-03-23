package connection

type IConnection interface {
	ConnectionType() uint8
	Close() error
	Read() ([]byte, error)
	OnMessage(func([]byte))
	Write([]byte) error
	Address() string
	OnError(func(error))
	OnClose(func(error))
	State() int
	ReadLoop()
	String() string
	IsLive() bool
}

const (
	StateIdle         = 0
	StateReading      = 1
	StateStopping     = 2
	StateStopped      = 3
	StateClosing      = 4
	StateDisconnected = 5
)
