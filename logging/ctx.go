package logging

import "context"

func WrapCtx(ctx context.Context, key, val string) context.Context {
	var (
		mapCtx map[string]string = nil
		ok     bool
	)
	originalValue := ctx.Value(CtxValLoggingContext)
	if mapCtx, ok = originalValue.(map[string]string); mapCtx == nil || !ok {
		mapCtx = make(map[string]string)
	}
	mapCtx[key] = val
	return context.WithValue(ctx, CtxValLoggingContext, mapCtx)
}
