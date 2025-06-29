package async

import (
	"testing"
	"time"

	"github.com/dlshle/gommon/errors"
	testutils "github.com/dlshle/gommon/testutils"
)

func TestFuture(t *testing.T) {
	testutils.NewTestGroup("Futures", "future library test").Cases([]*testutils.Assertion{
		testutils.NewTestCase("direct executor chain", "", func() bool {
			flipper := func(input interface{}) (interface{}, error) {
				return !input.(bool), nil
			}
			return NewComputedFuture(func() interface{} {
				return true
			}, NewGoRoutineExecutor).Then(flipper).Then(flipper).Then(flipper).Then(flipper).MustGet().(bool)
		}),
		testutils.NewTestCase("new goroutine executor chain with panic", "", func() bool {
			var errMsg string
			NewComputedFuture(func() interface{} {
				panic("err")
			}, NewGoRoutineExecutor).Then(func(_ interface{}) (interface{}, error) {
				return nil, nil
			}).Then(func(_ interface{}) (interface{}, error) {
				return nil, nil
			}).OnPanic(func(panicEntity interface{}) {
				errMsg = panicEntity.(string)
			}).Wait()
			return errMsg == "err"
		}),
		testutils.NewTestCase("async pool executor single chain with no panic", "", func() bool {
			counter := 0
			pool := NewAsyncPool("test", 128, 16)
			incrAndReturnCounter := func(prev interface{}) (interface{}, error) {
				counter = prev.(int) + 1
				return counter, nil
			}
			NewComputedFuture(func() interface{} {
				counter++
				return counter
			}, pool).Then(incrAndReturnCounter).Then(incrAndReturnCounter).Then(incrAndReturnCounter).Then(incrAndReturnCounter).Wait()
			t.Logf("started worker: %d", pool.NumStartedWorkers())
			return counter == 5
		}),
		testutils.NewTestCase("async pool executor single chain with multiple panic", "", func() bool {
			counter := 0
			errCounter := 0
			pool := NewAsyncPool("test", 128, 16)
			incrAndReturnCounter := func(prev interface{}) (interface{}, error) {
				counter = prev.(int) + 1
				return counter, nil
			}
			incrAndReturnErrCounter := func(interface{}) {
				errCounter++
			}
			NewComputedFuture(func() interface{} {
				counter++
				return counter
			}, pool).Then(incrAndReturnCounter).Then(func(interface{}) (interface{}, error) {
				panic("err")
			}).OnPanic(incrAndReturnErrCounter).Then(incrAndReturnCounter).OnPanic(incrAndReturnErrCounter).Then(incrAndReturnCounter).Wait()
			t.Logf("started worker: %d", pool.NumStartedWorkers())
			return counter == 2 && errCounter == 2
		}),
		testutils.NewTestCase("async pool executor single chain with multiple panic and cancellation", "", func() bool {
			counter := 0
			pool := NewAsyncPool("test", 128, 16)
			incrAndReturnCounter := func(prev interface{}) (interface{}, error) {
				counter = prev.(int) + 1
				return counter, nil
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
		}).WithMultiple(10, true).(*testutils.Assertion),
		testutils.NewTestCase("async pool executor with promised future", "", func() bool {
			start := time.Now()
			t.Logf("started promised future")
			return From(func(ra ResultAcceptor, ea ErrorAcceptor) {
				go func() {
					time.Sleep(1 * time.Second)
					ra(1)
				}()
			}).Then(func(i interface{}) (interface{}, error) {
				if i.(int) != 1 {
					panic("error return from prev future! expecting 1")
				}
				time.Sleep(1 * time.Second)
				return 2, nil
			}).Then(func(i interface{}) (interface{}, error) {
				if i.(int) != 2 {
					panic("error return from prev future! expecting 2")
				}
				return time.Since(start) > time.Second*2, nil
			}).ThenWithExecutor(func(i interface{}) (interface{}, error) {
				testutils.AssertTrue(i.(bool))
				return true, nil
			}, NewAsyncPool("", 128, 64)).MustGet().(bool)
		}).WithMultiple(10, true).(*testutils.Assertion),
		testutils.NewTestCase("error catching propogation", "", func() bool {
			mappedErr, err := From(func(ra ResultAcceptor, ea ErrorAcceptor) {
				go func() {
					time.Sleep(1 * time.Second)
					ea(errors.Error("mock error"))
				}()
			}).Then(func(i interface{}) (interface{}, error) {
				testutils.AssertEquals("failed", "first")
				return nil, nil
			}).Then(func(i interface{}) (interface{}, error) {
				testutils.AssertEquals("failed", "second")
				return nil, nil
			}).OnError(func(err error) {
				testutils.AssertEquals(err.Error(), "mock error")
			}).Get()
			testutils.AssertEquals(err.Error(), "mock error")
			testutils.AssertNil(mappedErr)
			return true
		}),
		testutils.NewTestCase("error catching propogation with mapping", "", func() bool {
			mappedErr, err := From(func(ra ResultAcceptor, ea ErrorAcceptor) {
				go func() {
					time.Sleep(1 * time.Second)
					ea(errors.Error("mock error"))
				}()
			}).Then(func(i interface{}) (interface{}, error) {
				testutils.AssertEquals("failed", "first")
				return nil, nil
			}).Then(func(i interface{}) (interface{}, error) {
				testutils.AssertEquals("failed", "second")
				return nil, nil
			}).OnError(func(err error) {
				testutils.AssertNonNil(err)
			}).MapError(func(err error) interface{} {
				testutils.AssertEquals(err.Error(), "mock error")
				return "mapped:" + err.Error()
			}).Get()
			// since error is mapped, we expect result from mappedErr
			testutils.AssertNil(err)
			casted, ok := mappedErr.(string)
			testutils.AssertTrue(ok)
			testutils.AssertEquals(casted, "mapped:mock error")
			return true
		}),
		testutils.NewTestCase("panic catching propogation", "", func() bool {
			mappedErr, err := newAsyncTaskFuture(func() {
				panic(1)
			}, NewGoRoutineExecutor).Then(func(i interface{}) (interface{}, error) {
				testutils.AssertEquals("failed", "first")
				return nil, nil
			}).Then(func(i interface{}) (interface{}, error) {
				testutils.AssertEquals("failed", "second")
				return nil, nil
			}).OnPanic(func(err interface{}) {
				casted, ok := err.(int)
				testutils.AssertTrue(ok)
				testutils.AssertEquals(casted, 1)
			}).Get()
			testutils.AssertNil(mappedErr)
			testutils.AssertNil(err)
			return true
		}),
		testutils.NewTestCase("panic catching propogation with mapping", "", func() bool {
			mappedErr, err := newAsyncTaskFuture(func() {
				panic(1)
			}, NewGoRoutineExecutor).Then(func(i interface{}) (interface{}, error) {
				testutils.AssertEquals("failed", "first")
				return nil, nil
			}).Then(func(i interface{}) (interface{}, error) {
				testutils.AssertEquals("failed", "second")
				return nil, nil
			}).OnError(func(err error) {
				testutils.AssertNonNil(err)
			}).MapPanic(func(err interface{}) interface{} {
				casted, ok := err.(int)
				testutils.AssertTrue(ok)
				testutils.AssertEquals(casted, 1)
				return casted + 1
			}).Get()
			testutils.AssertNil(err)
			casted, ok := mappedErr.(int)
			testutils.AssertTrue(ok)
			testutils.AssertEquals(casted, 2)
			return true
		}),
		testutils.NewTestCase("promise chain", "", func() bool {
			res := From(func(ra ResultAcceptor, ea ErrorAcceptor) {
				ra(1)
			}).ThenAsync(func(i interface{}) (Future, error) {
				return From(func(ra ResultAcceptor, ea ErrorAcceptor) {
					go func() {
						num, ok := i.(int)
						if !ok {
							ea(errors.Error("error"))
							return
						}
						ra(num + 1)
					}()
				}), nil
			}).Then(func(i interface{}) (interface{}, error) {
				if num, ok := i.(int); !ok || num != 2 {
					return false, nil
				}
				return true, nil
			}).MustGet().(bool)
			testutils.AssertTrue(res)

			res1 := ImmediateFuture(1).ThenAsync(func(i interface{}) (Future, error) {
				return From(func(ra ResultAcceptor, ea ErrorAcceptor) {
					go func() {
						num, ok := i.(int)
						if !ok {
							ea(errors.Error("error"))
							return
						}
						ra(num + 1)
					}()
				}), nil
			}).ThenAsync(func(i interface{}) (Future, error) {
				return ImmediateFuture(i).Then(func(i interface{}) (interface{}, error) {
					return i.(int) + 1, nil
				}), nil
			}).MustGet().(int)
			testutils.AssertEquals(res1, 3)
			return true
		}),
		testutils.NewTestCase("multiple chainned futures", "", func() bool {
			f0_1 := From(func(ra ResultAcceptor, ea ErrorAcceptor) {
				time.Sleep(1 * time.Second)
				ra(1)
			})
			f1_2 := f0_1.Then(func(i interface{}) (interface{}, error) {
				return i.(int) + 1, nil
			})
			f2_3 := f0_1.Then(func(i interface{}) (interface{}, error) {
				return i.(int) + 2, nil
			})
			f3_err := f2_3.Then(func(i interface{}) (interface{}, error) {
				return nil, errors.Error("f3 error")
			})
			f3_1_err := f3_err.Then(func(i interface{}) (interface{}, error) {
				return i.(int) + 3, nil
			})
			_, f3_err_info := f3_err.Get()
			f3_1_res, f3_1_err_info := f3_1_err.Get()

			f0_1_r, f0_1_e := f0_1.Get()

			testutils.AssertEquals(f0_1.MustGet().(int), 1)
			testutils.AssertEquals(f1_2.MustGet().(int), 2)
			testutils.AssertEquals(f2_3.MustGet().(int), 3)
			testutils.AssertEquals(f3_err_info.Error(), "f3 error")
			testutils.AssertNil(f3_1_res)
			testutils.AssertEquals(f3_1_err_info.Error(), "f3 error")
			testutils.AssertEquals(f0_1_r.(int), 1)
			testutils.AssertNil(f0_1_e)
			return true
		}),
	}).Do(t)
}
