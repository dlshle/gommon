package logger

import "github.com/dlshle/gommon/gr_context"

// a "thread" safe go-routine level logging context maintainer
// it lives and dies with the associated go-routine

const (
	prefix     = "$logging_"
	prefix_len = 9
)

func GrSet(k, v string) {
	gr_context.Put(prefix+k, v)
}

// deprecated
func Set(k, v string) {
	GrSet(k, v)
}

func GrGet(k string) string {
	rawValue := gr_context.Get(prefix + k)
	if rawValue == nil {
		return ""
	}
	return rawValue.(string)
}

// deprecated
func Get(k string) string {
	return GrGet(k)
}

func GrGetAll() (res map[string]string) {
	res = make(map[string]string)
	subset := gr_context.GetByPrefix(prefix)
	for k, v := range subset {
		res[k[9:]] = v.(string)
	}
	return
}

// deprecated
func GetAll() map[string]string {
	return GrGetAll()
}

// deprecated
func Delete(k string) {
	GrDelete(k)
}

func GrDelete(k string) {
	gr_context.Delete(prefix + k)
}

// deprecated
func Clear() {
	GrClear()
}

func GrClear() {
	gr_context.ClearByPrefix(prefix)
}
