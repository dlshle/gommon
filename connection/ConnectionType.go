package connection

var typeStringMap map[uint8]string

func init() {
	typeStringMap = make(map[uint8]string)
	typeStringMap[TypeTCP] = "TCP"
	typeStringMap[TypeWS] = "WS"
	typeStringMap[TypeUDP] = "UDP"
	typeStringMap[TypeRTC] = "RTC"
	typeStringMap[TypeHTTP] = "HTTP"
}

const (
	TypeTCP = iota
	TypeWS
	TypeUDP
	TypeRTC
	TypeHTTP
)

func IsAsyncType(connType uint8) bool {
	return connType < TypeHTTP
}

func TypeString(connType uint8) string {
	return typeStringMap[connType]
}
