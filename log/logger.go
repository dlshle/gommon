package log

import (
	"context"
	"os"

	"github.com/dlshle/gommon/errors"
	"github.com/dlshle/gommon/logging"
)

// DefaultGlobalLogger is a package-level logger configured with a caller depth
// that accounts for this package's wrapper functions.
var DefaultGlobalLogger logging.Logger

func init() {
	DefaultGlobalLogger = logging.NewDefaultLogger(os.Stdout, "", logging.LogAllWaterMark).WithCallerDepth(4)
}

func Trace(ctx context.Context, records ...string) {
	DefaultGlobalLogger.Trace(ctx, records...)
}

func Debug(ctx context.Context, records ...string) {
	DefaultGlobalLogger.Debug(ctx, records...)
}

func Info(ctx context.Context, records ...string) {
	DefaultGlobalLogger.Info(ctx, records...)
}

func Warn(ctx context.Context, records ...string) {
	DefaultGlobalLogger.Warn(ctx, records...)
}

func Error(ctx context.Context, records ...string) {
	DefaultGlobalLogger.Error(ctx, records...)
}

func TrackableError(ctx context.Context, err *errors.TrackableError, records ...string) {
	DefaultGlobalLogger.TrackableError(ctx, err, records...)
}

func Fatal(ctx context.Context, records ...string) {
	DefaultGlobalLogger.Fatal(ctx, records...)
}

func Tracef(ctx context.Context, format string, records ...interface{}) {
	DefaultGlobalLogger.Tracef(ctx, format, records...)
}

func Debugf(ctx context.Context, format string, records ...interface{}) {
	DefaultGlobalLogger.Debugf(ctx, format, records...)
}

func Infof(ctx context.Context, format string, records ...interface{}) {
	DefaultGlobalLogger.Infof(ctx, format, records...)
}

func Warnf(ctx context.Context, format string, records ...interface{}) {
	DefaultGlobalLogger.Warnf(ctx, format, records...)
}

func Errorf(ctx context.Context, format string, records ...interface{}) {
	DefaultGlobalLogger.Errorf(ctx, format, records...)
}

func TrackableErrorf(ctx context.Context, err *errors.TrackableError, format string, records ...interface{}) {
	DefaultGlobalLogger.TrackableErrorf(ctx, err, format, records...)
}

func Fatalf(ctx context.Context, format string, records ...interface{}) {
	DefaultGlobalLogger.Fatalf(ctx, format, records...)
}

func SetWaterMark(waterMark logging.Level) {
	DefaultGlobalLogger.SetWaterMark(waterMark)
}

func SetWriter(writer logging.LogWriter) {
	DefaultGlobalLogger.SetWriter(writer)
}
