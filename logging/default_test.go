package logging

import (
	"bytes"
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dlshle/gommon/errors"
)

type captureWriter struct {
	mu       sync.Mutex
	entities []*LogEntity
}

func (c *captureWriter) Write(entity *LogEntity) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entities = append(c.entities, entity)
	return nil
}

func (c *captureWriter) Entities() []*LogEntity {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]*LogEntity, len(c.entities))
	copy(out, c.entities)
	return out
}

func TestDefaultLoggerLevelFiltering(t *testing.T) {
	cap := &captureWriter{}
	logger := CreateDefaultLogger(cap, "test", INFO)

	logger.Debug(context.Background(), "debug message")
	logger.Info(context.Background(), "info message")
	logger.Warn(context.Background(), "warn message")

	entities := cap.Entities()
	if len(entities) != 2 {
		t.Fatalf("expected 2 entities, got %d", len(entities))
	}
	if entities[0].Level != INFO {
		t.Errorf("expected first entity level INFO, got %v", entities[0].Level)
	}
	if entities[1].Level != WARN {
		t.Errorf("expected second entity level WARN, got %v", entities[1].Level)
	}
}

func TestDefaultLoggerPrefixAndContext(t *testing.T) {
	cap := &captureWriter{}
	logger := CreateDefaultLogger(cap, "root", LogAllWaterMark)
	logger.SetContext("request_id", "abc123")

	child := logger.WithPrefix("child").WithContext(map[string]string{"child_key": "child_val"})
	child.Info(context.Background(), "hello")

	entities := cap.Entities()
	if len(entities) != 1 {
		t.Fatalf("expected 1 entity, got %d", len(entities))
	}
	if !strings.Contains(entities[0].Prefix, "child") {
		t.Errorf("expected prefix to contain 'child', got %q", entities[0].Prefix)
	}
	if entities[0].Context["request_id"] != "abc123" {
		t.Errorf("expected inherited context request_id=abc123, got %q", entities[0].Context["request_id"])
	}
	if entities[0].Context["child_key"] != "child_val" {
		t.Errorf("expected child context child_key=child_val, got %q", entities[0].Context["child_key"])
	}
}

func TestDefaultLoggerContextIsolation(t *testing.T) {
	cap := &captureWriter{}
	logger := CreateDefaultLogger(cap, "", LogAllWaterMark)
	logger.SetContext("k", "parent")

	child := logger.WithContext(map[string]string{"k": "child"})
	child.Info(context.Background(), "child")
	logger.Info(context.Background(), "parent")

	entities := cap.Entities()
	if len(entities) != 2 {
		t.Fatalf("expected 2 entities, got %d", len(entities))
	}
	if entities[0].Context["k"] != "child" {
		t.Errorf("expected child context k=child, got %q", entities[0].Context["k"])
	}
	if entities[1].Context["k"] != "parent" {
		t.Errorf("expected parent context k=parent, got %q", entities[1].Context["k"])
	}
}

func TestDefaultLoggerConcurrentLogging(t *testing.T) {
	cap := &captureWriter{}
	logger := CreateDefaultLogger(cap, "", LogAllWaterMark)
	logger.SetContext("k", "v")

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			logger.Infof(context.Background(), "message %d", n)
		}(i)
	}
	wg.Wait()

	if len(cap.Entities()) != 100 {
		t.Fatalf("expected 100 entities, got %d", len(cap.Entities()))
	}
}

func TestDefaultLoggerCallerFile(t *testing.T) {
	cap := &captureWriter{}
	logger := CreateDefaultLogger(cap, "", LogAllWaterMark)
	logger.Info(context.Background(), "caller test")

	entities := cap.Entities()
	if len(entities) != 1 {
		t.Fatalf("expected 1 entity, got %d", len(entities))
	}
	if !strings.Contains(entities[0].File, "default_test.go:") {
		t.Errorf("expected caller file default_test.go, got %q", entities[0].File)
	}
}

func TestDefaultLoggerTrackableError(t *testing.T) {
	cap := &captureWriter{}
	logger := CreateDefaultLogger(cap, "", LogAllWaterMark)

	err := errors.Error("boom")
	logger.TrackableError(context.Background(), err, "failure")

	entities := cap.Entities()
	if len(entities) != 1 {
		t.Fatalf("expected 1 entity, got %d", len(entities))
	}
	if entities[0].Level != ERROR {
		t.Errorf("expected ERROR level, got %v", entities[0].Level)
	}
	if !strings.Contains(entities[0].Message, "failure") {
		t.Errorf("expected message to contain user text, got %q", entities[0].Message)
	}
	if _, ok := entities[0].Context["stacktrace"]; !ok {
		t.Errorf("expected stacktrace in context")
	}
}

func TestDefaultLoggerMessageTruncate(t *testing.T) {
	cap := &captureWriter{}
	logger := CreateDefaultLogger(cap, "", LogAllWaterMark)
	logger.SetMessageTruncateThreshold(5)

	logger.Info(context.Background(), "hello world")

	entities := cap.Entities()
	if len(entities) != 1 {
		t.Fatalf("expected 1 entity, got %d", len(entities))
	}
	if entities[0].Message != "hello..." {
		t.Errorf("expected truncated message, got %q", entities[0].Message)
	}
}

func TestSimpleStringWriter(t *testing.T) {
	var buf bytes.Buffer
	w := NewConsoleLogWriter(&buf)
	entity := newLogEntity(INFO, "svc", map[string]string{"k": "v"}, time.Now(), "hello", "file.go:10")
	if err := w.Write(entity); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "[INFO]") {
		t.Errorf("expected [INFO] in output, got %q", out)
	}
	if !strings.Contains(out, "hello") {
		t.Errorf("expected message in output, got %q", out)
	}
}
