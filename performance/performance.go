package performance

import (
	"log"
	"os"
	"time"
)

var globalLogger = log.New(os.Stdout, "[Performance]", log.Ldate|log.Ltime|log.Lshortfile)

func Measure(task func()) time.Duration {
	from := time.Now()
	task()
	return time.Now().Sub(from)
}

func MeasureWithLog(id string, task func()) time.Duration {
	duration := Measure(task)
	globalLogger.Println("[task-", id, "] duration = ", duration)
	return duration
}
