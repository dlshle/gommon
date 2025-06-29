package async

import (
	"testing"
	"time"

	testutils "github.com/dlshle/gommon/testutils"
)

func TestSingleRequest(t *testing.T) {
	requestGroup := NewRequestGroup()
	testutils.NewGroup("single request", "").Cases(
		testutils.NewWithDescription("basic request", "", func() {
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
			testutils.AssertEquals(result.(int), counter)
		}),
		testutils.NewWithDescription("two continue requests", "", func() {
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
			testutils.AssertEquals(result.(int), counter)
		}),
		testutils.NewWithDescription("two separate request", "", func() {
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
			testutils.AssertEquals(counter, counter1)
		}),
	).Do(t)
}
