package logger

import (
	"fmt"
	"io"
	"log"
)

// Simple logger wrapper
type SimpleLogger struct {
	*log.Logger
	w       io.Writer
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

func (l *SimpleLogger) Copy() *SimpleLogger {
	return New(l.w, l.Prefix(), l.verbose)
}

func (l *SimpleLogger) AppendPrefix(prefix string) {
	l.SetPrefix(fmt.Sprintf("%s%s", l.Prefix(), prefix))
}

func (l *SimpleLogger) WithPrefix(prefix string) *SimpleLogger {
	c := l.Copy()
	c.AppendPrefix(prefix)
	return c
}

type silentWriter struct{}

func (w *silentWriter) Write(p []byte) (n int, err error) {
	return 0, nil
}

var sw = &silentWriter{}

func New(out io.Writer, prefix string, verbose bool) *SimpleLogger {
	l := &SimpleLogger{log.New(sw, prefix, log.Ldate|log.Ltime), out, verbose}
	if verbose {
		l.SetOutput(out)
	}
	return l
}

func LogError(logger *SimpleLogger, fnName string, err error) {
	if err != nil {
		logger.Printf("error happened at %s due to %s", fnName, err.Error())
	}
}
