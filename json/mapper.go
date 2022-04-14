package json

import (
	"reflect"
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
		default:
			if reflect.TypeOf(v).Kind() == reflect.Slice {
				m[k] = j.toListString(v)
			}
		}
	}
	return m
}

func (j *jsonMapper) toListString(v interface{}) string {
	var builder strings.Builder
	builder.WriteByte('[')
	switch v.(type) {
	case []string:
		counter := 0
		segment := v.([]string)
		l := len(segment)
		for _, str := range v.([]string) {
			builder.WriteByte('"')
			builder.WriteString(str)
			builder.WriteByte('"')
			counter++
			if counter < l {
				builder.WriteByte(',')
			}
		}
		break
	case []int:
		counter := 0
		segment := v.([]int)
		l := len(segment)
		for _, e := range segment {
			builder.WriteString(strconv.Itoa(e))
			counter++
			if counter < l {
				builder.WriteByte(',')
			}
		}
		break
	case []int32:
		counter := 0
		segment := v.([]int32)
		l := len(segment)
		for _, e := range segment {
			builder.WriteString(strconv.FormatInt(int64(e), 10))
			counter++
			if counter < l {
				builder.WriteByte(',')
			}
		}
		break
	case []int64:
		counter := 0
		segment := v.([]int64)
		l := len(segment)
		for _, e := range segment {
			builder.WriteString(strconv.FormatInt(e, 10))
			counter++
			if counter < l {
				builder.WriteByte(',')
			}
		}
		break
	case []uint:
		counter := 0
		segment := v.([]uint)
		l := len(segment)
		for _, e := range segment {
			builder.WriteString(strconv.FormatUint(uint64(e), 10))
			counter++
			if counter < l {
				builder.WriteByte(',')
			}
		}
		break
	case []uint32:
		counter := 0
		segment := v.([]uint32)
		l := len(segment)
		for _, e := range segment {
			builder.WriteString(strconv.FormatUint(uint64(e), 10))
			counter++
			if counter < l {
				builder.WriteByte(',')
			}
		}
		break
	case []uint64:
		counter := 0
		segment := v.([]uint64)
		l := len(segment)
		for _, e := range segment {
			builder.WriteString(strconv.FormatUint(e, 10))
			counter++
			if counter < l {
				builder.WriteByte(',')
			}
		}
		break
	case []float32:
		counter := 0
		segment := v.([]float32)
		l := len(segment)
		for _, e := range segment {
			builder.WriteString(strconv.FormatFloat(float64(e), 'f', j.floatPrecision, 32))
			counter++
			if counter < l {
				builder.WriteByte(',')
			}
		}
		break
	case []float64:
		counter := 0
		segment := v.([]float64)
		l := len(segment)
		for _, e := range segment {
			builder.WriteString(strconv.FormatFloat(e, 'f', j.floatPrecision, 32))
			counter++
			if counter < l {
				builder.WriteByte(',')
			}
		}
		break
	case []bool:
		counter := 0
		segment := v.([]bool)
		l := len(segment)
		for _, b := range segment {
			if b {
				builder.WriteString("true")
			} else {
				builder.WriteString("false")
			}
			counter++
			if counter < l {
				builder.WriteByte(',')
			}
		}
		break
	case []Mapper:
		counter := 0
		segment := v.([]Mapper)
		l := len(segment)
		for _, str := range v.([]Mapper) {
			builder.WriteString(str.ToString())
			counter++
			if counter < l {
				builder.WriteByte(',')
			}
		}
		break
	}
	builder.WriteByte(']')
	return builder.String()
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
