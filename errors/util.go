package errors

import (
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"strings"
)

const DefaultStackDepth = 16

type stack []uintptr

func (s *stack) Format() string {
	frames := runtime.CallersFrames(*s)
	var b strings.Builder
	for {
		frame, more := frames.Next()
		b.WriteRune('\n')
		b.WriteString(frame.Function)
		b.WriteRune('\n')
		b.WriteRune('\t')
		b.WriteString(frame.File)
		b.WriteRune(':')
		b.WriteString(strconv.Itoa(frame.Line))
		if !more {
			break
		}
	}
	return b.String()
}

type TrackableError struct {
	err        error
	stacktrace *stack
}

func (q *TrackableError) Error() string {
	return fmt.Sprintf("%s\nroot causing stacktrace:\n%s", q.err.Error(), q.stacktrace.Format())
}

func (q *TrackableError) Stacktrace() string {
	return q.stacktrace.Format()
}

func (q *TrackableError) CausingError() error {
	return q.err
}

func Error(msg string) *TrackableError {
	return newTrackableErr(errors.New(msg), stacktraceWithDepth(DefaultStackDepth, 1))
}

func newTrackableErr(err error, stacktrace *stack) *TrackableError {
	if te, ok := err.(*TrackableError); ok {
		stacktrace = te.stacktrace
	}
	return &TrackableError{
		err:        err,
		stacktrace: stacktrace,
	}
}

func stacktraceWithDepth(depth int, frameSkips int) *stack {
	pcs := make([]uintptr, depth)
	n := runtime.Callers(frameSkips+2, pcs[:]) // Skip 2 frames(excluding runtime.Callers, stacktraceWithDepth(xxx,xxx))
	var st stack = pcs[:n]
	return &st
}

func StackTrace(frameSkips int) string {
	return stacktraceWithDepth(DefaultStackDepth, frameSkips+1).Format()
}

func Errorf(formatter string, fields ...any) *TrackableError {
	return newTrackableErr(fmt.Errorf(formatter, fields...), stacktraceWithDepth(DefaultStackDepth, 1))
}

func ErrorWith(errMsgs ...string) *TrackableError {
	var errMsgBuilder strings.Builder
	for _, msg := range errMsgs {
		errMsgBuilder.WriteString(msg)
		errMsgBuilder.WriteByte(';')
	}
	return newTrackableErr(errors.New(errMsgBuilder.String()), stacktraceWithDepth(DefaultStackDepth, 1))
}

func WrapWithStackTrace(err error) *TrackableError {
	return newTrackableErr(err, stacktraceWithDepth(DefaultStackDepth, 1))
}
