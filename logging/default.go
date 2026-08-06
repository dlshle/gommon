package logging

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/dlshle/gommon/errors"
)

// DefaultLogger is a thread-safe concrete Logger implementation.
type DefaultLogger struct {
	mu                   sync.RWMutex
	writer               LogWriter
	prefix               string
	logLevelWaterMark    Level
	context              map[string]string
	subLoggers           []Logger
	callerDepth          int
	msgTruncateThreshold int
}

const (
	DefaultCallerDepth          = 3
	DefaultMsgTruncateThreshold = 1024 * 15 // 15kb
)

func StdOutLevelLogger(prefix string) Logger {
	return NewDefaultLogger(os.Stdout, prefix, LogAllWaterMark)
}

func NewDefaultLogger(writer io.Writer, prefix string, waterMark Level) *DefaultLogger {
	return &DefaultLogger{
		writer:               NewConsoleLogWriter(writer),
		prefix:               prefix,
		logLevelWaterMark:    waterMark,
		context:              make(map[string]string),
		subLoggers:           make([]Logger, 0),
		callerDepth:          DefaultCallerDepth,
		msgTruncateThreshold: DefaultMsgTruncateThreshold,
	}
}

func CreateDefaultLogger(entityWriter LogWriter, prefix string, loggingMark Level) Logger {
	return &DefaultLogger{
		writer:               entityWriter,
		prefix:               prefix,
		logLevelWaterMark:    loggingMark,
		context:              make(map[string]string),
		subLoggers:           make([]Logger, 0),
		callerDepth:          DefaultCallerDepth,
		msgTruncateThreshold: DefaultMsgTruncateThreshold,
	}
}

func (l *DefaultLogger) copy() *DefaultLogger {
	l.mu.RLock()
	defer l.mu.RUnlock()
	ctxCopy := make(map[string]string, len(l.context))
	for k, v := range l.context {
		ctxCopy[k] = v
	}
	return &DefaultLogger{
		writer:               l.writer,
		prefix:               l.prefix,
		logLevelWaterMark:    l.logLevelWaterMark,
		context:              ctxCopy,
		subLoggers:           make([]Logger, 0),
		callerDepth:          l.callerDepth,
		msgTruncateThreshold: l.msgTruncateThreshold,
	}
}

func (l *DefaultLogger) outputWithExtraContext(ctx context.Context, level Level, extraContext map[string]string, data ...string) {
	l.mu.RLock()
	waterMark := l.logLevelWaterMark
	threshold := l.msgTruncateThreshold
	l.mu.RUnlock()
	if level < waterMark {
		return
	}

	var builder bytes.Buffer
	for _, piece := range data {
		builder.WriteString(piece)
	}
	if builder.Len() > threshold {
		builder.Truncate(threshold)
		builder.WriteString("...")
	}

	prefix, ctxCopy := l.snapshotPrefixAndContext()
	for k, v := range extraContext {
		ctxCopy[k] = v
	}
	logEntity := newLogEntity(level, prefix, ctxCopy, time.Now(), builder.String(), l.getFileName())
	_ = l.writer.Write(logEntity)
}

func (l *DefaultLogger) snapshotPrefixAndContext() (string, map[string]string) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	ctxCopy := make(map[string]string, len(l.context))
	for k, v := range l.context {
		ctxCopy[k] = v
	}
	return l.prefix, ctxCopy
}

func (l *DefaultLogger) getFileName() string {
	_, file, line, ok := runtime.Caller(l.callerDepth)
	if !ok {
		return "???:0"
	}
	return filepath.Base(file) + ":" + strconv.Itoa(line)
}

func (l *DefaultLogger) Debug(ctx context.Context, records ...string) {
	l.outputWithExtraContext(ctx, DEBUG, nil, records...)
}

func (l *DefaultLogger) Trace(ctx context.Context, records ...string) {
	l.outputWithExtraContext(ctx, TRACE, nil, records...)
}

func (l *DefaultLogger) Info(ctx context.Context, records ...string) {
	l.outputWithExtraContext(ctx, INFO, nil, records...)
}

func (l *DefaultLogger) Warn(ctx context.Context, records ...string) {
	l.outputWithExtraContext(ctx, WARN, nil, records...)
}

func (l *DefaultLogger) Error(ctx context.Context, records ...string) {
	l.outputWithExtraContext(ctx, ERROR, nil, records...)
}

func (l *DefaultLogger) TrackableError(ctx context.Context, err *errors.TrackableError, records ...string) {
	l.outputWithExtraContext(ctx, ERROR, l.stackTraceContext(err), records...)
}

func (l *DefaultLogger) Fatal(ctx context.Context, records ...string) {
	l.outputWithExtraContext(ctx, FATAL, nil, records...)
}

