package json

import (
	"github.com/dlshle/gommon/test_utils"
	"testing"
)

func TestJsonMapper(t *testing.T) {
	test_utils.NewTestGroup("", "").Cases([]*test_utils.Assertion{
		test_utils.NewTestCase("", "", func() bool {
			m := NewJSONMapper()
			m.Field("hello", "world").Field("a", true).Field("b", uint32(321423)).Field("c", 123213.342304).FloatPrecision(1).Field("omitted", nil).OmitNilValue(true)
			mm := NewJSONMapper()
			str := mm.Field("o", m).Field("ending", true).Field("someNil", nil).Field("omitted", "").OmitEmptyStringValue(true).ToString()
			t.Logf(str)
			return false
		}),
	}).Do(t)
}
