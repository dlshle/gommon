package Optional

import "testing"

func TestOptional(t *testing.T) {
	optional := Of("asd")
	val := optional.Transform(func(s string) string {
		return s + "!"
	}).OrElse("empty")
	t.Log(val)

	val = Of("").Transform(func(s string) string {
		return s + "!"
	}).OrElse("empty")
	t.Log(val)

	type Box struct {
		val string
	}
	val = Map(Of(Box{"asd"}), func(box Box) string {
		return box.val
	}).OrElse("empty")
	t.Log(val)
}
