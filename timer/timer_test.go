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
			t.Logf("start: %v", start)
			var atomicInt atomic.Int32
			atomicInt.Store(0)
			tm := New(time.Second, func() {
				atomicInt.Add(1)
				t.Logf("%v ran on %v", time.Now(), atomicInt.Load())
			})
			tm.Repeat()
			// wait should wait for at least 500 nano secs
			tm.Wait()
			if time.Since(start) < time.Second {
				t.Fatalf("time elapse should be at least 1 second")
			}
			if time.Since(start) > time.Second*4/3 {
				t.Fatalf("time elapse should be no longer than 2/3 seconds")
			}
			if atomicInt.Load() != 1 {
				t.Fatalf("callback is ran for more than once")
			}
			time.Sleep(2*time.Second + time.Millisecond*200)
			t.Logf("what now %v", time.Now())
			tm.Stop()
			time.Sleep(time.Millisecond * 100)
			if atomicInt.Load() != 3 {
				t.Fatalf("fn isn't run with correct frequencie")
			}
		})
	})
}
