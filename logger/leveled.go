package logger

import (
	"bytes"
	"fmt"
	"github.com/dlshle/gommon/stringz"
	"io"
	"os"
	"runtime"
	"strconv"
	"time"
)

type LevelLogger struct {
	writer            LogWriter
	prefix            string
	logLevelWaterMark int
	context           map[string]string
	enableGRContext   bool
	truncateLimit     int
}

const LogAllWaterMark = -1

var nilStringBytes = []byte{'n', 'i', 'l'}

func StdOutLevelLogger(prefix string) Logger {
	return CreateLevelLogger(NewConsoleLogWriter(os.Stdout), prefix, LogAllWaterMark)
}

func NewLevelLogger(writer io.Writer, prefix string, format int, waterMark int) Logger {
	return LevelLogger{
		writer:            NewConsoleLogWriter(writer),
		prefix:            prefix,
		logLevelWaterMark: waterMark,
		context:           make(map[string]string),
		enableGRContext:   false,
	}
}

func CreateLevelLogger(entityWriter LogWriter, prefix string, loggingMark int) Logger {
	return LevelLogger{
		writer:            entityWriter,
		prefix:            prefix,
		logLevelWaterMark: loggingMark,
		context:           make(map[string]string),
		enableGRContext:   true,
	}
}

func (l LevelLogger) output(level int, data ...string) {
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
	logEntity := &LogEntity{
		Level:     level,
		File:      l.getFileName(),
		Timestamp: time.Now(),
		Prefix:    l.prefix,
		Context:   l.prepareContext(),
		Message:   builder.String(),
	}
	l.writer.Write(logEntity)
}

func (l LevelLogger) getFileName() string {
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

func (l LevelLogger) prepareContext() map[string]string {
	allContext := make(map[string]string)
	for k, v := range getGlobalContexts() {
		allContext[k] = v
	}
	for k, v := range l.context {
		allContext[k] = v
	}
	if l.enableGRContext {
		for k, v := range GetAll() {
			allContext[k] = v
		}
	}
	return allContext
}

func (l LevelLogger) Debug(records ...string) {
	l.output(DEBUG, records...)
}

func (l LevelLogger) Trace(records ...string) {
	l.output(TRACE, records...)
}

func (l LevelLogger) Info(records ...string) {
	l.output(INFO, records...)
}

func (l LevelLogger) Warn(records ...string) {
	l.output(WARN, records...)
}

func (l LevelLogger) Error(records ...string) {
	l.output(ERROR, records...)
}

func (l LevelLogger) Fatal(records ...string) {
	l.output(FATAL, records...)
}

func (l LevelLogger) Debugf(format string, records ...interface{}) {
	l.output(DEBUG, fmt.Sprintf(format, records...))
}

func (l LevelLogger) Tracef(format string, records ...interface{}) {
	l.output(TRACE, fmt.Sprintf(format, records...))
}

func (l LevelLogger) Infof(format string, records ...interface{}) {
	l.output(INFO, fmt.Sprintf(format, records...))
}

func (l LevelLogger) Warnf(format string, records ...interface{}) {
	l.output(WARN, fmt.Sprintf(format, records...))
}

func (l LevelLogger) Errorf(format string, records ...interface{}) {
	l.output(ERROR, fmt.Sprintf(format, records...))
}

func (l LevelLogger) Fatalf(format string, records ...interface{}) {
	l.output(FATAL, fmt.Sprintf(format, records...))
}

func (l LevelLogger) Prefix(prefix string) {
	l.prefix = prefix
}

func (l LevelLogger) Format(format int) {
	// no-op
}

func (l LevelLogger) Writer(writer LogWriter) {
	l.writer = writer
}

// create new logger
func (l LevelLogger) WithPrefix(prefix string) Logger {
	return CreateLevelLogger(l.writer, prefix, l.logLevelWaterMark)
}

func (l LevelLogger) WithFormat(format int) Logger {
	return CreateLevelLogger(l.writer, l.prefix, l.logLevelWaterMark)
}

func (l LevelLogger) WithWriter(writer LogWriter) Logger {
	return CreateLevelLogger(writer, l.prefix, l.logLevelWaterMark)
}

func (l LevelLogger) WithGRContextLogging(useGRCL bool) Logger {
	return LevelLogger{
		writer:            l.writer,
		prefix:            l.prefix,
		logLevelWaterMark: l.logLevelWaterMark,
		context:           l.context,
		enableGRContext:   useGRCL,
	}
}

func (l LevelLogger) WithContext(context map[string]string) Logger {
	return LevelLogger{
		writer:            l.writer,
		prefix:            l.prefix,
		logLevelWaterMark: l.logLevelWaterMark,
		context:           context,
	}
}
