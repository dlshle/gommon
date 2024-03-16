package utils

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

var Rando *rand.Rand

func init() {
	Rando = NewRand()
}

func ByteToUpperCase(b byte) byte {
	if b > 96 && b < 123 {
		return b - 32
	}
	return b
}

func ByteToLowerCase(b byte) byte {
	if b > 64 && b < 91 {
		return b + 32
	}
	return b
}

func ToCamelCase(name string) string {
	return fmt.Sprintf("%c%s", ByteToLowerCase(name[0]), name[1:])
}

func ToPascalCase(name string) string {
	return fmt.Sprintf("%c%s", ByteToUpperCase(name[0]), name[1:])
}

func GetOr(obj interface{}, otherwise func() interface{}) interface{} {
	if obj != nil {
		return obj
	}
	return otherwise()
}

func ConditionalPick(cond bool, onTrue interface{}, onFalse interface{}) interface{} {
	if cond {
		return onTrue
	} else {
		return onFalse
	}
}

func ConditionalGet(cond bool, onTrue func() interface{}, onFalse func() interface{}) interface{} {
	if cond {
		return onTrue()
	} else {
		return onFalse()
	}
}

func SliceToSet(slice []interface{}) map[interface{}]bool {
	m := make(map[interface{}]bool)
	for _, v := range slice {
		m[v] = true
	}
	return m
}

func TypedSliceToSet[T comparable](slice []T) map[T]bool {
	m := make(map[T]bool)
	for _, v := range slice {
		m[v] = true
	}
	return m
}

func CopySet(set map[interface{}]bool) map[interface{}]bool {
	c := make(map[interface{}]bool)
	for k, v := range set {
		c[k] = v
	}
	return c
}

func SetIntersections(l map[interface{}]bool, r map[interface{}]bool) map[interface{}]bool {
	lCopy := CopySet(l)
	rCopy := CopySet(r)
	for k := range lCopy {
		if rCopy[k] {
			lCopy[k] = false
			rCopy[k] = false
		} else {
			rCopy[k] = true
		}
	}
	return rCopy
}

func StringArrayToInterfaceArray(arr []string) []interface{} {
	res := make([]interface{}, len(arr))
	for i := range arr {
		res[i] = arr[i]
	}
	return res
}

func NewRand() *rand.Rand {
	return rand.New(rand.NewSource(time.Now().Unix()))
}

func ProcessWithError(processors []func() error) (err error) {
	for _, processor := range processors {
		if err = processor(); err != nil {
			return
		}
	}
	return
}

func ProcessWithErrors(funcs ...func() error) error {
	return ProcessWithError(funcs)
}

func GetIpAddrAndPort(remoteAddr string) (res [2]string, err error) {
	for i := len(remoteAddr) - 1; i > 0; i-- {
		if remoteAddr[i] == ':' && i < len(remoteAddr)-1 {
			res[0] = remoteAddr[:i]
			res[1] = remoteAddr[i+1:]
			return
		}
	}
	err = fmt.Errorf("invalid remote addr format(%s)", remoteAddr)
	return
}

func IsStringsNotEmpty(targets ...string) bool {
	for _, str := range targets {
		if str == "" {
			return false
		}
	}
	return true
}

func EncodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func DecodeBase64(encoded string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(encoded)
}

func StringMapToJSON(s map[string]string) string {
	var buffer bytes.Buffer
	buffer.WriteRune('{')
	l := len(s)
	counter := 0
	for k, v := range s {
		buffer.WriteRune('"')
		buffer.WriteString(k)
		buffer.WriteString("\":\"" + v + "\"")
		if counter < l-1 {
			buffer.WriteRune(',')
		}
		counter++
	}
	buffer.WriteRune('}')
	return buffer.String()
}

func CheckNonZero[T comparable](val T) T {
	var zeroVal T
	if val == zeroVal {
		panic(fmt.Sprintf("value %v is a zero val", val))
	}
	return val
}

func UnmarshalJSONEntity[T any](data []byte) (holder T, err error) {
	err = json.Unmarshal([]byte(data), &holder)
	return
}

func EncodeString(str string) string {
	return strings.ReplaceAll(str, "\"", "\\\"")
}

func RandomStringWithSize(size int) string {
	str := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	bytes := []byte(str)
	result := []byte{}
	r := NewRand()
	for i := 0; i < size; i++ {
		result = append(result, bytes[r.Intn(len(bytes))])
	}
	return string(result)
}
