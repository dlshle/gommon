package async

import (
	"testing"
	"time"

	testutils "github.com/dlshle/gommon/testutils"
)

func TestBarrier(t *testing.T) {
	testutils.NewTestGroup("waitlock", "").Cases([]*testutils.Assertion{
		testutils.NewTestCase("lock and relock", "", func() bool {
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
