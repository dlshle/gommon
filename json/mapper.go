package json

import (
	"strconv"
	"strings"
)

type Mapper interface {
	OmitEmptyStringValue(bool) Mapper
	OmitNilValue(bool) Mapper
	FloatPrecision(n int) Mapper
	Field(k string, v interface{}) Mapper
	Omit(key string) Mapper
	ToMap() map[string]interface{}
	StringMap() map[string]string
	ToString() string
}

type jsonMapper struct {
	floatPrecision  int
	omitEmptyString bool
	omitNil         bool
	m               map[string]interface{}
}

func (j *jsonMapper) OmitEmptyStringValue(b bool) Mapper {
	j.omitEmptyString = b
	return j
}

func (j *jsonMapper) OmitNilValue(b bool) Mapper {
	j.omitNil = b
	return j
}

func (j *jsonMapper) Field(k string, v interface{}) Mapper {
	j.m[k] = v
	return j
}

func (j *jsonMapper) Omit(key string) Mapper {
	delete(j.m, key)
	return j
}

func (j *jsonMapper) FloatPrecision(n int) Mapper {
	j.floatPrecision = n
	return j
}

func (j *jsonMapper) ToMap() map[string]interface{} {
	return j.m
}

func (j *jsonMapper) StringMap() map[string]string {
	m := make(map[string]string)
	for k, v := range j.m {
		switch v.(type) {
		case string:
			if !j.omitEmptyString {
				m[k] = "\"" + v.(string) + "\""
			}
			break
		case int:
			m[k] = strconv.Itoa(v.(int))
			break
		case int32:
			m[k] = strconv.FormatInt(int64(v.(int32)), 10)
			break
		case int64:
			m[k] = strconv.FormatInt(v.(int64), 10)
			break
		case uint:
			m[k] = strconv.FormatUint(uint64(v.(uint)), 10)
			break
		case uint32:
			m[k] = strconv.FormatUint(uint64(v.(uint32)), 10)
			break
		case uint64:
			m[k] = strconv.FormatUint(v.(uint64), 10)
			break
		case float32:
			m[k] = strconv.FormatFloat(float64(v.(float32)), 'f', j.floatPrecision, 32)
			break
		case float64:
			m[k] = strconv.FormatFloat(v.(float64), 'f', j.floatPrecision, 64)
			break
		case bool:
			bv := v.(bool)
			if bv {
				m[k] = "true"
			} else {
				m[k] = "false"
			}
			break
		case Mapper:
			m[k] = v.(Mapper).ToString()
			break
		case nil:
			if !j.omitNil {
				m[k] = "null"
			}
			break
		}
	}
	return m
}

func (j *jsonMapper) ToString() string {
	counter := 0
	m := j.StringMap()
	l := len(m)
	var builder strings.Builder
	builder.WriteByte('{')
	for k, v := range m {
		builder.WriteByte('"')
		builder.WriteString(k)
		builder.WriteByte('"')
		builder.WriteByte(':')
		builder.WriteString(v)
		counter++
		if counter < l {
			builder.WriteByte(',')
		}
	}
	builder.WriteByte('}')
	return builder.String()
}

func NewJSONMapper() Mapper {
	return &jsonMapper{m: make(map[string]interface{}), floatPrecision: 5}
}
