package logger

import (
	"io"
	"os"
)

type TeeWriter struct {
	writers []io.Writer
}

func (tw *TeeWriter) Write(p []byte) (n int, err error) {
	for _, w := range tw.writers {
		_, err = w.Write(p)
	}
	return 0, err
}

func NewTeeWriter(writers []io.Writer) *TeeWriter {
	return &TeeWriter{
		writers,
	}
}

func NewFileConsoleTeeWriter(logPath string) *TeeWriter {
	return &TeeWriter{
		[]io.Writer{
			os.Stdout,
		},
	}
}
