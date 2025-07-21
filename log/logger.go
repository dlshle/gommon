package log

import (
	"context"

	"github.com/dlshle/gommon/errors"
	"github.com/dlshle/gommon/logging"
)

// a simplified logging wrapper for logging.GlobalLogger

func Trace(ctx context.Context, records ...string) {
	logging.GlobalLogger.Trace(ctx, records...)
}

func Debug(ctx context.Context, records ...string) {
	logging.GlobalLogger.Debug(ctx, records...)
}

func Info(ctx context.Context, records ...string) {
	logging.GlobalLogger.Info(ctx, records...)
}

func Warn(ctx context.Context, records ...string) {
	logging.GlobalLogger.Warn(ctx, records...)
}

func Error(ctx context.Context, records ...string) {
	logging.GlobalLogger.Error(ctx, records...)
}

func TrackableError(ctx context.Context, err *errors.TrackableError, records ...string) {
	logging.GlobalLogger.TrackableError(ctx, err, records...)
}

func Fatal(ctx context.Context, records ...string) {
	logging.GlobalLogger.Fatal(ctx, records...)
}

func Tracef(ctx context.Context, format string, records ...interface{}) {
	logging.GlobalLogger.Tracef(ctx, format, records...)
}

func Debugf(ctx context.Context, format string, records ...interface{}) {
	logging.GlobalLogger.Debugf(ctx, format, records...)
}

func Infof(ctx context.Context, format string, records ...interface{}) {
	logging.GlobalLogger.Infof(ctx, format, records...)
}

func Warnf(ctx context.Context, format string, records ...interface{}) {
	logging.GlobalLogger.Warnf(ctx, format, records...)
}

func Errorf(ctx context.Context, format string, records ...interface{}) {
	logging.GlobalLogger.Errorf(ctx, format, records...)
}

func TrackableErrorf(ctx context.Context, err *errors.TrackableError, format string, records ...interface{}) {
	logging.GlobalLogger.TrackableErrorf(ctx, err, format, records...)
}

func Fatalf(ctx context.Context, format string, records ...interface{}) {
	logging.GlobalLogger.Fatalf(ctx, format, records...)
}

func SetWaterMark(waterMark int) {
	logging.GlobalLogger.SetWaterMark(waterMark)
}

func Writer(writer logging.LogWriter) {
	logging.GlobalLogger.Writer(writer)
}
