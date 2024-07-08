package http

import (
	"math/rand"
	"net/http"
	"sync"
	"time"
)

var (
	requestPool sync.Pool = sync.Pool{
		New: func() any {
			return &http.Request{}
		},
	}
)

// request status error message_dispatcher
var requestStatusErrorStringMap map[int32]string
var requestStatusErrorCodeMap map[int32]int

// trackableRequest Status
const (
	RequestStatusIdle       = 0
	RequestStatusWaiting    = 1
	RequestStatusInProgress = 2
	RequestStatusCancelled  = 9
	RequestStatusDone       = 10
)

// Rand utils
var randomGenerator = rand.New(rand.NewSource(time.Now().UnixNano()))

func initRequestStatusErrorMaps() {
	requestStatusErrorCodeMap = make(map[int32]int)
	requestStatusErrorStringMap = make(map[int32]string)
	requestStatusErrorStringMap[RequestStatusInProgress] = "Handle is in progress"
	requestStatusErrorCodeMap[RequestStatusInProgress] = ErrRequestInProgress
	requestStatusErrorStringMap[RequestStatusCancelled] = "Handle is cancelled"
	requestStatusErrorCodeMap[RequestStatusCancelled] = ErrRequestCancelled
	requestStatusErrorStringMap[RequestStatusDone] = "Handle is finished"
	requestStatusErrorCodeMap[RequestStatusDone] = ErrRequestFinished
}

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
