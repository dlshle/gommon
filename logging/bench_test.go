package logging

import (
	"bytes"
	"encoding/json"
	"sync"
	"testing"
	"time"
)

func makeEntity() *LogEntity {
	return newLogEntity(INFO, "service", map[string]string{
		"request_id": "abc123",
		"user_id":    "42",
	}, time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC), `hello "world"`, "file.go:10")
}

func BenchmarkLogEntityNew(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = makeEntity()
	}
}

func BenchmarkLogEntityPooled(b *testing.B) {
	pool := sync.Pool{
		New: func() any {
			return new(LogEntity)
		},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e := pool.Get().(*LogEntity)
		e.Level = INFO
		e.Prefix = "service"
		e.Context = map[string]string{
			"request_id": "abc123",
			"user_id":    "42",
		}
		e.Timestamp = time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
		e.Message = `hello "world"`
		e.File = "file.go:10"
		e.Level = 0
		e.Prefix = ""
		e.Context = nil
		e.Timestamp = time.Time{}
		e.Message = ""
		e.File = ""
		pool.Put(e)
	}
}

func BenchmarkJSONWriterBuiltin(b *testing.B) {
	var buf bytes.Buffer
	w := NewJSONWriter(&buf)
	entity := makeEntity()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		_ = w.Write(entity)
	}
}

// manualJSONWriter reproduces the old hand-rolled JSON builder for comparison.
type manualJSONWriter struct {
	ioWriter *bytes.Buffer
	sep      string
}

func (w *manualJSONWriter) Write(entity *LogEntity) error {
	var buffer bytes.Buffer
	buffer.WriteRune('{')
	writeKVPair(&buffer, quoteWith("timestamp"), quoteWith(entity.Timestamp.Format(time.RFC3339)))
	buffer.WriteRune(',')
	writeKVPair(&buffer, quoteWith("file"), quoteWith(entity.File))
	buffer.WriteRune(',')
	writeKVPair(&buffer, quoteWith("level"), quoteWith(entity.Level.String()))
	buffer.WriteRune(',')
	prefixStr, _ := json.Marshal(entity.Prefix)
	writeKVPairWithByteValue(&buffer, quoteWith("prefix"), prefixStr)
	buffer.WriteRune(',')
	msgStr, _ := json.Marshal(entity.Message)
	writeKVPairWithByteValue(&buffer, quoteWith("message"), msgStr)
	buffer.WriteRune(',')
	writeKVPair(&buffer, quoteWith("context"), `{"request_id":"abc123","user_id":"42"}`)
	buffer.WriteRune('}')
	buffer.WriteString(w.sep)
	_, err := w.ioWriter.Write(buffer.Bytes())
	return err
}

func quoteWith(val string) string {
	return "\"" + val + "\""
}

func writeKVPair(b *bytes.Buffer, k, v string) {
	b.WriteString(k)
	b.WriteRune(':')
	b.WriteString(v)
}

func writeKVPairWithByteValue(b *bytes.Buffer, k string, v []byte) {
	b.WriteString(k)
	b.WriteRune(':')
	b.Write(v)
}

func BenchmarkJSONWriterManual(b *testing.B) {
	var buf bytes.Buffer
	w := &manualJSONWriter{ioWriter: &buf}
	entity := makeEntity()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		_ = w.Write(entity)
	}
}
