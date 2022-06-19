package data_structures

import (
	"github.com/dlshle/gommon/test_utils"
	"github.com/dlshle/gommon/utils"
	"testing"
)

func TestLinkedList(t *testing.T) {
	l := NewLinkedList[float32](true)
	tg := test_utils.NewTestGroup("LinkedList", "LinkedListTest")
	tg.Then("Append first", "head and tail should equal to 1", func() bool {
		l.Append(1)
		return l.Head() == 1 && l.Head() == l.Tail()
	}).Then("Append second", "tail == 2 and head == 1", func() bool {
		l.Append(2)
		return l.Tail() == 2 && l.Head() == 1
	}).Then("Prepend third", "head == 0 and tail == 2", func() bool {
		l.Prepend(0)
		return l.Head() == 0 && l.Tail() == 2
	}).Cases([]*test_utils.Assertion{
		test_utils.NewTestCase("Insert on last", "tail should be 3", func() bool {
			l.Insert(3, 3)
			return l.Head() == 0 && l.Tail() == 3
		}),
		test_utils.NewTestCase("Insert on 1 with 0.5", "l[1] == 0.5", func() bool {
			l.Insert(1, 0.5)
			return l.Get(1) == 0.5
		}),
		test_utils.NewTestCase("Get middle", "l[2] == 1", func() bool {
			return l.Get(2) == 1
		}),
		test_utils.NewTestCase("Remove 0.5(index 1)", "l[0] == 0, l[1] == 1, l[2] == 2", func() bool {
			l.Remove(1)
			return l.Get(0) == 0 && l.Get(1) == 1 && l.Get(2) == 2
		}),
	}).Concurrently("Concurrent write operations",
		"prepend -1~-3s, append 4~6s",
		func() {
			l.Prepend(-1)
		}, func() {
			l.Prepend(-2)
		}, func() {
			l.Append(4)
		}, func() {
			l.Prepend(-3)
		}, func() {
			l.Append(5)
		}, func() {
			l.Append(6)
		}).Then("Result after concurrent operation", "should contain 10 identical numbers with smallest = -3, largest = 6", func() bool {
		if l.Size() != 10 {
			t.Log("size != 10")
			return false
		}
		set := utils.TypedSliceToSet(l.ToSlice())
		t.Log("slice: ", set)
		if !(set[-3] && set[6]) {
			return false
		}
		return true
	}).Then("Poll and Pop", "Polled = -10, Popped = 10", func() bool {
		l.Prepend(-9)
		l.Prepend(-10)
		l.Append(9)
		l.Append(10)
		if l.Poll() != -10 {
			return false
		}
		if l.Pop() != 10 {
			return false
		}
		return l.Head() == -9 && l.Tail() == 9
	}).Do(t)
}
