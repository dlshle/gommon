package logging

import (
	"bytes"
	"encoding/json"
	"io"
	"sync"
	"time"

	"github.com/dlshle/gommon/utils"
)

var bufferPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

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

func (w *JSONWriter) Write(entity *LogEntity) {
	w.ioWriter.Write(w.getJSONEntityBytes(entity))
}

func (w *JSONWriter) getJSONEntityBytes(entity *LogEntity) []byte {
	buffer := bufferPool.Get().(*bytes.Buffer)
	defer func() {
		buffer.Reset()
		bufferPool.Put(buffer)
	}()
	buffer.WriteRune('{')
	w.writeKVPair(buffer, w.quoteWith("timestamp"), w.quoteWith(entity.Timestamp.Format(time.RFC3339)))
	buffer.WriteRune(',')
	w.writeKVPair(buffer, w.quoteWith("file"), w.quoteWith(entity.File))
	buffer.WriteRune(',')
	w.writeKVPair(buffer, w.quoteWith("level"), w.quoteWith(LogLevelPrefixMap[entity.Level]))
	buffer.WriteRune(',')
	prefixStr, _ := json.Marshal(entity.Prefix)
	w.writeKVPair(buffer, w.quoteWith("prefix"), string(prefixStr))
	buffer.WriteRune(',')
	msgStr, _ := json.Marshal(entity.Message)
	w.writeKVPair(buffer, w.quoteWith("message"), string(msgStr))
	buffer.WriteRune(',')
	w.writeKVPair(buffer, w.quoteWith("context"), utils.StringMapToJSON(entity.Context))
	buffer.WriteRune('}')
	buffer.WriteString(w.sep)
	return buffer.Bytes()
}

func (w *JSONWriter) quoteWith(val string) string {
	return "\"" + val + "\""
}

func (w *JSONWriter) writeKVPair(b *bytes.Buffer, k, v string) {
	b.WriteString(k)
	b.WriteRune(':')
	b.WriteString(v)
}
