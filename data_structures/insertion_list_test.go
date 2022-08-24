package data_structures

import "testing"

func TestInsertionList(t *testing.T) {
	list := NewInsertionList(func(l int, r int) int {
		return l - r
	})
	list.Add(int(5))
	list.Add(int(7))
	list.Add(int(1))
	t.Log(list.AsSlice())
	if list.Remove(int(2)) {
		t.Fail()
	}
	if list.Find(int(2)) > -1 {
		t.Fail()
	}
	if !list.Remove(int(1)) {
		t.Fail()
	}
	t.Log(list.AsSlice())
	if !list.Remove(int(7)) {
		t.Fail()
	}
	if list.RemoveAt(10) {
		t.Fail()
	}
	if !list.Remove(int(5)) {
		t.Fail()
	}
	if list.Remove(int(2)) {
		t.Fail()
	}
	t.Log(list.AsSlice())
	list.Add(int(5))
	if !list.RemoveAt(0) {
		t.Fail()
	}
	t.Log(list.AsSlice())
}
