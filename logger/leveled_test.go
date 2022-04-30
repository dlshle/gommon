package logger

import "testing"

func TestLeveled(t *testing.T) {
	l := StdOutLevelLogger("[test]")

	l.Info("hello")
	l.Info("hello", " oijfeiosjio04jt94")
	t.Fail()
}
