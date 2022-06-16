package Optional

import "testing"

func TestOptional(t *testing.T) {
	optional := Of("asd")
	val := optional.Map(func(s string) string {
		return s + "!"
	}).OrElse("empty")
	t.Log(val)

	val = Of("").Map(func(s string) string {
		return s + "!"
	}).OrElse("empty")
	t.Log(val)
}
