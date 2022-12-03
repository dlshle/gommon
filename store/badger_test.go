package store

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/dlshle/gommon/test_utils"
)

type TestEntity struct {
	K string `json:"k"`
	T string `json:"t"`
	V string `json:"v"`
}

type TestEntitySerializeHandler struct{}

func (h TestEntitySerializeHandler) KeySerializer(k string) ([]byte, error) {
	return []byte(k), nil
}

func (h TestEntitySerializeHandler) KeyDeserializer(key []byte) (string, error) {
	return string(key), nil
}

func (h TestEntitySerializeHandler) ValueSerializer(v TestEntity) ([]byte, error) {
	return []byte(fmt.Sprintf(`{"k":"%s","t":"%s","v":"%s"}`, v.K, v.T, v.V)), nil
}

func (h TestEntitySerializeHandler) ValueDeserializer(data []byte) (te TestEntity, err error) {
	err = json.Unmarshal(data, &te)
	return
}

type logDataKVSerializationHandler struct{}

func newLogDataKVSerializationHandler() BadgerStoreSerializeHandler[uint64, *map[string]string] {
	return logDataKVSerializationHandler{}
}

func (h logDataKVSerializationHandler) KeySerializer(k uint64) ([]byte, error) {
	return uint64ToBytes(k), nil
}

func (h logDataKVSerializationHandler) KeyDeserializer(k []byte) (uint64, error) {
	return bytesToUint64(k), nil
}

func (h logDataKVSerializationHandler) ValueSerializer(v *map[string]string) ([]byte, error) {
	return json.Marshal(*v)
}

func (h logDataKVSerializationHandler) ValueDeserializer(v []byte) (*map[string]string, error) {
	var holder map[string]string
	err := json.Unmarshal(v, &holder)
	return &holder, err
}

func TestBadgerStore(t *testing.T) {
	var (
		db       KVStore[string, TestEntity]
		data     TestEntity
		existing TestEntity
		err      error
	)
	test_utils.NewGroup("badger", "test badger store basic functionalities").Cases(test_utils.New("creation", func() {
		db, err = NewBadgerStore[string, TestEntity]("db", TestEntitySerializeHandler{})
		test_utils.AssertNil(err)
	}), test_utils.New("test crud", func() {
		defer func() {
			test_utils.AssertNil(db.Close())
		}()
		existing = TestEntity{K: "test", T: "something", V: "hello"}
		_, err := db.Put("test", existing)
		test_utils.AssertNil(err)
		data, err = db.Get("test")
		test_utils.AssertNil(err)
		test_utils.AssertEquals(data, existing)
		data.V = "newV"
		_, err = db.Update("test", data)
		test_utils.AssertNil(err)
		data, err = db.Get("test")
		test_utils.AssertNil(err)
		test_utils.AssertEquals(data.V, "newV")
		_, err = db.Delete("test")
		test_utils.AssertNil(err)
		data, err = db.Get("test")
		test_utils.AssertNonNil(err)
		test_utils.AssertEquals(data, TestEntity{})
		existing = TestEntity{V: "a"}
		_, err = db.Put("test1", existing)
		test_utils.AssertNil(err)
		test_utils.AssertNil(db.Close())

		db, err = NewBadgerStore[string, TestEntity]("db", TestEntitySerializeHandler{})
		test_utils.AssertNil(err)
		data, err = db.Get("test1")
		test_utils.AssertNil(err)
		test_utils.AssertEquals(data, existing)
		_, err = db.Delete("test1")
		test_utils.AssertNil(err)
	}), test_utils.New("load again and test data", func() {
		db, err = NewBadgerStore[string, TestEntity]("db", TestEntitySerializeHandler{})
		test_utils.AssertNil(err)
		var sequentialData = map[string]TestEntity{"a": {V: "a"}, "b": {V: "b"}}
		_, err = db.BulkPut(sequentialData)
		test_utils.AssertNil(err)
		var allData []TestEntity
		allData, err = db.Query(func(k string, record TestEntity) bool {
			return true
		})
		test_utils.AssertNil(err)
		test_utils.AssertEquals(len(allData), 2)
		test_utils.AssertEquals(allData[0], TestEntity{V: "a"})
		test_utils.AssertEquals(allData[1], TestEntity{V: "b"})
		_, err = db.Delete("a")
		test_utils.AssertNil(err)
		_, err = db.Delete("b")
		test_utils.AssertNil(err)
		test_utils.AssertNil(db.Drop())
		test_utils.AssertNil(db.Close())
	}), test_utils.New("test auto incr db", func() {
		dba, err := NewAutoIncrBadgerStore("dba", newLogDataKVSerializationHandler())
		test_utils.AssertNil(err)
		everyOne, err := dba.Query(func(key uint64, record *map[string]string) bool { return true })
		test_utils.AssertNil(err)
		test_utils.AssertEquals(len(everyOne), 0)
		_, _, err = dba.Create(&map[string]string{"a": "b"})
		test_utils.AssertNil(err)
		_, _, err = dba.Create(&map[string]string{"b": "c"})
		test_utils.AssertNil(err)
		everyOne, err = dba.Query(func(key uint64, record *map[string]string) bool { return true })
		test_utils.AssertNil(err)
		test_utils.AssertEquals(len(everyOne), 2)
		test_utils.AssertNil(dba.Drop())
		test_utils.AssertNil(dba.Close())
	})).Do(t)
}
