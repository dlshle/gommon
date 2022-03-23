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
	test_utils.NewTestGroup("asyncPool", "").Cases([]*test_utils.Assertion{
		test_utils.NewTestCase("basic scheduling", "", func() bool {
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
			return computedValue.Get().(bool)
		}),
		test_utils.NewTestCase("multiple scheduling", "", func() bool {
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
			return intVal == 100
		}),
	}).Do(t)
	t.Logf("num goroutines: %d", pool.NumGoroutineInitiated())
}
