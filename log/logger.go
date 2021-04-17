package log

import (
	"io"
	"strconv"
	"sync/atomic"
	"time"
)

//log.New(os.Stdout, fmt.Sprintf("HttpClient[%s]", id), log.Ldate|log.Ltime|log.Lshortfile),

// TODO logger that utilizes the log package and can support multiple log levels both async and sync

func init() {
	initDefaultLevels()
	initDateTimeConfigs()
	initDefaultLevelStringMap()
	initDefaultLoggerConfigs()
}

const (
	DefaultLevelDebug = 0
	DefaultLevelInfo = 1
	DefaultLevelWarn = 2
	DefaultLevelError = 3
	DefaultLevelFatal = 4

	LogTimestampOnly = 0x100
	LogDateTime      = 0x011
	LogDateOnly      = 0x010
	LogTimeOnly      = 0x001
)

var DefaultLevels []int
var DefaultLevelStringMap map[int]string
var DateTimeConfigList []int
var DateTimeConfigSet map[int]bool

func initDefaultLevels() {
	DefaultLevels = []int{DefaultLevelDebug, DefaultLevelInfo, DefaultLevelWarn, DefaultLevelError, DefaultLevelFatal}
}

func initDateTimeConfigs() {
	initDateTimeConfigList()
	initDateTimeConfigSet()
}

func initDateTimeConfigList() {
	DateTimeConfigList = []int{LogTimestampOnly, LogDateTime, LogDateOnly, LogTimeOnly}
}

func initDateTimeConfigSet() {
	DateTimeConfigSet = make(map[int]bool)
	for _, c := range DateTimeConfigList {
		DateTimeConfigSet[c] = true
	}
}

func initDefaultLevelStringMap() {
	DefaultLevelStringMap = make(map[int]string)
	DefaultLevelStringMap[DefaultLevelDebug] = "[DEBUG]"
	DefaultLevelStringMap[DefaultLevelInfo] = "[INFO]"
	DefaultLevelStringMap[DefaultLevelWarn] = "[WARN]"
	DefaultLevelStringMap[DefaultLevelError] = "[ERROR]"
	DefaultLevelStringMap[DefaultLevelFatal] = "[FATAL]"
}

type Logger struct {
	writer io.Writer
	dateTimeConfig int // 3-bit int TIME_STAMP DATE TIME
	logFile bool
	prefix string
	idCounter uint32
}

func (l *Logger) log(s []byte) uint32 {
	l.idCounter = atomic.AddUint32(&l.idCounter, 1)
	l.writer.Write(s)
	return l.idCounter
}

func (l *Logger) SetWriter(writer io.Writer) {
	l.writer = writer
}

func (l *Logger) SetPrefix(prefix string) {
	l.prefix = prefix
}

func (l *Logger) updateDateTimeConfigWith(c int, use bool) {
	if use {
		l.dateTimeConfig |= c
	} else {
		l.dateTimeConfig &= ^c
	}
}

func (l *Logger) SetUseDate(use bool) {
	l.updateDateTimeConfigWith(LogDateOnly, use)
}

func (l *Logger) SetUseTime(use bool) {
	l.updateDateTimeConfigWith(LogTimeOnly, use)
}

func (l *Logger) SetUseTimestamp(use bool) {
	l.updateDateTimeConfigWith(LogTimestampOnly, use)
}

func (l *Logger) SetDateTimeConfig(config int) {
	if DateTimeConfigSet[config] {
		l.updateDateTimeConfigWith(config, true)
	}
}

func (l *Logger) SetUseFileName(use bool) {
	l.logFile = use
}

func (l *Logger) DateTimePrefix(t time.Time) string {
	dateTimeConfig := l.dateTimeConfig
	if dateTimeConfig > LogTimestampOnly {
		dateTimeConfig = LogTimestampOnly
	}
	return dLoggerDateTimePrefixHandlerMap[dateTimeConfig](t)
}

type ILogger interface {
	log(s []byte) uint32
	Log(level int, time time.Time, s []string) uint32 // will transform level+string data to []byte
	Debug(s... string) uint32
	Info(s... string) uint32
	Warn(s... string) uint32
	Error(s... string) uint32
	Fatal(s... string) uint32
	DateTimePrefix(t time.Time) string
	SetWriter(writer io.Writer)
	SetPrefix(prefix string)
	SetUseDate(use bool)
	SetUseTime(use bool)
	SetUseTimestamp(use bool)
	SetDateTimeConfig(config int)
	SetUseFileName(use bool)
	Builder() ILogBuilder
}

type LoggerBuilder struct {
	logger ILogger
}

type ILogBuilder interface {
	Writer(io.Writer) ILogBuilder
	LogDate(bool) ILogBuilder
	LogTime(bool) ILogBuilder
	LogTimeStamp(bool) ILogBuilder
	DateTimeConfig(config int) ILogBuilder
	LogFile(bool) ILogBuilder
	Prefix(string) ILogBuilder
	Build() ILogger
}

func NewLoggerBuilder(baseLogger ILogger) *LoggerBuilder {
	return &LoggerBuilder{baseLogger}
}

func (b *LoggerBuilder) Writer(writer io.Writer) ILogBuilder {
	b.logger.SetWriter(writer)
	return b
}

