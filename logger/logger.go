package logger

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
}
