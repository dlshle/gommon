package test_utils

import "github.com/dlshle/gommon/utils"

func AssertSlicesEqual(l []interface{}, r []interface{}) bool {
	if len(l) != len(r) {
		return false
	}
	for i := range l {
		if l[i] != r[i] {
			return false
		}
	}
	return true
}

func AssertUnOrderedSlicesEqual(l []interface{}, r []interface{}) bool {
	return AssertSetsEqual(utils.SliceToSet(l), utils.SliceToSet(r))
}

func AssertSetsEqual(l map[interface{}]bool, r map[interface{}]bool) bool {
	return len(utils.SetIntersections(l, r)) == 0
}
