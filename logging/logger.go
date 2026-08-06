package logging

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/dlshle/gommon/errors"
)

var GlobalLogger Logger = NewDefaultLogger(os.Stdout, "", LogAllWaterMark)

// Level is a typed log severity.
type Level int

const (
	TRACE Level = iota
	DEBUG
	INFO
	WARN
	ERROR
	FATAL
)

const LogAllWaterMark Level = -1

var levelPrefix = map[Level]string{
	TRACE: "TRACE",
	DEBUG: "DEBUG",
	INFO:  "INFO",
	WARN:  "WARN",
	ERROR: "ERROR",
	FATAL: "FATAL",
}

func (l Level) String() string {
	if p, ok := levelPrefix[l]; ok {
		return p
	}
	return fmt.Sprintf("LEVEL(%d)", l)
}

func (l Level) MarshalJSON() ([]byte, error) {
	return []byte(`"` + l.String() + `"`), nil
}

// Logger is the primary logging interface.
type Logger interface {
	Trace(ctx context.Context, records ...string)
	Debug(ctx context.Context, records ...string)
	Info(ctx context.Context, records ...string)
	Warn(ctx context.Context, records ...string)
	Error(ctx context.Context, records ...string)
	TrackableError(ctx context.Context, err *errors.TrackableError, records ...string)
	Fatal(ctx context.Context, records ...string)

	Tracef(ctx context.Context, format string, records ...interface{})
	Debugf(ctx context.Context, format string, records ...interface{})
	Infof(ctx context.Context, format string, records ...interface{})
	Warnf(ctx context.Context, format string, records ...interface{})
	Errorf(ctx context.Context, format string, records ...interface{})
	TrackableErrorf(ctx context.Context, err *errors.TrackableError, format string, records ...interface{})
	Fatalf(ctx context.Context, format string, records ...interface{})

	SetContext(k, v string)
	SetWaterMark(Level)
	SetMessageTruncateThreshold(threshold int)
	WaterMarkWithPropogate(Level)
	DeleteContext(k string)
	Prefix(prefix string)
	PrefixWithPropogate(prefix string)
	SetWriter(writer LogWriter)
	SetWriterWithPropogate(writer LogWriter)

	WithPrefix(prefix string) Logger
	WithWriter(writer LogWriter) Logger
	WithContext(context map[string]string) Logger
	WithWaterMark(Level) Logger
	WithMessageTruncateThreshold(threshold int) Logger
	WithCallerDepth(callerDepth int) Logger
}

// LogEntity is the unit of data passed to a LogWriter.
// Writers receive a pointer for efficiency, but they must synchronously
// consume the entity; they must not retain it after Write returns.
type LogEntity struct {
	Level     Level             `json:"level"`
	Prefix    string            `json:"prefix"`
	Context   map[string]string `json:"context"`
	Timestamp time.Time         `json:"timestamp"`
	Message   string            `json:"message"`
	File      string            `json:"file"`
}

func newLogEntity(level Level, prefix string, context map[string]string, timestamp time.Time, message string, file string) *LogEntity {
	return &LogEntity{
		Level:     level,
		Prefix:    prefix,
		Context:   context,
		Timestamp: timestamp,
		Message:   message,
		File:      file,
	}
}

// LogWriter is the sink interface for log entities.
type LogWriter interface {
	Write(entity *LogEntity) error
}

// SimpleStringWriter formats entities as plain text lines.
type SimpleStringWriter struct {
	consoleWriter io.Writer
}

func NewConsoleLogWriter(writer io.Writer) LogWriter {
	return &SimpleStringWriter{consoleWriter: writer}
}

func (w *SimpleStringWriter) Write(logEntity *LogEntity) error {
	var builder bytes.Buffer
	builder.WriteString(logEntity.Timestamp.Format(time.RFC3339))
	builder.WriteString(" [")
	builder.WriteString(logEntity.Level.String())
	builder.WriteString("] ")
	builder.WriteString(logEntity.Prefix)
	builder.WriteString(" ")
	builder.WriteString(logEntity.File)
	builder.WriteString(" ")
	builder.WriteString(logEntity.Message)
	builder.WriteByte('\n')
	_, err := w.consoleWriter.Write(builder.Bytes())
	return err
}

// NoopWriter discards every entity.
type NoopWriter struct{}

func NewNoopWriter() LogWriter {
	return &NoopWriter{}
}

func (w *NoopWriter) Write(entity *LogEntity) error {
	return nil
}
