package logger

var ctx = make(map[string]string)

func SetGlobalContext(k, v string) {
	ctx[k] = v
}

func DeleteGlobalContext(k string) {
	delete(ctx, k)
}

func ClearGlobalContext() {
	for k := range ctx {
		delete(ctx, k)
	}
}

func getGlobalContexts() map[string]string {
	return ctx
}
