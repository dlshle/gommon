package async

import (
	"fmt"
	"reflect"
	"time"
)

func RaceTimeoutWithOperation(duration time.Duration, op func()) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	done := make(chan struct{})
	var panicEntity interface{}
	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				panicEntity = recovered
			}
			close(done)
		}()
		op()
	}()
	select {
	case <-timer.C:
		// If op completed concurrently with the timeout, prefer completion.
		select {
		case <-done:
			if panicEntity != nil {
				panic(panicEntity)
			}
			return nil
		default:
			return fmt.Errorf(TimeoutMsg)
		}
	case <-done:
		if panicEntity != nil {
			panic(panicEntity)
		}
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
