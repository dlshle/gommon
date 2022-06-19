package test_utils

import (
	"fmt"
	"github.com/dlshle/gommon/utils"
	"strings"
)

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

const (
	pnUnequalError = "assertion failure: "
)

func AssertEquals[T comparable](l T, r T) {
	if l != r {
		panic(pnUnequalError + fmt.Sprintf("%v and %v are not equal", l, r))
	}
}

func isAssertionFailurePanic(recovered interface{}) bool {
	if panicString, ok := recovered.(string); ok {
		return strings.HasPrefix(panicString, pnUnequalError)
		return true
	}
	return false
}
