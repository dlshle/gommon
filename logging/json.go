package logging

import (
	"encoding/json"
	"io"
)

type JSONWriter struct {
	ioWriter io.Writer
	sep      string
}

func NewJSONWriter(ioWriter io.Writer) LogWriter {
	return &JSONWriter{
		ioWriter: ioWriter,
		sep:      "",
	}
}

func NewlineSeparatedJSONWriter(ioWriter io.Writer) LogWriter {
	return NewJSONWriterWithSep(ioWriter, "\n")
}

func NewJSONWriterWithSep(ioWriter io.Writer, sep string) LogWriter {
	return &JSONWriter{
		ioWriter: ioWriter,
		sep:      sep,
	}
}

func (w *JSONWriter) Write(entity *LogEntity) error {
	bytes, err := json.Marshal(entity)
	if err != nil {
		return err
	}
	if w.sep != "" {
		bytes = append(bytes, []byte(w.sep)...)
	}
	_, err = w.ioWriter.Write(bytes)
	return err
}
