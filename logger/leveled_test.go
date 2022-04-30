package logger

import (
	"sync"
	"testing"
)

func TestLeveled(t *testing.T) {
	l := StdOutLevelLogger("[test]")
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			Set("x", "y")
			Set("y", "z")
			l.Info("ctx test")
			Clear()
			wg.Done()
		}()
	}
	l.Info("hello")
	l.Info("hello", " oijfeiosjio04jt94")
	wg.Wait()
}
