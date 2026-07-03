package async

import (
	"sync/atomic"
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
			isOpen := atomic.Bool{}
			go func() {
				b.Wait()
				isOpen.Store(true)
			}()
			time.Sleep(time.Millisecond * 1)
			if isOpen.Load() {
				return false
			}
			b.Open()
			time.Sleep(time.Millisecond * 1)
			if !isOpen.Load() {
				return false
			}
			b.Lock()
			isOpen.Store(false)
			go func() {
				b.Wait()
				isOpen.Store(true)
			}()
			time.Sleep(time.Millisecond * 1)
			if isOpen.Load() {
				return false
			}
			b.Open()
			time.Sleep(time.Millisecond * 1)
			return isOpen.Load()
		}),
	}).Do(t)
}
