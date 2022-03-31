package stringz

import (
	"fmt"
	"github.com/dlshle/gommon/performance"
	"github.com/dlshle/gommon/test_utils"
	"strconv"
	"testing"
)

func TestUtil(t *testing.T) {
	doSilientTest := true
	test_utils.NewTestGroup("stringz util", "").Cases([]*test_utils.Assertion{
		test_utils.NewTestCase("benchmark stringz with others", "", func() bool {
			stringzResult := performance.Measure(func() {
				builder := Builder()
				for i := 0; i < 1000; i++ {
					builder.String("a").Byte('b').Int(3)
				}
				builder.Build()
			})
			stringConcatResult := performance.Measure(func() {
				str := ""
				for i := 0; i < 1000; i++ {
					str += "a" + string('b') + strconv.Itoa(3)
				}
			})
			sprintfResult := performance.Measure(func() {
				str := ""
				for i := 0; i < 1000; i++ {
					str = fmt.Sprintf("%s%s%b%d", str, "a", 'b', 3)
				}
			})
			// t.Logf("builderTime: %d, concatTime: %d, sprintfTime: %d", stringzResult.Nanoseconds(), stringConcatResult.Nanoseconds(), sprintfResult.Nanoseconds())
			if !doSilientTest {
				if stringzResult < stringConcatResult {
					t.Logf("stringz < stringConcat by %d", stringConcatResult-stringzResult)
				}
				if stringzResult < sprintfResult {
					t.Logf("stringz < sprint by %d", stringConcatResult-sprintfResult)
				}
			}
			return stringzResult < stringConcatResult && stringzResult < sprintfResult
		}).WithMultiple(1000, false).NoAssertionLog().(*test_utils.Assertion),
		test_utils.NewTestCase("short write benchmark", "", func() bool {
			stringzResult := performance.Measure(func() {
				builder := Builder()
				for i := 0; i < 10; i++ {
					builder.String("a").Byte('b').Int(3)
				}
				builder.Build()
			})
			stringConcatResult := performance.Measure(func() {
				str := ""
				for i := 0; i < 10; i++ {
					str += "a" + string('b') + strconv.Itoa(3)
				}
			})
			sprintfResult := performance.Measure(func() {
				str := ""
				for i := 0; i < 10; i++ {
					str = fmt.Sprintf("%s%s%b%d", str, "a", 'b', 3)
				}
			})
			if !doSilientTest {
				if stringzResult < stringConcatResult {
					t.Logf("stringz < stringConcat by %d", stringConcatResult-stringzResult)
				}
				if stringzResult < sprintfResult {
					t.Logf("stringz < sprint by %d", stringConcatResult-sprintfResult)
				}
			}
			// t.Logf("builderTime: %d, concatTime: %d, sprintfTime: %d", stringzResult.Nanoseconds(), stringConcatResult.Nanoseconds(), sprintfResult.Nanoseconds())
			return stringzResult < stringConcatResult && stringzResult < sprintfResult
		}).WithMultiple(1000, false).NoAssertionLog().(*test_utils.Assertion),
	}).Do(t)
}
