package retry

import (
	"context"
	"time"
)

type RetryOptions struct {
	// MaxRetries is the total number of attempts, including the first attempt.
	MaxRetries      int
	Interval        time.Duration
	Backoff         float32
	RetryConditions []func(error) bool
}

type RetryOpt func(*RetryOptions) *RetryOptions

func WithRetryOptions(options *RetryOptions) RetryOpt {
	return func(ro *RetryOptions) *RetryOptions {
		return options
	}
}

func WithRetryCondition(cond func(error) bool) RetryOpt {
	return func(ro *RetryOptions) *RetryOptions {
		ro.RetryConditions = append(ro.RetryConditions, cond)
		return ro
	}
}

func WithMaxRetries(maxRetries int) RetryOpt {
	return func(ro *RetryOptions) *RetryOptions {
		ro.MaxRetries = maxRetries
		return ro
	}
}

func WithBackoff(backoff float32) RetryOpt {
	return func(ro *RetryOptions) *RetryOptions {
		ro.Backoff = backoff
		return ro
	}
}

func WithInterval(interval time.Duration) RetryOpt {
	return func(ro *RetryOptions) *RetryOptions {
		ro.Interval = interval
		return ro
	}
}

func singleBackoffRetryOpt(ro *RetryOptions) *RetryOptions {
	return &RetryOptions{
		Backoff:         1,
		Interval:        ro.Interval,
		MaxRetries:      ro.MaxRetries,
		RetryConditions: ro.RetryConditions,
	}
}

func Retry(task func() error, opts ...RetryOpt) error {
	return RetryWithContext(context.Background(), task, opts...)
}

func RetryWithContext(ctx context.Context, task func() error, opts ...RetryOpt) error {
	return RetryWithBackoffContext(ctx, task, append(opts, singleBackoffRetryOpt)...)
}

func RetryWithBackoff(task func() error, opts ...RetryOpt) (err error) {
	return RetryWithBackoffContext(context.Background(), task, opts...)
}

func RetryWithBackoffContext(ctx context.Context, task func() error, opts ...RetryOpt) (err error) {
	cfg := &RetryOptions{
		MaxRetries: 1,
		Backoff:    1,
	}
	for _, opt := range opts {
		cfg = opt(cfg)
	}

	validateAndFixRetryOption(cfg)

	interval := cfg.Interval
	for i := 0; i < cfg.MaxRetries; i++ {
		if err = ctx.Err(); err != nil {
			return err
		}
		if err = task(); err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			if !isErrorRetryable(cfg, err) {
				return err
			}

			// don't sleep after the last attempt
			if i < cfg.MaxRetries-1 {
				if err = wait(ctx, interval); err != nil {
					return err
				}
				interval = time.Duration(float32(interval) * cfg.Backoff)
			}
			continue
		}
		return nil
	}
	return
}

func wait(ctx context.Context, interval time.Duration) error {
	if interval <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(interval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func validateAndFixRetryOption(cfg *RetryOptions) {
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 1
	}
	if cfg.Backoff <= 0 {
		cfg.Backoff = 1
	}
}

func isErrorRetryable(cfg *RetryOptions, err error) bool {
	// by default(if no retry condition), all errors are retriable
	if len(cfg.RetryConditions) == 0 {
		return true
	}
	for _, cond := range cfg.RetryConditions {
		if cond(err) {
			return true
		}
	}
	return false
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
