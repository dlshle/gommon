package logging

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/dlshle/gommon/errors"
)

type DefaultLogger struct {
	writer               LogWriter
	prefix               string
	logLevelWaterMark    int
	context              map[string]string
	enableGRContext      bool
	subLoggers           []Logger
	enableAutoStackTrace bool
	callerDepth          int
	msgTruncateThreshold int // max size of msg to truncate
}

const (
	LogAllWaterMark             = -1
	DefaultCallerDepth          = 3
	DefaultMsgTruncateThreshold = 1024 * 15 // 15kb
)

var nilStringBytes = []byte{'n', 'i', 'l'}

func StdOutLevelLogger(prefix string) Logger {
	return CreateDefaultLogger(NewConsoleLogWriter(os.Stdout), prefix, LogAllWaterMark)
}

func NewDefaultLogger(writer io.Writer, prefix string, format int, waterMark int) *DefaultLogger {
	return &DefaultLogger{
		writer:               NewConsoleLogWriter(writer),
		prefix:               prefix,
		logLevelWaterMark:    waterMark,
		context:              make(map[string]string),
		enableGRContext:      false,
		subLoggers:           make([]Logger, 0),
		enableAutoStackTrace: false,
		callerDepth:          DefaultCallerDepth,
		msgTruncateThreshold: DefaultMsgTruncateThreshold,
	}
}

func CreateDefaultLogger(entityWriter LogWriter, prefix string, loggingMark int) Logger {
	return &DefaultLogger{
		writer:               entityWriter,
		prefix:               prefix,
		logLevelWaterMark:    loggingMark,
		context:              make(map[string]string),
		enableGRContext:      true,
		subLoggers:           make([]Logger, 0),
		enableAutoStackTrace: false,
		msgTruncateThreshold: DefaultMsgTruncateThreshold,
	}
}

func (l *DefaultLogger) copy() *DefaultLogger {
	return &DefaultLogger{
		writer:               l.writer,
		prefix:               l.prefix,
		logLevelWaterMark:    l.logLevelWaterMark,
		context:              l.context,
		enableGRContext:      l.enableGRContext,
		subLoggers:           l.subLoggers,
		enableAutoStackTrace: l.enableAutoStackTrace,
		callerDepth:          l.callerDepth,
		msgTruncateThreshold: l.msgTruncateThreshold,
	}
}

func (l *DefaultLogger) output(ctx context.Context, level int, data ...string) {
	if level < l.logLevelWaterMark {
		return
	}
	var builder bytes.Buffer
	if data == nil {
		builder.Write(nilStringBytes)
	} else if len(data) == 1 {
		builder.WriteString(data[0])
	} else {
		for _, piece := range data {
			builder.WriteString(piece)
		}
	}
	builder.Truncate(l.msgTruncateThreshold)
	logEntity := newLogEntity(level, l.prefix, l.prepareContext(ctx), time.Now(), builder.String(), l.getFileName())
	l.writer.Write(logEntity)
	logEntity.recycle()
}

func (l *DefaultLogger) getFileName() string {
	_, file, line, ok := runtime.Caller(l.callerDepth)
	if !ok {
		file = "???"
		line = 0
	}
	short := file
	for i := len(file) - 1; i > 0; i-- {
		if file[i] == '/' {
			short = file[i+1:]
			break
		}
	}
	file = short
	return file + ":" + strconv.Itoa(line)
}

func (l *DefaultLogger) prepareContext(ctx context.Context) map[string]string {
	allContext := make(map[string]string)
	for k, v := range l.context {
		allContext[k] = v
	}
	if ctx != nil {
		iLoggingCtx := ctx.Value(CtxValLoggingContext)
		if iLoggingCtx != nil {
			loggingCtx, ok := iLoggingCtx.(map[string]string)
			if ok {
				for k, v := range loggingCtx {
					allContext[k] = v
				}
			}
		}
	}
	return allContext
}

func (l *DefaultLogger) Debug(ctx context.Context, records ...string) {
	l.output(ctx, DEBUG, records...)
}

func (l *DefaultLogger) Trace(ctx context.Context, records ...string) {
	l.output(ctx, TRACE, records...)
}

func (l *DefaultLogger) Info(ctx context.Context, records ...string) {
	l.output(ctx, INFO, records...)
}

func (l *DefaultLogger) Warn(ctx context.Context, records ...string) {
	l.output(ctx, WARN, records...)
}

func (l *DefaultLogger) Error(ctx context.Context, records ...string) {
	l.output(l.wrapCtxWithStackTraceIfNotPresent(ctx, nil), ERROR, records...)
}

func (l *DefaultLogger) TrackableError(ctx context.Context, err *errors.TrackableError, records ...string) {
	l.output(l.wrapCtxWithStackTraceIfNotPresent(ctx, err), ERROR, append(records, err.Error())...)
}

func (l *DefaultLogger) Fatal(ctx context.Context, records ...string) {
	l.output(l.wrapCtxWithStackTraceIfNotPresent(ctx, nil), FATAL, records...)
}

func (l *DefaultLogger) Debugf(ctx context.Context, format string, records ...interface{}) {
	l.output(ctx, DEBUG, fmt.Sprintf(format, records...))
}

func (l *DefaultLogger) Tracef(ctx context.Context, format string, records ...interface{}) {
	l.output(ctx, TRACE, fmt.Sprintf(format, records...))
}