func (l *DefaultLogger) Debugf(ctx context.Context, format string, records ...interface{}) {
	l.outputWithExtraContext(ctx, DEBUG, nil, fmt.Sprintf(format, records...))
}

func (l *DefaultLogger) Tracef(ctx context.Context, format string, records ...interface{}) {
	l.outputWithExtraContext(ctx, TRACE, nil, fmt.Sprintf(format, records...))
}

func (l *DefaultLogger) Infof(ctx context.Context, format string, records ...interface{}) {
	l.outputWithExtraContext(ctx, INFO, nil, fmt.Sprintf(format, records...))
}

func (l *DefaultLogger) Warnf(ctx context.Context, format string, records ...interface{}) {
	l.outputWithExtraContext(ctx, WARN, nil, fmt.Sprintf(format, records...))
}

func (l *DefaultLogger) Errorf(ctx context.Context, format string, records ...interface{}) {
	l.outputWithExtraContext(ctx, ERROR, nil, fmt.Sprintf(format, records...))
}

func (l *DefaultLogger) TrackableErrorf(ctx context.Context, err *errors.TrackableError, format string, records ...interface{}) {
	l.outputWithExtraContext(ctx, ERROR, l.stackTraceContext(err), fmt.Sprintf(format, records...))
}

func (l *DefaultLogger) Fatalf(ctx context.Context, format string, records ...interface{}) {
	l.outputWithExtraContext(ctx, FATAL, nil, fmt.Sprintf(format, records...))
}

func (l *DefaultLogger) stackTraceContext(err *errors.TrackableError) map[string]string {
	if err == nil {
		return nil
	}
	return map[string]string{"stacktrace": err.Stacktrace()}
}

func (l *DefaultLogger) SetContext(k, v string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.context[k] = v
}

func (l *DefaultLogger) DeleteContext(k string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.context, k)
}

func (l *DefaultLogger) Prefix(prefix string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.prefix = prefix
}

func (l *DefaultLogger) PrefixWithPropogate(prefix string) {
	l.mu.Lock()
	l.prefix = prefix
	subs := make([]Logger, len(l.subLoggers))
	copy(subs, l.subLoggers)
	l.mu.Unlock()
	for _, subLogger := range subs {
		subLogger.PrefixWithPropogate(prefix)
	}
}

func (l *DefaultLogger) SetWriter(writer LogWriter) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.writer = writer
}

func (l *DefaultLogger) SetWriterWithPropogate(writer LogWriter) {
	l.mu.Lock()
	l.writer = writer
	subs := make([]Logger, len(l.subLoggers))
	copy(subs, l.subLoggers)
	l.mu.Unlock()
	for _, subLogger := range subs {
		subLogger.SetWriterWithPropogate(writer)
	}
}

func (l *DefaultLogger) WithPrefix(prefix string) Logger {
	subLogger := l.copy()
	subLogger.prefix = prefix
	l.registerSubLogger(subLogger)
	return subLogger
}

func (l *DefaultLogger) WithWriter(writer LogWriter) Logger {
	subLogger := l.copy()
	subLogger.writer = writer
	l.registerSubLogger(subLogger)
	return subLogger
}

func (l *DefaultLogger) WithContext(context map[string]string) Logger {
	subLogger := l.copy()
	for k, v := range context {
		subLogger.context[k] = v
	}
	l.registerSubLogger(subLogger)
	return subLogger
}

func (l *DefaultLogger) WithWaterMark(waterMark Level) Logger {
	subLogger := l.copy()
	subLogger.logLevelWaterMark = waterMark
	l.registerSubLogger(subLogger)
	return subLogger
}

func (l *DefaultLogger) WithMessageTruncateThreshold(threshold int) Logger {
	subLogger := l.copy()
	subLogger.msgTruncateThreshold = threshold
	l.registerSubLogger(subLogger)
	return subLogger
}

func (l *DefaultLogger) WithCallerDepth(callerDepth int) Logger {
	subLogger := l.copy()
	subLogger.callerDepth = callerDepth
	l.registerSubLogger(subLogger)
	return subLogger
}

func (l *DefaultLogger) registerSubLogger(sub Logger) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.subLoggers = append(l.subLoggers, sub)
}

func (l *DefaultLogger) SetWaterMark(waterMark Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logLevelWaterMark = waterMark
}

func (l *DefaultLogger) SetMessageTruncateThreshold(msgTruncateThreshold int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.msgTruncateThreshold = msgTruncateThreshold
}

func (l *DefaultLogger) WaterMarkWithPropogate(waterMark Level) {
	l.mu.Lock()
	l.logLevelWaterMark = waterMark
	subs := make([]Logger, len(l.subLoggers))
	copy(subs, l.subLoggers)
	l.mu.Unlock()
	for _, subLogger := range subs {
		subLogger.WaterMarkWithPropogate(waterMark)
	}
}
