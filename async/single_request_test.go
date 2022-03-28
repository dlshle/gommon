package async

import (
	"github.com/dlshle/gommon/test_utils"
	"testing"
	"time"
)

func TestSingleRequest(t *testing.T) {
	requestGroup := NewRequestGroup()
	test_utils.NewTestGroup("single request", "").Cases([]*test_utils.Assertion{
		test_utils.NewTestCase("basic request", "", func() bool {
			counter := 0
			incr := func() (interface{}, error) {
				time.Sleep(time.Second)
				counter++
				return counter, nil
			}
			for i := 0; i < 100; i++ {
				go func() {
					requestGroup.Do("incr", incr)
				}()
			}
			result, _ := requestGroup.Do("incr", incr)
			t.Logf("result: %d, counter: %d", result, counter)
			return result == counter && counter == 1
		}),
		test_utils.NewTestCase("two continue requests", "", func() bool {
			counter := 0
			incr := func() (interface{}, error) {
				time.Sleep(time.Second)
				counter++
				return counter, nil
			}
			for i := 0; i < 100; i++ {
				go func() {
					requestGroup.Do("incr", incr)
				}()
			}
			requestGroup.Do("incr", incr)
			for i := 0; i < 500; i++ {
				go func() {
					requestGroup.Do("incr", incr)
				}()
			}
			result, _ := requestGroup.Do("incr", incr)
			t.Logf("result: %d, counter: %d", result, counter)
			return result == counter && counter == 2
		}),
		test_utils.NewTestCase("two separate request", "", func() bool {
			counter := 0
			counter1 := 0
			incr := func() (interface{}, error) {
				time.Sleep(time.Second)
				counter++
				return counter, nil
			}
			incr1 := func() (interface{}, error) {
				time.Sleep(time.Second)
				counter1++
				return counter1, nil
			}
			for i := 0; i < 100; i++ {
				go func() {
					requestGroup.Do("incr", incr)
				}()
			}
			for i := 0; i < 500; i++ {
				go func() {
					requestGroup.Do("incr1", incr1)
				}()
			}
			requestGroup.Do("incr", incr)
			t.Logf("counter: %d, counter1: %d", counter, counter1)
			return counter == counter1 && counter == 1
		}),
	}).Do(t)
}
