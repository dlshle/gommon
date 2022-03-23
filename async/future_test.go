package async

import (
	"testing"
	"time"
	"github.com/dlshle/gommon/test_utils"
)

func TestFuture(t *testing.T) {
	test_utils.NewTestGroup("Future", "future library test").Cases([]*test_utils.Assertion{
		test_utils.NewTestCase("direct executor chain", "", func() bool {
			flipper := func(input interface{}) interface{} {
				return !input.(bool)
			}
			return NewComputedFuture(func() interface{} {
				return true
			}, NewGoRoutineExecutor).Then(flipper).Then(flipper).Then(flipper).Then(flipper).Run().Get().(bool)
		}),
		test_utils.NewTestCase("new goroutine executor chain with panic", "", func() bool {
			var errMsg string
			NewComputedFuture(func() interface{} {
				panic("err")
			}, NewGoRoutineExecutor).Then(func(_ interface{}) interface{} {
				return nil
			}).Then(func(_ interface{}) interface{} {
				return nil
			}).OnPanic(func(panicEntity interface{}) {
				errMsg = panicEntity.(string)
			}).Run().Wait()
			return errMsg == "err"
		}),
		test_utils.NewTestCase("async pool executor single chain with no panic", "", func() bool {
			counter := 0
			pool := NewAsyncPool("test", 128, 16)
			incrAndReturnCounter := func(prev interface{}) interface{} {
				counter = prev.(int) + 1
				return counter
			}
			NewComputedFuture(func() interface{} {
				counter++
				return counter
			}, pool).Then(incrAndReturnCounter).Then(incrAndReturnCounter).Then(incrAndReturnCounter).Then(incrAndReturnCounter).Run().Wait()
			t.Logf("started worker: %d", pool.NumStartedWorkers())
			return counter == 5
		}),
		test_utils.NewTestCase("async pool executor single chain with multiple panic", "", func() bool {
			counter := 0
			errCounter := 0
			pool := NewAsyncPool("test", 128, 16)
			incrAndReturnCounter := func(prev interface{}) interface{} {
				counter = prev.(int) + 1
				return counter
			}
			incrAndReturnErrCounter := func(interface{}) {
				errCounter++
			}
			NewComputedFuture(func() interface{} {
				counter++
				return counter
			}, pool).Then(incrAndReturnCounter).Then(func(interface{}) interface{} {
				panic("err")
			}).OnPanic(incrAndReturnErrCounter).Then(incrAndReturnCounter).OnPanic(incrAndReturnErrCounter).Then(incrAndReturnCounter).Run().Wait()
			t.Logf("started worker: %d", pool.NumStartedWorkers())
			return counter == 2 && errCounter == 2
		}),
		test_utils.NewTestCase("async pool executor single chain with multiple panic and cancellation", "", func() bool {
			counter := 0
			pool := NewAsyncPool("test", 128, 16)
			incrAndReturnCounter := func(prev interface{}) interface{} {
				counter = prev.(int) + 1
				return counter
			}
			f := NewComputedFuture(func() interface{} {
				time.Sleep(time.Second * 3)
				counter++
				return counter
			}, pool).Then(incrAndReturnCounter)
			f.Then(incrAndReturnCounter).OnPanic(func(err interface{}) {
				t.Logf("%v", err)
			}).Run()
			f.Cancel()
			t.Logf("started worker: %d", pool.NumStartedWorkers())
			return counter == 0
		}).WithMultiple(10, true).(*test_utils.Assertion),
	}).Do(t)
}
