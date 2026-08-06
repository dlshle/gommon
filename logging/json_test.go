package logging

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestJSONWriterOutput(t *testing.T) {
	var buf bytes.Buffer
	w := NewJSONWriter(&buf)
	entity := newLogEntity(INFO, "svc", map[string]string{"k": "v"}, time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC), "hello", "file.go:10")
	if err := w.Write(entity); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	out := buf.String()
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, out)
	}

	if parsed["level"] != "INFO" {
		t.Errorf("expected level INFO, got %v", parsed["level"])
	}
	if parsed["message"] != "hello" {
		t.Errorf("expected message hello, got %v", parsed["message"])
	}
	if parsed["prefix"] != "svc" {
		t.Errorf("expected prefix svc, got %v", parsed["prefix"])
	}
	if parsed["file"] != "file.go:10" {
		t.Errorf("expected file file.go:10, got %v", parsed["file"])
	}
	if parsed["timestamp"] != "2024-01-02T03:04:05Z" {
		t.Errorf("expected timestamp, got %v", parsed["timestamp"])
	}
	ctx, ok := parsed["context"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected context object, got %T", parsed["context"])
	}
	if ctx["k"] != "v" {
		t.Errorf("expected context.k=v, got %v", ctx["k"])
	}
}

func TestJSONWriterEscapesStrings(t *testing.T) {
	var buf bytes.Buffer
	w := NewJSONWriter(&buf)
	entity := newLogEntity(INFO, "", nil, time.Now(), `hello "world"`, "file.go:10")
	if err := w.Write(entity); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	out := buf.String()
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, out)
	}
	if parsed["message"] != `hello "world"` {
		t.Errorf("expected escaped message, got %v", parsed["message"])
	}
}

func TestNewlineSeparatedJSONWriter(t *testing.T) {
	var buf bytes.Buffer
	w := NewlineSeparatedJSONWriter(&buf)
	entity := newLogEntity(INFO, "", nil, time.Now(), "hello", "file.go:10")
	if err := w.Write(entity); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	if !strings.HasSuffix(buf.String(), "\n") {
		t.Errorf("expected newline suffix, got %q", buf.String())
	}
}
