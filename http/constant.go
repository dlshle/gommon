package http

// Pool
// constants
const (
	PoolStatusIdle        = 0
	PoolStatusStarting    = 1
	PoolStatusRunning     = 2
	PoolStatusTerminating = 3
	PoolStatusStopped     = 4
)

var poolStatusStringMap map[int]string

func initPoolStatusStringMap() {
	poolStatusStringMap = make(map[int]string)
	poolStatusStringMap[PoolStatusIdle] = "Idle"
	poolStatusStringMap[PoolStatusStarting] = "Starting"
	poolStatusStringMap[PoolStatusRunning] = "Running"
	poolStatusStringMap[PoolStatusTerminating] = "Terminating"
	poolStatusStringMap[PoolStatusStopped] = "Stopped"
}
