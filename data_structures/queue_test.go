package data_structures

import (
	"sync"
	"testing"

	"github.com/dlshle/gommon/test_utils"
)

func TestQueue(t *testing.T) {
	test_utils.NewGroup("queue", "").Cases(test_utils.New("sequential write and read", func() {
		counter := 0
		SIZE := 10
		q := NewSafeQueue[int]()
		for i := 0; i < SIZE; i++ {
			q.Enqueue(i)
		}
		for !q.IsEmpty() {
			test_utils.AssertEquals(counter, q.Dequeue())
			counter++
		}
		test_utils.AssertEquals(counter, SIZE)
	}), test_utils.New("concurrent write", func() {
		var wg sync.WaitGroup
		SIZE := 10
		q := NewSafeQueue[int]()
		for i := 0; i < SIZE; i++ {
			wg.Add(1)
			go func(n int) {
				q.Enqueue(n)
				wg.Done()
			}(i)
		}
		wg.Wait()
		test_utils.AssertEquals(q.Size(), SIZE)
	})).Do(t)
}
