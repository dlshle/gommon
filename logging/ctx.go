package logging

import (
	"context"
)

// WrapCtx returns a new context enriched with a logging key-value pair.
// If the context already carries logging values, they are copied and merged
// with the new pair. Logging values stored this way are merged into the
// LogEntity.Context by DefaultLogger when a log call is made.
func WrapCtx(ctx context.Context, key, val string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	originalValue := ctx.Value(CtxValLoggingContext)
	mapCtx := make(map[string]string)
	if originalValue != nil {
		if original, ok := originalValue.(map[string]string); ok {
			for k, v := range original {
				mapCtx[k] = v
			}
		}
	}
	mapCtx[key] = val
	return context.WithValue(ctx, CtxValLoggingContext, mapCtx)
}
