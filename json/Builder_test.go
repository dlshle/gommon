package json

import "testing"

func TestBuilder(t *testing.T) {
	subBuilder := NewJsonBuilder()
	subMap := subBuilder.Field("hello", true).Field("world", 123).Field("say", "myName").BuildMap()
	builder := NewJsonBuilder()
	json, _ := builder.Field("sub", subMap).Field("number", 321).BuildString()
	t.Log(json)
}
