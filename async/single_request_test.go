package async

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	testutils "github.com/dlshle/gommon/testutils"
)

func TestSingleRequest(t *testing.T) {
	requestGroup := NewRequestGroup()
	testutils.NewGroup("single request", "").Cases(
		testutils.NewWithDescription("basic request", "", func() {
			var counter atomic.Int32
			incr := func() (interface{}, error) {
				time.Sleep(time.Second)
				counter.Add(1)
				return int(counter.Load()), nil
			}
			var wg sync.WaitGroup
			for i := 0; i < 100; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					requestGroup.Do("incr", incr)
				}()
			}
			wg.Wait()
			testutils.AssertEquals(int(counter.Load()), 1)
		}),
		testutils.NewWithDescription("two continue requests", "", func() {
			var counter atomic.Int32
			incr := func() (interface{}, error) {
				time.Sleep(time.Second)
				counter.Add(1)
				return int(counter.Load()), nil
			}
			var wg sync.WaitGroup
			for i := 0; i < 100; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					requestGroup.Do("incr", incr)
				}()
			}
			wg.Wait()
			testutils.AssertEquals(int(counter.Load()), 1)

			for i := 0; i < 500; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					requestGroup.Do("incr", incr)
				}()
			}
			wg.Wait()
			testutils.AssertEquals(int(counter.Load()), 2)
		}),
		testutils.NewWithDescription("two separate request", "", func() {
			var counter atomic.Int32
			var counter1 atomic.Int32
			incr := func() (interface{}, error) {
				time.Sleep(time.Second)
				counter.Add(1)
				return int(counter.Load()), nil
			}
			incr1 := func() (interface{}, error) {
				time.Sleep(time.Second)
				counter1.Add(1)
				return int(counter1.Load()), nil
			}
			var wg sync.WaitGroup
			for i := 0; i < 100; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					requestGroup.Do("incr", incr)
				}()
			}
			for i := 0; i < 500; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					requestGroup.Do("incr1", incr1)
				}()
			}
			wg.Wait()
			testutils.AssertEquals(int(counter.Load()), 1)
			testutils.AssertEquals(int(counter1.Load()), 1)
		}),
	).Do(t)
}
