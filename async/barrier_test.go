package async

import (
	"testing"
	"time"
	"github.com/dlshle/gommon/test_utils"
)

func TestBarrier(t *testing.T) {
	test_utils.NewTestGroup("waitlock", "").Cases([]*test_utils.Assertion{
		test_utils.NewTestCase("lock and relock", "", func() bool {
			b := NewWaitLock()
			if b.IsOpen() {
				return false
			}
			isOpen := false
			go func() {
				b.Wait()
				isOpen = true
			}()
			time.Sleep(time.Millisecond * 1)
			if isOpen {
				return false
			}
			b.Open()
			time.Sleep(time.Millisecond * 1)
			if !isOpen {
				return false
			}
			b.Lock()
			isOpen = false
			go func() {
				b.Wait()
				isOpen = true
			}()
			time.Sleep(time.Millisecond * 1)
			if isOpen {
				return false
			}
			b.Open()
			time.Sleep(time.Millisecond * 1)
			return isOpen
		}),
	}).Do(t)
}
