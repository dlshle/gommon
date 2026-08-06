package logging

type TeeWriter struct {
	writers []LogWriter
}

func (tw *TeeWriter) Write(entity *LogEntity) error {
	var firstErr error
	for _, w := range tw.writers {
		if err := w.Write(entity); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func NewTeeWriter(writers ...LogWriter) LogWriter {
	return &TeeWriter{
		writers: writers,
	}
}
