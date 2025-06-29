package notification

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/dlshle/gommon/async"
	test_utils "github.com/dlshle/gommon/testutils"
)

func TestNotificationEmitter(t *testing.T) {
	emitter := New[string](10)
	test_utils.NewGroup("notification emitter", "notification emitter tests").Cases(
		test_utils.New("sync notification listeners", func() {
			var counter int32 = 0
			incrementCounter := func(s string) {
				atomic.AddInt32(&counter, 1)
			}
			disposer, err := emitter.On("test", incrementCounter)
			disposer1, err1 := emitter.On("test", incrementCounter)
			test_utils.AssertNil(err)
			test_utils.AssertNil(err1)
			test_utils.AssertTrue(emitter.HasEvent("test"))
			emitter.Notify("test", "hello")
			disposer1()
			emitter.Notify("test", "hello")
			disposer()
			emitter.Notify("test", "hello")
			test_utils.AssertEquals(counter, 3)
			_, err = emitter.Once("test", incrementCounter)
			test_utils.AssertNil(err)
			emitter.NotifyAsync("test", "hello", async.NewGoRoutineExecutor)
			time.Sleep(time.Second)
			emitter.Notify("test", "hello")
			test_utils.AssertEquals(counter, 4)
			test_utils.AssertFalse(emitter.HasEvent("test"))
			emitter.On("test", incrementCounter)
			emitter.OffAll("test")
			emitter.Notify("test", "hi")
			test_utils.AssertEquals(counter, 4)
			test_utils.AssertFalse(emitter.HasEvent("test"))
			emitter.Once("test", func(s string) {
				time.Sleep(time.Minute)
				incrementCounter(s)
			})
			test_utils.AssertEquals(counter, 4)
		}),
	).Do(t)
}
