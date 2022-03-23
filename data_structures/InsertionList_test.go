package data_structures

import "testing"

type ComparableInt int

func (c ComparableInt) Compare(comp IComparable) int {
	ccomp := int(comp.(ComparableInt))
	return int(c) - ccomp
}

func TestInsertionList(t *testing.T) {
	list := NewInsertionList()
	list.Add(ComparableInt(5))
	list.Add(ComparableInt(7))
	list.Add(ComparableInt(1))
	t.Log(list.AsSlice())
	if list.Remove(ComparableInt(2)) {
		t.Fail()
	}
	if list.Find(ComparableInt(2)) > -1 {
		t.Fail()
	}
	if !list.Remove(ComparableInt(1)) {
		t.Fail()
	}
	t.Log(list.AsSlice())
	if !list.Remove(ComparableInt(7)) {
		t.Fail()
	}
	if list.RemoveAt(10) {
		t.Fail()
	}
	if !list.Remove(ComparableInt(5)) {
		t.Fail()
	}
	if list.Remove(ComparableInt(2)) {
		t.Fail()
	}
	t.Log(list.AsSlice())
	list.Add(ComparableInt(5))
	if !list.RemoveAt(0) {
		t.Fail()
	}
	t.Log(list.AsSlice())
}
