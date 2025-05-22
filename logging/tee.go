package logging

type TeeWriter struct {
	writers []LogWriter
}

func (tw *TeeWriter) Write(entity *LogEntity) {
	for _, w := range tw.writers {
		w.Write(entity)
	}
}

func NewTeeWriter(writers ...LogWriter) LogWriter {
	return &TeeWriter{
		writers,
	}
}
