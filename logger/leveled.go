package logger

import (
	"bytes"
	"fmt"
	"github.com/dlshle/gommon/stringz"
	"io"
	"log"
	"os"
	"runtime"
	"strconv"
	"time"
)

type LevelLogger struct {
	writer            io.Writer
	format            int
	prefix            string
	logLevelWaterMark int
	context           map[string]string
}

const LogAllWaterMark = -1

var nilStringBytes = []byte{'n', 'i', 'l'}

func StdOutLevelLogger(prefix string) Logger {
	return NewLevelLogger(os.Stdout, prefix, log.LstdFlags|log.Lshortfile, LogAllWaterMark)
}

func NewLevelLogger(writer io.Writer, prefix string, format int, waterMark int) Logger {
	return LevelLogger{
		writer:            writer,
		prefix:            prefix,
		format:            format,
		logLevelWaterMark: waterMark,
	}
}

func (l LevelLogger) output(level int, data ...string) {
	if level < l.logLevelWaterMark {
		return
	}
	var builder bytes.Buffer
	fileNLine := l.getFileName()
	timeDate := time.Now().Format(time.RFC3339)
	builder.WriteString(timeDate)
	builder.WriteRune(' ')
	builder.WriteString(LogLevelPrefixMap[level])
	builder.WriteRune('[')
	builder.WriteString(fileNLine)
	builder.WriteRune(']')
	if l.prefix != "" {
		builder.WriteString(l.prefix)
	}
	if data == nil {
		builder.Write(nilStringBytes)
	} else if len(data) == 1 {
		builder.WriteString(data[0])
	} else {
		builder.WriteString(stringz.ConcatString(data...))
	}
	builder.WriteRune('\n')
	l.writer.Write(builder.Bytes())
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
}

// create new logger
func (l LevelLogger) WithPrefix(prefix string) Logger {
	return NewLevelLogger(l.writer, prefix, l.format, l.logLevelWaterMark)
}

func (l LevelLogger) WithFormat(format int) Logger {
	return NewLevelLogger(l.writer, l.prefix, format, l.logLevelWaterMark)
}

func (l LevelLogger) WithContext(context map[string]string) Logger {
	return LevelLogger{
		writer:            l.writer,
		prefix:            l.prefix,
		format:            l.format,
		logLevelWaterMark: l.logLevelWaterMark,
		context:           context,
	}
}
