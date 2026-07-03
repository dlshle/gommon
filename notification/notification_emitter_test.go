package notification

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dlshle/gommon/async"
	test_utils "github.com/dlshle/gommon/testutils"
)

func TestNotificationEmitter(t *testing.T) {
	emitter := New[string](10)
	test_utils.NewGroup("notification emitter", "notification emitter tests").Cases(
		test_utils.New("sync notification listeners", func() {
			var counter int32 = 0
			incrementCounter := func(s string) {
				atomic.AddInt32(&counter, 1)
			}
			disposer, err := emitter.On("test", incrementCounter)
			disposer1, err1 := emitter.On("test", incrementCounter)
			test_utils.AssertNil(err)
			test_utils.AssertNil(err1)
			test_utils.AssertTrue(emitter.HasEvent("test"))
			emitter.Notify("test", "hello")
			disposer1()
			emitter.Notify("test", "hello")
			disposer()
			emitter.Notify("test", "hello")
			test_utils.AssertEquals(counter, 3)
			_, err = emitter.Once("test", incrementCounter)
			test_utils.AssertNil(err)
			emitter.NotifyAsync("test", "hello", async.NewGoRoutineExecutor)
			time.Sleep(time.Second)
			emitter.Notify("test", "hello")
			test_utils.AssertEquals(counter, 4)
			test_utils.AssertFalse(emitter.HasEvent("test"))
			emitter.On("test", incrementCounter)
			emitter.OffAll("test")
			emitter.Notify("test", "hi")
			test_utils.AssertEquals(counter, 4)
			test_utils.AssertFalse(emitter.HasEvent("test"))
			emitter.Once("test", func(s string) {
				time.Sleep(time.Minute)
				incrementCounter(s)
			})
			test_utils.AssertEquals(counter, 4)
		}),
	).Do(t)
}

func TestNotificationEmitterConcurrency(t *testing.T) {
	test_utils.NewGroup("notification emitter concurrency", "notification emitter concurrency tests").Cases(
		test_utils.New("once fires exactly once under concurrency", func() {
			emitter := New[int](10)
			var counter int32 = 0
			onceListener := func(i int) {
				atomic.AddInt32(&counter, 1)
			}
			_, err := emitter.Once("event", onceListener)
			test_utils.AssertNil(err)

			var wg sync.WaitGroup
			for i := 0; i < 100; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					emitter.Notify("event", 1)
				}()
			}
			wg.Wait()
			test_utils.AssertEquals(atomic.LoadInt32(&counter), int32(1))
		}),

		test_utils.New("off removes the correct listener", func() {
			emitter := New[string](10)
			var counter0, counter1, counter2 int32
			listener0 := func(s string) { atomic.AddInt32(&counter0, 1) }
			listener1 := func(s string) { atomic.AddInt32(&counter1, 1) }
			listener2 := func(s string) { atomic.AddInt32(&counter2, 1) }
			listeners := []EventListener[string]{listener0, listener1, listener2}
			for _, l := range listeners {
				_, err := emitter.On("event", l)
				test_utils.AssertNil(err)
			}
			emitter.Off("event", listener1)
			emitter.Notify("event", "hi")
			test_utils.AssertEquals(atomic.LoadInt32(&counter0), int32(1))
			test_utils.AssertEquals(atomic.LoadInt32(&counter1), int32(0))
			test_utils.AssertEquals(atomic.LoadInt32(&counter2), int32(1))
		}),

		test_utils.New("notify does not deadlock on listener panic", func() {
			emitter := New[string](10)
			panicListener := func(s string) {
				panic("intentional")
			}
			normalListener := func(s string) {}
			_, err := emitter.On("event", panicListener)
			test_utils.AssertNil(err)
			_, err = emitter.On("event", normalListener)
			test_utils.AssertNil(err)

			done := make(chan struct{})
			go func() {
				emitter.Notify("event", "hi")
				close(done)
			}()
			select {
			case <-done:
				test_utils.AssertTrue(true)
			case <-time.After(time.Second):
				test_utils.AssertTrue(false)
			}
		}),

		test_utils.New("notify async invokes all listeners", func() {
			emitter := New[string](10)
			var counter int32 = 0
			for i := 0; i < 5; i++ {
				_, err := emitter.On("event", func(s string) {
					atomic.AddInt32(&counter, 1)
				})
				test_utils.AssertNil(err)
			}
			emitter.NotifyAsync("event", "hi", async.NewGoRoutineExecutor)
			time.Sleep(500 * time.Millisecond)
			test_utils.AssertEquals(atomic.LoadInt32(&counter), int32(5))
		}),

		test_utils.New("concurrent on and off", func() {
			emitter := New[int](256)
			listener := func(i int) {}
			var wg sync.WaitGroup
			for i := 0; i < 100; i++ {
				wg.Add(2)
				go func() {
					defer wg.Done()
					emitter.On("event", listener)
				}()
				go func() {
					defer wg.Done()
					emitter.Off("event", listener)
				}()
			}
			wg.Wait()
			test_utils.AssertTrue(true)
		}),
	).Do(t)
}
