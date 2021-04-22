package logger

import (
	"io"
	"log"
)

// Simple logger wrapper
type SimpleLogger struct {
	*log.Logger
	w io.Writer
	verbose bool
}

func (l *SimpleLogger) Verbose(use bool) {
	toUse := use && !l.verbose
	toSilent := !use && l.verbose
	if toUse {
		l.SetOutput(l.w)
	} else if toSilent {
		l.SetOutput(sw)
	}
}

type silentWriter struct {}

func (w *silentWriter) Write(p []byte) (n int, err error) {
	return 0, nil
}

var sw *silentWriter= &silentWriter{}

func New(out io.Writer, prefix string, verbose bool) *SimpleLogger {
	l := &SimpleLogger{log.New(sw, prefix, log.Ldate | log.Ltime), out, verbose}
	if verbose {
		l.SetOutput(out)
	}
	return l
}