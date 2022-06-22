package test_utils

import (
	"fmt"
	"strings"

	"github.com/dlshle/gommon/utils"
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
	assertionFailureError = "assertion failure: "
)

func AssertNil(val interface{}) {
	if val != nil {
		panic(assertionFailureError + fmt.Sprintf("value %v isn't nil", val))
	}
}

func AssertNonNil(val interface{}) {
	if val == nil {
		panic(assertionFailureError + fmt.Sprintf("value %v isn nil", val))
	}
}

func AssertStringEmpty(val string) {
	AssertEquals(val, "")
}

func AssertTrue(val bool) {
	if !val {
		panic(assertionFailureError + "value isn't true")
	}
}

func AssertFalse(val bool) {
	if val {
		panic(assertionFailureError + "value isn't false")
	}
}

func AssertPanic(cb func()) {
	defer func() {
		recovered := recover()
		if recovered == nil {
			panic(assertionFailureError + "no panic value is recovered")
		}
	}()
	cb()
}

func AssertPanicValue[T comparable](cb func(), expected T) {
	defer func() {
		recovered := recover()
		if recovered == nil {
			panic(assertionFailureError + "no panic value is recovered")
		}
		converted, ok := recovered.(T)
		if !ok {
			panic(assertionFailureError + "unable to cast recovered panic value to the expected type")
		}
		AssertEquals(converted, expected)
	}()
	cb()
}

func AssertEquals[T comparable](l T, r T) {
	if l != r {
		panic(assertionFailureError + fmt.Sprintf("%v and %v are not equal", l, r))
	}
}

func isAssertionFailurePanic(recovered interface{}) bool {
	if panicString, ok := recovered.(string); ok {
		return strings.HasPrefix(panicString, assertionFailureError)
	}
	return false
}
