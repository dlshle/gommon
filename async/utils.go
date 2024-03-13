package async

import (
	"fmt"
	"reflect"
	"time"
)

func RaceTimeoutWithOperation(duration time.Duration, op func()) error {
	timer := time.NewTimer(duration)
	opChan := make(chan bool)
	go func() {
		op()
		close(opChan)
	}()
	select {
	case <-timer.C:
		return fmt.Errorf(TimeoutMsg)
	case <-opChan:
		return nil
	}
}

func Race(channels ...chan interface{}) {
	if channels == nil || len(channels) == 0 {
		return
	}
	cases := make([]reflect.SelectCase, len(channels), len(channels))
	for i, channel := range channels {
		cases[i] = reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(channel),
		}
	}
	reflect.Select(cases)
}
