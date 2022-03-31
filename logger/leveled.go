package logger

import (
	"fmt"
	"github.com/dlshle/gommon/stringz"
	"io"
	"log"
)

type LevelLogger struct {
	logger            *log.Logger
	format            int
	prefix            string
	logLevelWaterMark int
}

func NewLevelLogger(writer io.Writer, prefix string, format int, waterMark int) Logger {
	return LevelLogger{
		logger:            log.New(writer, prefix, format),
		prefix:            prefix,
		logLevelWaterMark: waterMark,
	}
}

func (l LevelLogger) output(level int, data ...string) {
	if level < l.logLevelWaterMark {
		return
	}
	if data == nil {
		l.logger.Output(1, "nil\n")
		return
	}
	if len(data) == 1 {
		l.logger.Output(1, LogLevelPrefixMap[level]+data[0]+"\n")
		return
	}
	concatenatedData := stringz.ConcatString(data...)
	l.logger.Output(1, LogLevelPrefixMap[level]+concatenatedData+"\n")
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
	l.format = format
	l.logger.SetFlags(format)
}

// create new logger
func (l LevelLogger) WithPrefix(prefix string) Logger {
	return NewLevelLogger(l.logger.Writer(), prefix, l.format, l.logLevelWaterMark)
}

func (l LevelLogger) WithFormat(format int) Logger {
	return NewLevelLogger(l.logger.Writer(), l.prefix, format, l.logLevelWaterMark)
}
