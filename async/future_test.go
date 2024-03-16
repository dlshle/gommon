package async

import (
	"testing"
	"time"

	"github.com/dlshle/gommon/errors"
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
			}, NewGoRoutineExecutor).Then(flipper).Then(flipper).Then(flipper).Then(flipper).MustGet().(bool)
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
			}).Wait()
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
			}, pool).Then(incrAndReturnCounter).Then(incrAndReturnCounter).Then(incrAndReturnCounter).Then(incrAndReturnCounter).Wait()
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
			}).OnPanic(incrAndReturnErrCounter).Then(incrAndReturnCounter).OnPanic(incrAndReturnErrCounter).Then(incrAndReturnCounter).Wait()
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
			})
			f.Cancel()
			t.Logf("started worker: %d", pool.NumStartedWorkers())
			return counter == 0
		}).WithMultiple(10, true).(*test_utils.Assertion),
		test_utils.NewTestCase("async pool executor with promised future", "", func() bool {
			start := time.Now()
			t.Logf("started promised future")
			return From(func(ra ResultAcceptor, ea ErrorAcceptor) {
				go func() {
					time.Sleep(1 * time.Second)
					ra(1)
				}()
			}).Then(func(i interface{}) interface{} {
				if i.(int) != 1 {
					panic("error return from prev future! expecting 1")
				}
				time.Sleep(1 * time.Second)
				return 2
			}).Then(func(i interface{}) interface{} {
				if i.(int) != 2 {
					panic("error return from prev future! expecting 2")
				}
				return time.Since(start) > time.Second*2
			}).ThenWithExecutor(func(i interface{}) interface{} {
				test_utils.AssertTrue(i.(bool))
				return true
			}, NewAsyncPool("", 128, 64)).MustGet().(bool)
		}).WithMultiple(10, true).(*test_utils.Assertion),
		test_utils.NewTestCase("error catching propogation", "", func() bool {
			mappedErr, err := From(func(ra ResultAcceptor, ea ErrorAcceptor) {
				go func() {
					time.Sleep(1 * time.Second)
					ea(errors.Error("mock error"))
				}()
			}).Then(func(i interface{}) interface{} {
				test_utils.AssertEquals("failed", "first")
				return nil
			}).Then(func(i interface{}) interface{} {
				test_utils.AssertEquals("failed", "second")
				return nil
			}).OnError(func(err error) {
				test_utils.AssertEquals(err.Error(), "mock error")
			}).Get()
			test_utils.AssertEquals(err.Error(), "mock error")
			test_utils.AssertNil(mappedErr)
			return true
		}),
		test_utils.NewTestCase("error catching propogation with mapping", "", func() bool {
			mappedErr, err := From(func(ra ResultAcceptor, ea ErrorAcceptor) {
				go func() {
					time.Sleep(1 * time.Second)
					ea(errors.Error("mock error"))
				}()
			}).Then(func(i interface{}) interface{} {
				test_utils.AssertEquals("failed", "first")
				return nil
			}).Then(func(i interface{}) interface{} {
				test_utils.AssertEquals("failed", "second")
				return nil
			}).OnError(func(err error) {
				// error handler should process with this fn iff no MapError is invoked
				test_utils.AssertEquals("failed", "shuoldn't come here since error is mapped")
			}).MapError(func(err error) interface{} {
				test_utils.AssertEquals(err.Error(), "mock error")
				return err
			}).Get()
			test_utils.AssertEquals(err.Error(), "mock error")
			casted, ok := mappedErr.(error)
			test_utils.AssertTrue(ok)
			test_utils.AssertEquals(casted.Error(), "mock error")
			return true
		}),
		test_utils.NewTestCase("panic catching propogation", "", func() bool {
			mappedErr, err := newAsyncTaskFuture(func() {
				panic(1)
			}, NewGoRoutineExecutor).Then(func(i interface{}) interface{} {
				test_utils.AssertEquals("failed", "first")
				return nil
			}).Then(func(i interface{}) interface{} {
				test_utils.AssertEquals("failed", "second")
				return nil
			}).OnPanic(func(err interface{}) {
				casted, ok := err.(int)
				test_utils.AssertTrue(ok)
				test_utils.AssertEquals(casted, 1)
			}).Get()
			test_utils.AssertNil(mappedErr)
			test_utils.AssertNil(err)
			return true
		}),
		test_utils.NewTestCase("pani catching propogation with mapping", "", func() bool {
			mappedErr, err := newAsyncTaskFuture(func() {
				panic(1)
			}, NewGoRoutineExecutor).Then(func(i interface{}) interface{} {
				test_utils.AssertEquals("failed", "first")
				return nil
			}).Then(func(i interface{}) interface{} {
				test_utils.AssertEquals("failed", "second")
				return nil
			}).OnError(func(err error) {
				test_utils.AssertEquals("failed", "shouldn't come here")
			}).MapPanic(func(err interface{}) interface{} {
				casted, ok := err.(int)
				test_utils.AssertTrue(ok)
				test_utils.AssertEquals(casted, 1)
				return casted + 1
			}).Get()
			test_utils.AssertNil(err)
			casted, ok := mappedErr.(int)
			test_utils.AssertTrue(ok)
			test_utils.AssertEquals(casted, 2)
			return true
		}),
	}).Do(t)
}
