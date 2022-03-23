package logger

import "testing"

func NewTestLogger(t *testing.T) *SimpleLogger {
	return New(testLoggerWriterWrapper{
		t: t,
	}, "", true)
}

type testLoggerWriterWrapper struct {
	t *testing.T
}

func (l testLoggerWriterWrapper) Write(p []byte) (int, error) {
	l.t.Logf("%s", string(p))
	return 0, nil
}