func (b *LoggerBuilder) LogDate(use bool) ILogBuilder {
	b.logger.SetUseDate(use)
	return b
}

func (b *LoggerBuilder) LogTime(use bool) ILogBuilder {
	b.logger.SetUseTime(use)
	return b
}

func (b *LoggerBuilder) LogFile(use bool) ILogBuilder {
	b.logger.SetUseFileName(use)
	return b
}

func (b *LoggerBuilder) LogTimeStamp(use bool) ILogBuilder {
	b.logger.SetUseTimestamp(use)
	return b
}

func (b *LoggerBuilder) DateTimeConfig(config int) ILogBuilder {
	b.logger.SetDateTimeConfig(config)
	return b
}

func (b *LoggerBuilder) Prefix(prefix string) ILogBuilder {
	b.logger.SetPrefix(prefix)
	return b
}

func (b* LoggerBuilder) Build() ILogger {
	return b.logger
}

// Default logger
const (
	DLoggerStatusStarted = 1
	DLoggerStatusStopping = 10
	DLoggerStatusStopped = 20

)

func initDefaultLoggerConfigs() {
	initDLogDateTimePrefixHandlers()
	initDLogHandlers()
}

var defaultLoggerLevelHandlerMap map[int]func(string)string
var dLoggerDateTimePrefixHandlerMap map[int]func(time.Time)string

type awaitableLogJob struct {
	c chan bool
	data []byte
	lid uint32
}

func initDLogHandlers() {
	defaultLoggerLevelHandlerMap = make(map[int]func(string)string)
	for _, level := range DefaultLevels {
		currLevel := level
		defaultLoggerLevelHandlerMap[currLevel] = func(msg string) string {
			return DefaultLevelStringMap[currLevel] + msg
		}
	}
}

func initDLogDateTimePrefixHandlers() {
	dLoggerDateTimePrefixHandlerMap = make(map[int]func(time.Time)string)
	dLoggerDateTimePrefixHandlerMap[LogTimestampOnly] = func (t time.Time) string {
		return strconv.FormatInt(t.Unix(), 10)
	}
	dLoggerDateTimePrefixHandlerMap[LogDateTime] = func (t time.Time) string {
		return t.Format("2006-01-02 15:04:05")
	}
	dLoggerDateTimePrefixHandlerMap[LogDateOnly] = func (t time.Time) string {
		return t.Format("2006-01-02")
	}
	dLoggerDateTimePrefixHandlerMap[LogTimeOnly] = func (t time.Time) string {
		return t.Format("15:04:05")
	}
}

func (j *awaitableLogJob) get() uint32 {
	<- j.c
	return j.lid
}

type DLogger struct {
	*Logger
	async bool
	dataChannel chan *awaitableLogJob
	status atomic.Value
}

func (d *DLogger) getStatus() uint8 {
	return d.status.Load().(uint8)
}

func (d *DLogger) setStatus(status uint8) {
	d.status.Store(status)
}

func (d *DLogger) worker() {
	for d.getStatus() < DLoggerStatusStopping {
		data := <- d.dataChannel
		data.lid = d.log(data.data)
		close(data.c)
	}
	d.setStatus(DLoggerStatusStopped)
}

func (d *DLogger) stop() {
	d.setStatus(DLoggerStatusStopping)
}


func (d *DLogger) prefixStrings(level int, t time.Time) string {
	return  d.DateTimePrefix(t) + " " + defaultLoggerLevelHandlerMap[level](d.prefix)
}

func (d *DLogger) Log(level int, t time.Time, s []string) uint32 {
	concatenatedMessage := ""
	for _, msg := range s {
		concatenatedMessage += msg
	}
	concatenatedMessage += "\n"
	awaitableLog := &awaitableLogJob{
		c:    make(chan bool),
		data: ([]byte)(d.prefixStrings(level, t) + " " + concatenatedMessage),
		lid:  0,
	}
	d.dataChannel <- awaitableLog
	if d.async {
		return 0
	}
	return awaitableLog.get()
}

func (d *DLogger) Debug(s... string) uint32 {
	return d.Log(DefaultLevelDebug, time.Now(), s)
}

func (d *DLogger) Info(s... string) uint32 {
	return d.Log(DefaultLevelInfo, time.Now(), s)
}

func (d *DLogger) Warn(s... string) uint32 {
	return d.Log(DefaultLevelWarn, time.Now(), s)
}

func (d *DLogger) Error(s... string) uint32 {
	return d.Log(DefaultLevelError, time.Now(), s)
}

func (d *DLogger) Fatal(s... string) uint32 {
	return d.Log(DefaultLevelFatal, time.Now(), s)
}

func (d *DLogger) Builder() ILogBuilder {
	return NewLoggerBuilder(d)
}

func NewDLogger(writer io.Writer, dateTimeConfig int, prefix string, useAsync bool) *DLogger {
	baseLogger := &Logger{
		writer,
		dateTimeConfig,
		false,
		prefix,
		0,
	}
	dLogger := &DLogger{
		Logger:      baseLogger,
		async:       useAsync,
		dataChannel: make(chan *awaitableLogJob, 1024),
		status:      atomic.Value{},
	}
	dLogger.setStatus(DLoggerStatusStarted)
	go dLogger.worker()
	return dLogger
}