package log

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/dlshle/gommon/errors"
	"github.com/dlshle/gommon/logging"
)

func TestPackageLevelLogging(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.NewDefaultLogger(&buf, "[test]", logging.LogAllWaterMark).WithCallerDepth(4)
	DefaultGlobalLogger = logger

	Info(context.Background(), "hello")
	out := buf.String()
	if !strings.Contains(out, "[INFO]") {
		t.Errorf("expected [INFO] in output, got %q", out)
	}
	if !strings.Contains(out, "hello") {
		t.Errorf("expected message in output, got %q", out)
	}
}

func TestPackageLevelTrackableError(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.NewDefaultLogger(&buf, "[test]", logging.LogAllWaterMark).WithCallerDepth(4)
	DefaultGlobalLogger = logger

	TrackableError(context.Background(), errors.Error("boom"), "failure")
	out := buf.String()
	if !strings.Contains(out, "[ERROR]") {
		t.Errorf("expected [ERROR] in output, got %q", out)
	}
	if !strings.Contains(out, "failure") {
		t.Errorf("expected user message in output, got %q", out)
	}
}

func TestSetWriter(t *testing.T) {
	var buf bytes.Buffer
	SetWriter(logging.NewConsoleLogWriter(&buf))
	Warn(context.Background(), "warning")
	out := buf.String()
	if !strings.Contains(out, "[WARN]") {
		t.Errorf("expected [WARN] in output, got %q", out)
	}
}
