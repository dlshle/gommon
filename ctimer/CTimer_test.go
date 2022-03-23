package ctimer

import (
	"testing"
	"time"
	"github.com/dlshle/gommon/test_utils"
)

func TestCTimer(t *testing.T) {
	test_utils.NewTestGroup("CTimer", "").Cases([]*test_utils.Assertion{
		test_utils.NewTestCase("simple timeout", "", func() bool {
			flag := 0
			New(time.Millisecond*500, func() {
				flag = 1
			}).Start()
			if flag > 0 {
				return false
			}
			time.Sleep(time.Millisecond * 500)
			return flag == 1
		}),
		test_utils.NewTestCase("simple reset", "", func() bool {
			flag := 0
			timer := New(time.Millisecond*500, func() {
				flag = 1
			})
			timer.Start()
			time.Sleep(time.Millisecond * 200)
			if flag > 0 {
				return false
			}
			timer.Reset()
			time.Sleep(time.Millisecond * 200)
			if flag != 0 {
				return false
			}
			time.Sleep(time.Millisecond * 310)
			return flag == 1
		}),
		test_utils.NewTestCase("multiple resets", "", func() bool {
			flag := 0
			timer := New(time.Millisecond*500, func() {
				flag = 1
			})
			timer.Start()
			time.Sleep(time.Millisecond * 200)
			if flag > 0 {
				return false
			}
			timer.Reset()
			time.Sleep(time.Millisecond * 200)
			if flag != 0 {
				return false
			}
			time.Sleep(time.Millisecond * 200)
			if flag != 0 {
				return false
			}
			timer.Reset()
			time.Sleep(time.Millisecond * 100)
			if flag != 0 {
				return false
			}
			timer.Reset()
			time.Sleep(time.Millisecond * 500)
			return flag == 1
		}),
		test_utils.NewTestCase("cancel test", "", func() bool {
			flag := 0
			timer := New(time.Millisecond*500, func() {
				flag = 1
			})
			timer.Start()
			time.Sleep(time.Millisecond * 200)
			timer.Cancel()
			time.Sleep(time.Millisecond * 400)
			return flag == 0
		}),
	}).Do(t)
}
