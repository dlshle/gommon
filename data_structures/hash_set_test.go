package data_structures

import (
	"sync"
	"testing"
	"time"

	test_utils "github.com/dlshle/gommon/testutils"
)

func TestSet(t *testing.T) {
	test_utils.NewGroup("set", "set tests").Cases(test_utils.New("generic", func() {
		set := NewSet[int]()
		set.Add(1)
		set.Add(2)
		test_utils.AssertEquals(set.Size(), 2)
	}), test_utils.New("concurrency", func() {
		safeSet := NewSafeSet[int]()
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			safeSet.Add(2)
			safeSet.Add(3)
			safeSet.Add(5)
			wg.Done()
		}()
		go func() {
			time.Sleep(time.Millisecond * 3)
			safeSet.Delete(2)
			safeSet.Delete(3)
			safeSet.Delete(5)
			wg.Done()
		}()
		wg.Wait()
		test_utils.AssertEquals(safeSet.Size(), 0)
	})).Do(t)
}
