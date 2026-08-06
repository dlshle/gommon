package logging

import (
	"context"
	"testing"
)

func TestWrapCtxEnrichesContext(t *testing.T) {
	ctx := WrapCtx(context.Background(), "request_id", "abc123")
	ctx = WrapCtx(ctx, "user_id", "42")

	val := ctx.Value(CtxValLoggingContext)
	if val == nil {
		t.Fatal("expected logging context value")
	}
	loggingCtx, ok := val.(map[string]string)
	if !ok {
		t.Fatalf("expected map[string]string, got %T", val)
	}
	if loggingCtx["request_id"] != "abc123" {
		t.Errorf("expected request_id=abc123, got %q", loggingCtx["request_id"])
	}
	if loggingCtx["user_id"] != "42" {
		t.Errorf("expected user_id=42, got %q", loggingCtx["user_id"])
	}
}

func TestWrapCtxWithNilContext(t *testing.T) {
	ctx := WrapCtx(nil, "key", "val")
	if ctx == nil {
		t.Fatal("expected non-nil context")
	}
	loggingCtx := ctx.Value(CtxValLoggingContext).(map[string]string)
	if loggingCtx["key"] != "val" {
		t.Errorf("expected key=val, got %q", loggingCtx["key"])
	}
}

func TestDefaultLoggerMergesContextValues(t *testing.T) {
	cap := &captureWriter{}
	logger := CreateDefaultLogger(cap, "", LogAllWaterMark)
	logger.SetContext("logger_key", "logger_val")

	ctx := WrapCtx(context.Background(), "request_id", "abc123")
	logger.Info(ctx, "hello")

	entities := cap.Entities()
	if len(entities) != 1 {
		t.Fatalf("expected 1 entity, got %d", len(entities))
	}
	if entities[0].Context["logger_key"] != "logger_val" {
		t.Errorf("expected logger context, got %v", entities[0].Context)
	}
	if entities[0].Context["request_id"] != "abc123" {
		t.Errorf("expected context value request_id=abc123, got %v", entities[0].Context)
	}
}
