package logger

import (
	"bytes"
	"io"
	"time"
)

const (
	TRACE = iota
	DEBUG
	INFO
	WARN
	ERROR
	FATAL

	pTrace = "[TRACE]"
	pDebug = "[DEBUG]"
	pInfo  = "[INFO]"
	pWarn  = "[WARN]"
	pError = "[ERROR]"
	pFatal = "[FATAL]"
)

var LogLevelPrefixMap = map[int]string{
	TRACE: pTrace,
	DEBUG: pDebug,
	INFO:  pInfo,
	WARN:  pWarn,
	ERROR: pError,
	FATAL: pFatal,
}

type Logger interface {
	Trace(records ...string)
	Debug(records ...string)
	Info(records ...string)
	Warn(records ...string)
	Error(records ...string)
	Fatal(records ...string)

	Tracef(format string, records ...interface{})
	Debugf(format string, records ...interface{})
	Infof(format string, records ...interface{})
	Warnf(format string, records ...interface{})
	Errorf(format string, records ...interface{})
	Fatalf(format string, records ...interface{})

	Prefix(prefix string)
	Format(format int)

	// create new logger
	WithPrefix(prefix string) Logger
	WithFormat(format int) Logger

	WithContext(context map[string]string) Logger
	WithGRContextLogging(bool) Logger
}

type LogEntity struct {
	Level     int
	Prefix    string
	Context   map[string]string
	Timestamp time.Time
	Message   string
	File      string
}

type LogWriter interface {
	Write(entity *LogEntity)
}

type ConsoleLogWriter struct {
	consoleWriter io.Writer
}

func NewConsoleLogWriter(writer io.Writer) LogWriter {
	return ConsoleLogWriter{
		writer,
	}
}

func (w ConsoleLogWriter) Write(logEntity *LogEntity) {
	var builder bytes.Buffer
	builder.WriteString(logEntity.Timestamp.Format(time.RFC3339))
	builder.WriteRune(' ')
	builder.WriteString(LogLevelPrefixMap[logEntity.Level])
	builder.WriteRune(' ')
	builder.WriteString(logEntity.Prefix)
	builder.WriteRune(' ')
	builder.WriteString(logEntity.File)
	// contexts
	contexts := logEntity.Context
	ctxLen := len(contexts)
	if ctxLen > 0 {
		builder.WriteRune(' ')
		ctxCnt := 0
		builder.WriteRune('{')
		for k, v := range contexts {
			builder.WriteString(k)
			builder.WriteRune(':')
			builder.WriteString(v)
			ctxCnt++
			if ctxCnt < ctxLen {
				builder.WriteRune(';')
			}
		}
		builder.WriteString("} ")
	}
	builder.WriteString(logEntity.Message)
	builder.WriteRune('\n')
	w.consoleWriter.Write(builder.Bytes())
}
