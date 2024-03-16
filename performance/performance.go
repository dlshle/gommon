package performance

import (
	"time"
)

func Measure(task func()) time.Duration {
	from := time.Now()
	task()
	return time.Since(from)
}
