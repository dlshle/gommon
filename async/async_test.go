package async

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	testutils "github.com/dlshle/gommon/testutils"
)

func TestAsyncPool(t *testing.T) {
	pool := NewAsyncPool("test", 10, 5)
	pool.Verbose(true)
	testutils.NewGroup("asyncPool", "").Cases(
		testutils.New("basic scheduling", func() {
			b := NewStatefulBarrier()
			go func() {
				time.Sleep(time.Second)
				b.OpenWith(false)
			}()
			pool.Schedule(func() {
				b.OpenWith(true)
			})
			computedValue := pool.ScheduleComputable(func() interface{} {
				return b.Get()
			})
			testutils.AssertTrue(computedValue.Get().(bool))
			t.Logf("num started go routines: %d", pool.NumGoroutineInitiated())
		}),
		testutils.New("multiple scheduling", func() {
			var intVal int32 = 0
			var wg sync.WaitGroup
			for i := 0; i < 100; i++ {
				wg.Add(1)
				pool.Schedule(func() {
					atomic.AddInt32(&intVal, 1)
					wg.Done()
				})
			}
			wg.Wait()
			testutils.AssertEquals(intVal, 100)
			t.Logf("num started go routines: %d", pool.NumGoroutineInitiated())
		}),
		testutils.New("stop and schedule", func() {
			var someVal int32 = 0
			pool.Stop()
			defer func() {
				recovered := recover()
				testutils.AssertNonNil(recovered)
				testutils.AssertEquals(someVal, 0)
				t.Logf("num started go routines: %d", pool.NumGoroutineInitiated())
			}()
			t.Logf("num started go routines: %d", pool.NumGoroutineInitiated())
			pool.Schedule(func() {
				atomic.AddInt32(&someVal, 1)
			})
		})).Do(t)
}
