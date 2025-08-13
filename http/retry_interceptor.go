package http

import (
	"github.com/dlshle/gommon/errors"
	"github.com/dlshle/gommon/retry"
)

func RetryInterceptor(options *retry.RetryOptions, retryStatusCodes map[int]bool) Interceptor {
	return func(request *Request, next func(*Request) (*Response, error)) (*Response, error) {
		return retry.Retry1(func() (*Response, error) {
			resp, err := next(request)
			if err != nil {
				return nil, err
			}
			if resp.Code == 0 || retryStatusCodes[resp.Code] {
				return nil, errors.Errorf("retryable error for code %d", resp.Code)
			}
			return resp, nil
		}, retry.WithRetryOptions(options))
	}
}
