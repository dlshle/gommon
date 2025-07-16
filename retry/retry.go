package retry

import "time"

type RetryOptions struct {
	MaxRetries int
	Interval   time.Duration
	Backoff    float32
}

type RetryOpt func(*RetryOptions) *RetryOptions

func WithRetryOptions(options *RetryOptions) RetryOpt {
	return func(ro *RetryOptions) *RetryOptions {
		return options
	}
}

func singleBackoffRetryOpt(ro *RetryOptions) *RetryOptions {
	return &RetryOptions{
		Backoff:    1,
		Interval:   ro.Interval,
		MaxRetries: ro.MaxRetries,
	}
}

func Retry(task func() error, opts ...RetryOpt) error {
	return RetryWithBackoff(task, append(opts, singleBackoffRetryOpt)...)
}

func RetryWithBackoff(task func() error, opts ...RetryOpt) (err error) {
	cfg := &RetryOptions{
		MaxRetries: 1,
		Backoff:    1,
	}
	for _, opt := range opts {
		cfg = opt(cfg)
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 1
	}
	interval := cfg.Interval
	for i := 0; i < cfg.MaxRetries; i++ {
		if err = task(); err != nil {
			time.Sleep(cfg.Interval)
			interval = time.Duration(float32(interval) * cfg.Backoff)
			continue
		}
		return nil
	}
	return
}

func Retry1[T any](task func() (T, error), opts ...RetryOpt) (T, error) {
	return RetryWithBackoff1(task, append(opts, singleBackoffRetryOpt)...)
}

func RetryWithBackoff1[T any](task func() (T, error), opts ...RetryOpt) (res T, err error) {
	t := func() error {
		res, err = task()
		return err
	}
	err = RetryWithBackoff(t, opts...)
	return
}
