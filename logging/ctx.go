package logging

import (
	"context"
)

func WrapCtx(ctx context.Context, key, val string) context.Context {
	originalValue := ctx.Value(CtxValLoggingContext)
	var mapCtx map[string]string
	if originalValue != nil {
		original := originalValue.(map[string]string)
		mapCtx = make(map[string]string)
		for k, v := range original {
			mapCtx[k] = v
		}
	}
	if mapCtx == nil {
		mapCtx = make(map[string]string)
	}
	mapCtx[key] = val
	return context.WithValue(ctx, CtxValLoggingContext, mapCtx)
}
