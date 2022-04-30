package gr_context

import (
	"github.com/dlshle/gommon/async"
	"testing"
)

func TestContext(t *testing.T) {
	printCtx := func() {
		t.Logf(Get("1").(string))
	}
	pool := async.NewAsyncPool("test", 5, 5)
	waitQueue := make([]async.Future, 5, 5)
	for i := 0; i < 5; i++ {
		fut := async.Run(func() {
			Put("1", "1")
			printCtx()
			Delete("1")
		}, pool)
		fut.Run()
		waitQueue[i] = fut
	}
	for _, fut := range waitQueue {
		fut.Wait()
	}
}
