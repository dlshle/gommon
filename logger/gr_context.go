package logger

import "github.com/dlshle/gommon/gr_context"

// a go routine safe global logging context maintainer

const (
	prefix     = "$logging_"
	prefix_len = 9
)

func Set(k, v string) {
	gr_context.Put(prefix+k, v)
}

func Get(k string) string {
	rawValue := gr_context.Get(prefix + k)
	if rawValue == nil {
		return ""
	}
	return rawValue.(string)
}

func getAll() (res map[string]string) {
	res = make(map[string]string)
	subset := gr_context.GetByPrefix(prefix)
	for k, v := range subset {
		res[k[9:]] = v.(string)
	}
	return
}

func Delete(k string) {
	gr_context.Delete(prefix + k)
}

func Clear() {
	gr_context.ClearByPrefix(prefix)
}
