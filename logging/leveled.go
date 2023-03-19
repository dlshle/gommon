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

	"github.com/dlshle/gommon/stringz"
)

type LevelLogger struct {
	writer            LogWriter
	prefix            string
	logLevelWaterMark int
	context           map[string]string
	enableGRContext   bool
	subLoggers        []Logger
}

const LogAllWaterMark = -1

var nilStringBytes = []byte{'n', 'i', 'l'}

func StdOutLevelLogger(prefix string) Logger {
	return CreateLevelLogger(NewConsoleLogWriter(os.Stdout), prefix, LogAllWaterMark)
}

func NewLevelLogger(writer io.Writer, prefix string, format int, waterMark int) Logger {
	return &LevelLogger{
		writer:            NewConsoleLogWriter(writer),
		prefix:            prefix,
		logLevelWaterMark: waterMark,
		context:           make(map[string]string),
		enableGRContext:   false,
		subLoggers:        make([]Logger, 0),
	}
}

func CreateLevelLogger(entityWriter LogWriter, prefix string, loggingMark int) Logger {
	return &LevelLogger{
		writer:            entityWriter,
		prefix:            prefix,
		logLevelWaterMark: loggingMark,
		context:           make(map[string]string),
		enableGRContext:   true,
		subLoggers:        make([]Logger, 0),
	}
}

func (l *LevelLogger) output(ctx context.Context, level int, data ...string) {
	if level < l.logLevelWaterMark {
		return
	}
	var builder bytes.Buffer
	if data == nil {
		builder.Write(nilStringBytes)
	} else if len(data) == 1 {
		builder.WriteString(data[0])
	} else {
		builder.WriteString(stringz.ConcatString(data...))
	}
	logEntity := newLogEntity(level, l.prefix, l.prepareContext(ctx), time.Now(), builder.String(), l.getFileName())
	l.writer.Write(logEntity)
	logEntity.recycle()
}

func (l *LevelLogger) getFileName() string {
	_, file, line, ok := runtime.Caller(3)
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

func (l *LevelLogger) prepareContext(ctx context.Context) map[string]string {
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

func (l *LevelLogger) Debug(ctx context.Context, records ...string) {
	l.output(ctx, DEBUG, records...)
}

func (l *LevelLogger) Trace(ctx context.Context, records ...string) {
	l.output(ctx, TRACE, records...)
}

func (l *LevelLogger) Info(ctx context.Context, records ...string) {
	l.output(ctx, INFO, records...)
}

func (l *LevelLogger) Warn(ctx context.Context, records ...string) {
	l.output(ctx, WARN, records...)
}

func (l *LevelLogger) Error(ctx context.Context, records ...string) {
	l.output(ctx, ERROR, records...)
}

func (l *LevelLogger) Fatal(ctx context.Context, records ...string) {
	l.output(ctx, FATAL, records...)
}

func (l *LevelLogger) Debugf(ctx context.Context, format string, records ...interface{}) {
	l.output(ctx, DEBUG, fmt.Sprintf(format, records...))
}

func (l *LevelLogger) Tracef(ctx context.Context, format string, records ...interface{}) {
	l.output(ctx, TRACE, fmt.Sprintf(format, records...))
}

func (l *LevelLogger) Infof(ctx context.Context, format string, records ...interface{}) {
	l.output(ctx, INFO, fmt.Sprintf(format, records...))
}

func (l *LevelLogger) Warnf(ctx context.Context, format string, records ...interface{}) {
	l.output(ctx, WARN, fmt.Sprintf(format, records...))
}

func (l *LevelLogger) Errorf(ctx context.Context, format string, records ...interface{}) {
	l.output(ctx, ERROR, fmt.Sprintf(format, records...))
}

func (l *LevelLogger) Fatalf(ctx context.Context, format string, records ...interface{}) {
	l.output(ctx, FATAL, fmt.Sprintf(format, records...))
}

func (l *LevelLogger) SetContext(k, v string) {
	l.context[k] = v
}

func (l *LevelLogger) DeleteContext(k string) {
	delete(l.context, k)
}

func (l *LevelLogger) Prefix(prefix string) {
	l.prefix = prefix
}

func (l *LevelLogger) PrefixWithPropogate(prefix string) {
	l.prefix = prefix
	for _, subLogger := range l.subLoggers {
		subLogger.PrefixWithPropogate(prefix)
	}
}

func (l *LevelLogger) Format(format int) {
	// no-op
}

func (l *LevelLogger) Writer(writer LogWriter) {
	l.writer = writer
}

func (l *LevelLogger) WriterWithPropogate(writer LogWriter) {
	l.writer = writer
	for _, subLogger := range l.subLoggers {
		subLogger.WriterWithPropogate(writer)
	}
}

// create new logger
func (l *LevelLogger) WithPrefix(prefix string) Logger {
	subLogger := CreateLevelLogger(l.writer, prefix, l.logLevelWaterMark)
	l.subLoggers = append(l.subLoggers, subLogger)
	return subLogger
}

func (l *LevelLogger) WithFormat(format int) Logger {
	subLogger := CreateLevelLogger(l.writer, l.prefix, l.logLevelWaterMark)
	l.subLoggers = append(l.subLoggers, subLogger)
	return subLogger
}

func (l *LevelLogger) WithWriter(writer LogWriter) Logger {
	subLogger := CreateLevelLogger(writer, l.prefix, l.logLevelWaterMark)
	l.subLoggers = append(l.subLoggers, subLogger)
	return subLogger
}

func (l *LevelLogger) WithGRContextLogging(useGRCL bool) Logger {
	subLogger := &LevelLogger{
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

func (l *LevelLogger) WithContext(context map[string]string) Logger {
	subLogger := &LevelLogger{
		writer:            l.writer,
		prefix:            l.prefix,
		logLevelWaterMark: l.logLevelWaterMark,
		context:           context,
		subLoggers:        make([]Logger, 0),
	}
	l.subLoggers = append(l.subLoggers, subLogger)
	return subLogger
}
