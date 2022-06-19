package test_utils

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorWhite  = "\033[37m"
)

type assertion struct {
	head                  *assertion
	id                    string
	description           string
	assertion             func()
	shouldAssert          bool
	next                  *assertion
	numRuns               int
	runMultipleInParallel bool
	noAssertionLog        bool
}

type Assertable interface {
	Concurrently(id string, desc string, actions ...func()) Assertable              // next group
	ThenWithDescription(id string, description string, assertion func()) Assertable // next
	Then(id string, assertion func()) Assertable                                    // next
	Cases(cases ...*assertion) Assertable
	WithMultipleRuns(numRuns int, parallel bool) Assertable
	NoAssertionLog() Assertable
	Do(t *testing.T)
}

func New(id string, assertionCase func()) *assertion {
	return NewWithDescription(id, "", assertionCase)
}

func NewWithDescription(id string, description string, assertionCase func()) *assertion {
	a := &assertion{
		id:           id,
		description:  description,
		assertion:    assertionCase,
		shouldAssert: true,
	}
	a.head = a
	return a
}

func NewGroup(id string, description string) Assertable {
	a := &assertion{
		id:          id,
		description: description,
	}
	a.head = a
	return a
}

func (a *assertion) WithMultipleRuns(numRuns int, parallel bool) Assertable {
	if numRuns < 0 {
		numRuns = 1
	}
	a.numRuns = numRuns
	a.runMultipleInParallel = parallel
	return a
}

func (a *assertion) Concurrently(id string, desc string, actions ...func()) Assertable {
	actionFunc := func() {
		var wg sync.WaitGroup
		panics := make([]interface{}, len(actions), len(actions))
		hasPanic := false
		for i, act := range actions {
			wg.Add(1)
			go (func(action func(), i int) {
				defer func() {
					if recovered := recover(); recovered != nil {
						panics[i] = recovered
						hasPanic = true
					}
				}()
				action()
				wg.Done()
			})(act, i)
		}
		wg.Wait()
		if hasPanic {
			panic(panics)
		}
	}
	a.next = &assertion{
		head:         a.head,
		id:           id,
		description:  desc,
		assertion:    actionFunc,
		shouldAssert: false,
	}
	return a.next
}

func (a *assertion) Then(id string, assertionCase func()) Assertable {
	a.next = &assertion{
		head:         a.head,
		id:           id,
		assertion:    assertionCase,
		shouldAssert: true,
	}
	return a.next
}

func (a *assertion) ThenWithDescription(id string, description string, assertionCase func()) Assertable {
	a.next = &assertion{
		head:         a.head,
		id:           id,
		description:  description,
		assertion:    assertionCase,
		shouldAssert: true,
	}
	return a.next
}

func (a *assertion) NoAssertionLog() Assertable {
	a.noAssertionLog = true
	return a
}

func (a *assertion) Cases(cases ...*assertion) Assertable {
	curr := a
	for _, c := range cases {
		if c != nil {
			curr.next = c
			c.head = curr.head
			curr = c
		}
	}
	return curr
}

func getIndentations(level int) string {
	if level == 0 {
		return ""
	}
	builder := strings.Builder{}
	for level > 0 {
		builder.WriteByte(' ')
		level--
	}
	return builder.String()
}

func (a *assertion) Do(t *testing.T) {
	startTime := time.Now()
	curr := a.head
	indent := 0
	for curr != nil {
		if curr.shouldAssert {
			t.Logf("%sRunning case %s%s\n", getIndentations(indent), curr.id, getDescription(curr))
		} else {
			t.Logf("%sRunning operation %s%s\n", getIndentations(indent), curr.id, getDescription(curr))
		}
		if curr.assertion != nil {
			if curr.shouldAssert {
				a.doAssertion(t, indent, curr)
			} else {
				curr.assertion()
			}
		} else {
			indent += 2
		}
		curr = curr.next
	}
	t.Log("All test finished, overall runtime: ", time.Since(startTime))
}

func (a *assertion) doAssertion(t *testing.T, indent int, node *assertion) {
	runner := func(indent int) bool {
		if a.noAssertionLog {
			return doAssertCase(nil, indent, node.id, node.description, node.assertion)
		} else {
			return doAssertCase(t, indent, node.id, node.description, node.assertion)
		}
	}
	if node.numRuns > 0 {
		var runnerModeStr string
		var wg sync.WaitGroup
		inParallel := node.runMultipleInParallel
		var multiCaseSucceedCounter int32 = 0
		doRunner := func(indent int) {
			if inParallel {
				wg.Add(1)
				go func() {
					if runner(indent) {
						atomic.AddInt32(&multiCaseSucceedCounter, 1)
					}
					wg.Done()
				}()
			} else {
				if runner(indent) {
					multiCaseSucceedCounter++
				}
			}
		}
		if inParallel {
			runnerModeStr = "in parallel"
		} else {
			runnerModeStr = "in series"
		}
		t.Logf("%sRun case [%s] %s %d times", getIndentations(indent), node.id, runnerModeStr, node.numRuns)
		indent += 4
		for i := 0; i < node.numRuns; i++ {
			doRunner(indent)
		}
		if inParallel {
			wg.Wait()
		}
		indent -= 4
		if node.numRuns > 1 {
			t.Logf("%sMutiple case success rate report: (%d/%d = %g)",
				getIndentations(indent),
				multiCaseSucceedCounter,
				node.numRuns,
				float64(multiCaseSucceedCounter)/float64(node.numRuns))
		}
	} else {
		runner(indent)
	}
}

func doAssertCase(t *testing.T, indent int, id string, desc string, assertion func()) bool {
	errorMessage := ""
	res := true
	defer func() {
		if recovered := recover(); recovered != nil {
			res = false
			if isAssertionFailurePanic(recovered) {
				errorMessage = recovered.(string)
			} else {
				errorMessage = "panic recovered, call stack trace: \n" + getCallers()
			}
		}
		if t != nil {
			if res {
				t.Logf("%s✅ %s passed\n", getIndentations(indent), id)
			} else {
				t.Errorf("%s❌ %s failed\n", getIndentations(indent), id)
				t.Error(colorRed + errorMessage)
			}
		}
	}()
	assertion()
	return res
}

func getCallers() string {
	callers := ""
	for i := 0; true; i++ {
		_, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		callers = callers + fmt.Sprintf("%s%v:%v\n", getIndentations(i*2), file, line)
	}
	return callers
}

func getDescription(a *assertion) string {
	if a.description == "" {
		return ""
	}
	return "[" + a.description + "]"
}
