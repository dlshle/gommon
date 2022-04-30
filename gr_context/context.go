package gr_context

import (
	"github.com/petermattis/goid"
	"strconv"
	"strings"
)

// no need to use lock as operations are on the same goroutine
var context = map[string]interface{}{}

func Put(key string, v interface{}) {
	context[getGoID()+key] = v
}

func Get(key string) interface{} {
	return context[getGoID()+key]
}

func Delete(key string) {
	delete(context, getGoID()+key)
}

func Clear() {
	id := getGoID()
	for k := range context {
		if strings.HasPrefix(k, id) {
			delete(context, k)
		}
	}
}

func getGoID() string {
	return strconv.FormatInt(goid.Get(), 10)
}
