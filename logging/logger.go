package logging

import (
	"bytes"
	"context"
	"io"
	"os"
	"sync"
	"time"

	"github.com/dlshle/gommon/errors"
	"github.com/dlshle/gommon/utils"
)

var logEntityPool sync.Pool

func init() {
	logEntityPool = sync.Pool{
		New: func() any {
			return new(LogEntity)
		},
	}
}

var GlobalLogger Logger = CreateLevelLogger(NewConsoleLogWriter(os.Stdout), "", LogAllWaterMark)

func SetLogger(logger Logger) {
	GlobalLogger = logger
}

const CtxValLoggingContext = "$logging_ctx"

const (
	TRACE = iota
	DEBUG
	INFO
	WARN
	ERROR
	FATAL

	pTrace = "TRACE"
	pDebug = "DEBUG"
	pInfo  = "INFO"
	pWarn  = "WARN"
	pError = "ERROR"
	pFatal = "FATAL"
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
	SetWaterMark(int)
	WaterMarkWithPropogate(int)
	DeleteContext(k string)
	Prefix(prefix string)
	PrefixWithPropogate(prefix string)
	Format(format int)
	Writer(writer LogWriter)
	WriterWithPropogate(writer LogWriter)

	// create new logger
	WithPrefix(prefix string) Logger
	WithFormat(format int) Logger
	WithWriter(writer LogWriter) Logger

	WithContext(context map[string]string) Logger
	WithGRContextLogging(bool) Logger
	WithWaterMark(int) Logger
}

type LogEntity struct {
	Level     int
	Prefix    string
	Context   map[string]string
	Timestamp time.Time
	Message   string
	File      string
}

func (e *LogEntity) recycle() {
	logEntityPool.Put(e)
}

func newLogEntity(level int, prefix string, context map[string]string, timestamp time.Time, message string, file string) *LogEntity {
	entity := logEntityPool.Get().(*LogEntity)
	entity.Level = level
	entity.Prefix = prefix
	entity.Context = context
	entity.Timestamp = timestamp
	entity.Message = message
	entity.File = file
	return entity
}

type LogWriter interface {
	Write(entity *LogEntity)
}

type SimpleStringWriter struct {
	consoleWriter io.Writer
}

func NewConsoleLogWriter(writer io.Writer) LogWriter {
	return SimpleStringWriter{
		writer,
	}
}

func (w SimpleStringWriter) Write(logEntity *LogEntity) {
	var builder bytes.Buffer
	builder.WriteString(logEntity.Timestamp.Format(time.RFC3339))
	builder.WriteRune(' ')
	builder.WriteRune('[')
	builder.WriteString(LogLevelPrefixMap[logEntity.Level])
	builder.WriteRune(']')
	builder.WriteRune(' ')
	builder.WriteString(logEntity.Prefix)
	builder.WriteRune(' ')
	builder.WriteString(logEntity.File)
	builder.WriteRune(' ')
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

type JSONWriter struct {
	ioWriter io.Writer
}

func NewJSONWriter(ioWriter io.Writer) LogWriter {
	return JSONWriter{
		ioWriter: ioWriter,
	}
}

func (w JSONWriter) Write(entity *LogEntity) {
	w.ioWriter.Write(w.getJSONEntityBytes(entity))
}

func (w JSONWriter) getJSONEntityBytes(entity *LogEntity) []byte {
	var builder bytes.Buffer
	builder.WriteRune('{')
	w.writeKVPair(&builder, w.quoteWith("timestamp"), w.quoteWith(entity.Timestamp.Format(time.RFC3339)))
	builder.WriteRune(',')
	w.writeKVPair(&builder, w.quoteWith("file"), w.quoteWith(entity.File))
	builder.WriteRune(',')
	w.writeKVPair(&builder, w.quoteWith("level"), w.quoteWith(LogLevelPrefixMap[entity.Level]))
	builder.WriteRune(',')
	w.writeKVPair(&builder, w.quoteWith("prefix"), w.quoteWith(utils.EncodeJSONString(entity.Prefix)))
	builder.WriteRune(',')
	w.writeKVPair(&builder, w.quoteWith("message"), w.quoteWith(utils.EncodeJSONString(entity.Message)))
	builder.WriteRune(',')
	w.writeKVPair(&builder, w.quoteWith("context"), utils.StringMapToJSON(entity.Context))
	builder.WriteRune('}')
	return builder.Bytes()
}

func (w JSONWriter) quoteWith(val string) string {
	return "\"" + val + "\""
}

func (w JSONWriter) writeKVPair(b *bytes.Buffer, k, v string) {
	b.WriteString(k)
	b.WriteRune(':')
	b.WriteString(v)
}

type NoopWriter struct{}

func NewNoopWriter() NoopWriter {
	return NoopWriter{}
}

func (w NoopWriter) Write(entity *LogEntity) {}
