package json

import (
	"github.com/dlshle/gommon/test_utils"
	"testing"
)

type Box struct {
	s string
}

func (b Box) toJsonMap() Mapper {
	return NewJSONMapper().Field("s", b.s)
}

func TestJsonMapper(t *testing.T) {
	test_utils.NewTestGroup("", "").Cases([]*test_utils.Assertion{
		test_utils.NewTestCase("", "", func() bool {
			listOfBoxes := make([]Box, 5, 5)
			for i := 0; i < 5; i++ {
				listOfBoxes[i] = Box{"?"}
			}
			listOfMappers := make([]Mapper, 5, 5)
			for i := 0; i < 5; i++ {
				listOfMappers[i] = listOfBoxes[i].toJsonMap()
			}
			m := NewJSONMapper()
			m.Field("hello", "world").Field("a", true).Field("b", uint32(321423)).Field("c", 123213.342304).FloatPrecision(1).Field("omitted", nil).OmitNilValue(true)
			mm := NewJSONMapper()
			str := mm.Field("o", m).Field("list", []string{"a", "b"}).Field("mapList", listOfMappers).Field("emptyList", []float32{}).Field("ending", true).Field("someNil", nil).Field("omitted", "").OmitEmptyStringValue(true).ToString()
			t.Logf(str)
			return false
		}),
	}).Do(t)
}
