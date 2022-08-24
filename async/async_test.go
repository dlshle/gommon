package async

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dlshle/gommon/test_utils"
)

func TestAsyncPool(t *testing.T) {
	pool := NewAsyncPool("test", 10, 5)
	pool.Verbose(true)
	test_utils.NewGroup("asyncPool", "").Cases(
		test_utils.New("basic scheduling", func() {
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
			test_utils.AssertTrue(computedValue.Get().(bool))
			t.Logf("num started go routines: %d", pool.NumGoroutineInitiated())
		}),
		test_utils.New("multiple scheduling", func() {
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
			test_utils.AssertEquals(intVal, 100)
			t.Logf("num started go routines: %d", pool.NumGoroutineInitiated())
		}),
		test_utils.New("stop and schedule", func() {
			var someVal int32 = 0
			pool.Stop()
			defer func() {
				recovered := recover()
				test_utils.AssertNonNil(recovered)
				test_utils.AssertEquals(someVal, 0)
				t.Logf("num started go routines: %d", pool.NumGoroutineInitiated())
			}()
			t.Logf("num started go routines: %d", pool.NumGoroutineInitiated())
			pool.Schedule(func() {
				atomic.AddInt32(&someVal, 1)
			})
		})).Do(t)
}
