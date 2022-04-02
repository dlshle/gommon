package stringz

import (
	"reflect"
	"strconv"
	"strings"
	"sync"
)

var pool sync.Pool

func init() {
	pool = sync.Pool{
		New: func() interface{} {
			var builder strings.Builder
			return &stringBuilder{
				builder: builder,
			}
		},
	}
}

type StringBuilder interface {
	Pointer(interface{}) StringBuilder
	Bytes([]byte) StringBuilder
	String(string) StringBuilder
	Uint(uint) StringBuilder
	Uint32(uint32) StringBuilder
	Uint64(uint64) StringBuilder
	Int(int) StringBuilder
	Int32(int32) StringBuilder
	Int64(int64) StringBuilder
	Float32(float32) StringBuilder
	Float64(float64) StringBuilder
	Byte(byte) StringBuilder
	Bool(bool) StringBuilder
	Stringify(Stringify) StringBuilder
	Build() string
	BuildL() string
}

type stringBuilder struct {
	builder strings.Builder
}

func Builder() StringBuilder {
	return pool.Get().(StringBuilder)
}

func (b *stringBuilder) Bytes(stream []byte) StringBuilder {
	b.builder.Write(stream)
	return b
}

func (b *stringBuilder) String(s string) StringBuilder {
	b.builder.WriteString(s)
	return b
}

func (b *stringBuilder) Uint(u uint) StringBuilder {
	b.builder.WriteString(strconv.FormatUint(uint64(u), 10))
	return b
}

func (b *stringBuilder) Uint32(u uint32) StringBuilder {
	b.builder.WriteString(strconv.FormatUint(uint64(u), 10))
	return b
}

func (b *stringBuilder) Uint64(u uint64) StringBuilder {
	b.builder.WriteString(strconv.FormatUint(u, 10))
	return b
}

func (b *stringBuilder) Int(u int) StringBuilder {
	b.builder.WriteString(strconv.Itoa(u))
	return b
}

func (b *stringBuilder) Int32(u int32) StringBuilder {
	b.builder.WriteString(strconv.Itoa(int(u)))
	return b
}

func (b *stringBuilder) Int64(u int64) StringBuilder {
	b.builder.WriteString(strconv.FormatInt(u, 10))
	return b
}

func (b *stringBuilder) Float32(u float32) StringBuilder {
	b.builder.WriteString(strconv.FormatFloat(float64(u), 'f', 6, 32))
	return b
}

func (b *stringBuilder) Float64(u float64) StringBuilder {
	b.builder.WriteString(strconv.FormatFloat(u, 'f', 6, 64))
	return b
}

func (b *stringBuilder) Byte(u byte) StringBuilder {
	b.builder.WriteByte(u)
	return b
}

func (b *stringBuilder) Bool(v bool) StringBuilder {
	if v {
		b.builder.WriteString("true")
	} else {
		b.builder.WriteString("false")
	}
	return b
}

func (b *stringBuilder) Pointer(v interface{}) StringBuilder {
	if v == nil {
		b.String("nil")
		return b
	}
	reflectVal := reflect.ValueOf(v)
	b.Uint64(uint64(reflectVal.Pointer()))
	return b
}

func (b *stringBuilder) Stringify(s Stringify) StringBuilder {
	b.builder.WriteString(s.String())
	return b
}

func (b *stringBuilder) Build() string {
	defer func() {
		b.builder.Reset()
		pool.Put(b)
	}()
	return b.builder.String()
}

func (b *stringBuilder) BuildL() string {
	b.Byte('\n')
	return b.Build()
}

func ConcatString(strings ...string) string {
	builder := Builder()
	for _, str := range strings {
		builder.String(str)
	}
	return builder.Build()
}

func ConcatStringify(stringifys ...Stringify) string {
	builder := Builder()
	for _, strify := range stringifys {
		builder.Stringify(strify)
	}
	return builder.Build()
}
