package connection

type IClient interface {
	Connect(token string) (IConnection, error)
	// ReadLoop()
	// Disconnect() error
	// Write(data []byte) error
	// Read() ([]byte, error)
	OnConnectionEstablished(cb func(IConnection))
	OnDisconnect(cb func(error))
	OnMessage(cb func([]byte))
	OnError(cb func(error))
}
