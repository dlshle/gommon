package utils

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/dlshle/gommon/errors"
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
		buffer.WriteString(`":`)
		vs, _ := json.Marshal(v)
		buffer.WriteString(string(vs))
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

const hex = "0123456789abcdef"

func EncodeJSONString(src string) string {
	var b []byte
	buf := bytes.NewBuffer(b)
	start := 0
	for i := 0; i < len(src); {
		if b := src[i]; b < utf8.RuneSelf {
			buf.WriteString(src[start:i])
			switch b {
			case '\\', '"':
				buf.WriteRune('\\')
				buf.WriteByte(b)
			case '\b':
				buf.WriteRune('\\')
				buf.WriteRune('b')
			case '\f':
				buf.WriteRune('\\')
				buf.WriteRune('f')
			case '\n':
				buf.WriteRune('\\')
				buf.WriteRune('n')
			case '\r':
				buf.WriteRune('\\')
				buf.WriteRune('r')
			case '\t':
				buf.WriteRune('\\')
				buf.WriteRune('t')
			default:
				// This encodes bytes < 0x20 except for \b, \f, \n, \r and \t.
				// If escapeHTML is set, it also escapes <, >, and &
				// because they can lead to security holes when
				// user-controlled strings are rendered into JSON
				// and served to some browsers.
				buf.WriteRune('\\')
				buf.WriteRune('u')
				buf.WriteRune('0')
				buf.WriteRune('0')
				buf.WriteByte(hex[b>>4])
				buf.WriteByte(hex[b&0xF])
			}
			i++
			start = i
			continue
		}
		// TODO(https://go.dev/issue/56948): Use generic utf8 functionality.
		// For now, cast only a small portion of byte slices to a string
		// so that it can be stack allocated. This slows down []byte slightly
		// due to the extra copy, but keeps string performance roughly the same.
		n := len(src) - i
		if n > utf8.UTFMax {
			n = utf8.UTFMax
		}
		c, size := utf8.DecodeRuneInString(string(src[i : i+n]))
		if c == utf8.RuneError && size == 1 {
			buf.WriteString(src[start:i])
			buf.WriteString(`\ufffd`)
			i += size
			start = i
			continue
		}
		// U+2028 is LINE SEPARATOR.
		// U+2029 is PARAGRAPH SEPARATOR.
		// They are both technically valid characters in JSON strings,
		// but don't work in JSONP, which has to be evaluated as JavaScript,
		// and can lead to security holes there. It is valid JSON to
		// escape them, so we do so unconditionally.
		// See https://en.wikipedia.org/wiki/JSON#Safety.
		if c == '\u2028' || c == '\u2029' {
			buf.WriteString(src[start:i])
			buf.WriteRune('\\')
			buf.WriteRune('u')
			buf.WriteRune('2')
			buf.WriteRune('0')
			buf.WriteRune('2')
			buf.WriteByte(hex[c&0xF])
			i += size
			start = i
			continue
		}
		i += size
	}
	buf.WriteString(src[start:])
	return buf.String()
}

func DiscoverFiles(filePattern string) ([]string, error) {
	if filePattern == "" {
		return nil, errors.Error("file pattern is empty")
	}
	if !strings.Contains(filePattern, "/") {
		// prepend current directory
		filePattern = "." + string(filepath.Separator) + filePattern
	}
	return filepath.Glob(filePattern)
}

// Deduplicate removes duplicates from a slice and tells if the slice contains any duplicate
func Deduplicate[T comparable](s []T) ([]T, bool) {
	var (
		result       []T  = make([]T, 0)
		hasDuplicate bool = false
	)
	m := make(map[T]bool)
	for _, v := range s {
		if !m[v] {
			m[v] = true
		} else {
			hasDuplicate = true
		}
	}
	for k := range m {
		result = append(result, k)
	}
	return result, hasDuplicate
}
