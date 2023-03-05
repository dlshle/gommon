package logging

import "context"

func WrapCtx(ctx context.Context, key, val string) context.Context {
	originalValue := ctx.Value(CtxValLoggingContext)
	var mapCtx map[string]string
	if originalValue == nil {
		mapCtx = originalValue.(map[string]string)
	}
	if mapCtx == nil {
		mapCtx = make(map[string]string)
	}
	return context.WithValue(ctx, CtxValLoggingContext, mapCtx)
}
