package reflect

import (
	"fmt"
	"testing"
	"github.com/dlshle/gommon/test_utils"
)

type Box struct {
	name string `aname`
	age  int    `theage`
	p    IPrintable
}

type Boxed struct{}

type IPrintable interface {
	Print()
}

func (b *Boxed) Print() {
	fmt.Println("")
}

func TestGetFieldsAndTags(t *testing.T) {
	b := &Box{}
	test_utils.NewTestGroup("reflect_utils", "").
		Then("should not return error when getting from ptr kind", "", func() bool {
			m, err := GetFieldsAndTags(b)
			if err != nil {
				return false
			}
			return m["name"] == "aname"
		}).
		Then("should get correct tags from struct kind", "", func() bool {
			m, e := GetFieldsAndTags(*b)
			if e != nil {
				return false
			}
			return m["name"] == "aname"
		}).
		Do(t)
}

func TestGetValueByField(t *testing.T) {
	b := &Box{"hello", 1, &Boxed{}}
	test_utils.NewTestGroup("reflect_utils", "").Then("test getValue", "", func() bool {
		val, err := GetValueByField(b, "name")
		t.Log(val, err)
		return val.String() == "hello"
	}).Then("test getValue with non-ptr", "", func() bool {
		bb := Box{"hello", 1, &Boxed{}}
		val, err := GetValueByField(bb, "name")
		t.Log(val, err)
		return val.String() == "hello"
	}).Do(t)
	t.Log(GetFieldTypeByName(b, "p"))
	t.Log(GetObjectType(b))
}