func (l *DefaultLogger) Infof(ctx context.Context, format string, records ...interface{}) {
	l.output(ctx, INFO, fmt.Sprintf(format, records...))
}

func (l *DefaultLogger) Warnf(ctx context.Context, format string, records ...interface{}) {
	l.output(ctx, WARN, fmt.Sprintf(format, records...))
}

func (l *DefaultLogger) Errorf(ctx context.Context, format string, records ...interface{}) {
	l.output(l.wrapCtxWithStackTraceIfNotPresent(ctx, nil), ERROR, fmt.Sprintf(format, records...))
}

func (l *DefaultLogger) TrackableErrorf(ctx context.Context, err *errors.TrackableError, format string, records ...interface{}) {
	l.output(l.wrapCtxWithStackTraceIfNotPresent(ctx, err), ERROR, fmt.Sprintf(format, records...))
}

func (l *DefaultLogger) Fatalf(ctx context.Context, format string, records ...interface{}) {
	l.output(l.wrapCtxWithStackTraceIfNotPresent(ctx, nil), FATAL, fmt.Sprintf(format, records...))
}

func (l *DefaultLogger) wrapCtxWithStackTraceIfNotPresent(ctx context.Context, err *errors.TrackableError) context.Context {
	if ctx != nil {
		ctx = context.Background()
	}
	var stacktrace string
	if err != nil {
		stacktrace = err.Stacktrace()
	} else {
		if !l.enableAutoStackTrace {
			return ctx
		}
		stacktrace = errors.StackTrace(2)
	}
	ctx.Value(CtxValLoggingContext)
	ctx = WrapCtx(ctx, "stacktrace", stacktrace)
	return ctx
}

func (l *DefaultLogger) SetContext(k, v string) {
	l.context[k] = v
}

func (l *DefaultLogger) DeleteContext(k string) {
	delete(l.context, k)
}

func (l *DefaultLogger) Prefix(prefix string) {
	l.prefix = prefix
}

func (l *DefaultLogger) PrefixWithPropogate(prefix string) {
	l.prefix = prefix
	for _, subLogger := range l.subLoggers {
		subLogger.PrefixWithPropogate(prefix)
	}
}

func (l *DefaultLogger) Format(format int) {
	// no-op
}

func (l *DefaultLogger) Writer(writer LogWriter) {
	l.writer = writer
}

func (l *DefaultLogger) WriterWithPropogate(writer LogWriter) {
	l.writer = writer
	for _, subLogger := range l.subLoggers {
		subLogger.WriterWithPropogate(writer)
	}
}

// create new logger
func (l *DefaultLogger) WithPrefix(prefix string) Logger {
	subLogger := CreateDefaultLogger(l.writer, prefix, l.logLevelWaterMark)
	l.subLoggers = append(l.subLoggers, subLogger)
	return subLogger
}

func (l *DefaultLogger) WithFormat(format int) Logger {
	subLogger := CreateDefaultLogger(l.writer, l.prefix, l.logLevelWaterMark)
	l.subLoggers = append(l.subLoggers, subLogger)
	return subLogger
}

func (l *DefaultLogger) WithWriter(writer LogWriter) Logger {
	subLogger := CreateDefaultLogger(writer, l.prefix, l.logLevelWaterMark)
	l.subLoggers = append(l.subLoggers, subLogger)
	return subLogger
}

func (l *DefaultLogger) WithGRContextLogging(useGRCL bool) Logger {
	subLogger := &DefaultLogger{
		writer:            l.writer,
		prefix:            l.prefix,
		logLevelWaterMark: l.logLevelWaterMark,
		context:           l.context,
		enableGRContext:   useGRCL,
		subLoggers:        make([]Logger, 0),
	}
	l.subLoggers = append(l.subLoggers, subLogger)
	return subLogger
}

func (l *DefaultLogger) WithContext(context map[string]string) Logger {
	subLogger := &DefaultLogger{
		writer:            l.writer,
		prefix:            l.prefix,
		logLevelWaterMark: l.logLevelWaterMark,
		context:           context,
		subLoggers:        make([]Logger, 0),
	}
	l.subLoggers = append(l.subLoggers, subLogger)
	return subLogger
}

func (l *DefaultLogger) SetWaterMark(waterMark int) {
	l.logLevelWaterMark = waterMark
}

func (l *DefaultLogger) SetMessageTruncateThreshold(msgTruncateThreshold int) {
	l.msgTruncateThreshold = msgTruncateThreshold
}

func (l *DefaultLogger) WaterMarkWithPropogate(waterMark int) {
	l.logLevelWaterMark = waterMark
	for _, subLogger := range l.subLoggers {
		subLogger.WaterMarkWithPropogate(waterMark)
	}
}

func (l *DefaultLogger) WithWaterMark(waterMark int) Logger {
	subLogger := l.copy()
	l.subLoggers = append(l.subLoggers, subLogger)
	return subLogger
}

func (l *DefaultLogger) WithCallerDepth(callerDepth int) Logger {
	subLogger := l.copy()
	subLogger.callerDepth = callerDepth
	return subLogger
}

func (l *DefaultLogger) WithMessageTruncateThreshold(threshold int) Logger {
	subLogger := l.copy()
	subLogger.msgTruncateThreshold = threshold
	return subLogger
}
