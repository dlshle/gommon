package timer

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestTimer(t *testing.T) {
	t.Run("test basic timeout", func(t *testing.T) {
		t.Run("should run callback fn", func(t *testing.T) {
			var atomicBool atomic.Bool
			start := time.Now()
			atomicBool.Store(false)
			tm := New(time.Second, func() {
				atomicBool.Store(true)
			})
			tm.Start()
			tm.Wait()
			if !atomicBool.Load() {
				t.Fatalf("time elapse should be at least 2 seconds")
			}
			if time.Since(start) < time.Second {
				t.Fatalf("fn isn't run")
			}
		})
		t.Run("should reset timer when reset is triggered", func(t *testing.T) {
			start := time.Now()
			var atomicBool atomic.Bool
			atomicBool.Store(false)
			tm := New(time.Second, func() {
				atomicBool.Store(true)
			})
			tm.Start()
			time.Sleep(time.Nanosecond * 500)
			tm.Reset()
			tm.Wait()
			if time.Since(start) < (time.Second + time.Nanosecond*500) {
				t.Fatalf("time elapse should be at least 2 seconds")
			}
			if !atomicBool.Load() {
				t.Fatalf("fn isn't run")
			}
		})
		t.Run("stop should stop a timer", func(t *testing.T) {
			start := time.Now()
			var atomicInt atomic.Int32
			atomicInt.Store(0)
			tm := New(time.Nanosecond*500, func() {
				atomicInt.Add(1)
			})
			tm.Repeat()
			tm.Wait()
			if time.Since(start) < time.Nanosecond*500 {
				t.Fatalf("time elapse should be at least 2 seconds")
			}
			if time.Since(start) > time.Nanosecond*900 {
				t.Fatalf("time elapse should be at least 2 seconds")
			}
			time.Sleep(time.Second + time.Nanosecond*200)
			tm.Stop()
			time.Sleep(time.Nanosecond * 1000)
			if atomicInt.Load() != int32(time.Second/(500*time.Nanosecond)) {
				t.Fatalf("fn isn't run with correct frequencie")
			}
		})
	})
}
