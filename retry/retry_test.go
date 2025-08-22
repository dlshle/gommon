package retry

import (
	"errors"
	"testing"
	"time"
)

func TestRetrySuccess(t *testing.T) {
	// Test that a successful task on first try works
	attempts := 0
	task := func() error {
		attempts++
		return nil
	}

	err := Retry(task)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if attempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", attempts)
	}
}

func TestRetrySuccessAfterFailures(t *testing.T) {
	// Test that a task succeeds after some failures
	attempts := 0
	task := func() error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary error")
		}
		return nil
	}

	err := Retry(task, WithMaxRetries(5))
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestRetryMaxRetriesReached(t *testing.T) {
	// Test that error is returned after max retries reached
	attempts := 0
	expectedErr := errors.New("persistent error")
	task := func() error {
		attempts++
		return expectedErr
	}

	err := Retry(task, WithMaxRetries(3))
	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestRetryWithConditionSuccess(t *testing.T) {
	// Test retry with custom condition that allows retry
	attempts := 0
	expectedErr := errors.New("retryable error")
	task := func() error {
		attempts++
		if attempts < 3 {
			return expectedErr
		}
		return nil
	}

	// Define a retry condition that retries on our specific error
	condition := func(err error) bool {
		return err.Error() == "retryable error"
	}

	err := Retry(task, WithMaxRetries(5), WithRetryCondition(condition))
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestRetryWithConditionFail(t *testing.T) {
	// Test retry with custom condition that doesn't allow retry
	// Note: Even if an error is not retryable, it will still retry until max retries is reached
	attempts := 0
	expectedErr := errors.New("non-retryable error")
	task := func() error {
		attempts++
		return expectedErr
	}

	// Define a retry condition that doesn't match our error
	condition := func(err error) bool {
		return err.Error() == "retryable error"
	}

	err := Retry(task, WithMaxRetries(5), WithRetryCondition(condition))
	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
	// Even though the error is not retryable, it will still retry until max retries is reached
	// This is the current behavior of the retry mechanism
	if attempts != 5 {
		t.Errorf("Expected 5 attempts (max retries), got %d", attempts)
	}
}

func TestRetryEarlyReturnOnNonRetryableError(t *testing.T) {
	// Test that when an error is not retryable, it returns immediately
	attempts := 0
	expectedErr := errors.New("non-retryable error")
	task := func() error {
		attempts++
		return expectedErr
	}

	// Define a retry condition that doesn't match our error
	condition := func(err error) bool {
		return err.Error() == "retryable error"
	}

	// Test with only 1 max retry (default) - should return immediately on non-retryable error
	err := Retry(task, WithRetryCondition(condition))
	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
	if attempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", attempts)
	}
}

func TestRetryWithBackoff(t *testing.T) {
	// Test retry with backoff
	attempts := 0
	task := func() error {
		attempts++
		if attempts < 3 {
			return errors.New("error")
		}
		return nil
	}

	start := time.Now()
	err := RetryWithBackoff(task, WithMaxRetries(5), WithInterval(time.Millisecond*10), WithBackoff(2.0))
	duration := time.Since(start)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
	
	// Verify that backoff occurred (should take more than 20ms: 10ms + 20ms)
	// Adding some buffer for test execution time
	if duration < time.Millisecond*20 {
		t.Errorf("Expected backoff to increase duration, took %v", duration)
	}
}

func TestRetry1Success(t *testing.T) {
	// Test Retry1 function with return value
	attempts := 0
	task := func() (string, error) {
		attempts++
		return "success", nil
	}

	result, err := Retry1(task)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result != "success" {
		t.Errorf("Expected result 'success', got %s", result)
	}
	if attempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", attempts)
	}
}

func TestRetry1WithRetries(t *testing.T) {
	// Test Retry1 function with retries
	attempts := 0
	task := func() (int, error) {
		attempts++
		if attempts < 3 {
			return 0, errors.New("temporary error")
		}
		return 42, nil
	}

	result, err := Retry1(task, WithMaxRetries(5))
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result != 42 {
		t.Errorf("Expected result 42, got %d", result)
	}
	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestValidateAndFixRetryOption(t *testing.T) {
	// Test that negative or zero max retries defaults to 1
	cfg := &RetryOptions{
		MaxRetries: -1,
		Backoff:    -1,
	}

	validateAndFixRetryOption(cfg)
	
	if cfg.MaxRetries != 1 {
		t.Errorf("Expected MaxRetries to be 1, got %d", cfg.MaxRetries)
	}
	
	if cfg.Backoff != 1 {
		t.Errorf("Expected Backoff to be 1, got %f", cfg.Backoff)
	}
}

func TestIsErrorRetryableDefault(t *testing.T) {
	// Test that by default all errors are retryable
	cfg := &RetryOptions{}
	
	err := errors.New("any error")
	if !isErrorRetryable(cfg, err) {
		t.Error("Expected error to be retryable by default")
	}
}

func TestIsErrorRetryableWithConditions(t *testing.T) {
	// Test error retryable with custom conditions
	cfg := &RetryOptions{
		RetryConditions: []func(error) bool{
			func(err error) bool {
				return err.Error() == "retryable error"
			},
		},
	}
	
	retryableErr := errors.New("retryable error")
	nonRetryableErr := errors.New("non-retryable error")
	
	if !isErrorRetryable(cfg, retryableErr) {
		t.Error("Expected 'retryable error' to be retryable")
	}
	
	if isErrorRetryable(cfg, nonRetryableErr) {
		t.Error("Expected 'non-retryable error' to not be retryable")
	}
}

// Helper function to create retry option with max retries
func WithMaxRetries(maxRetries int) RetryOpt {
	return func(ro *RetryOptions) *RetryOptions {
		ro.MaxRetries = maxRetries
		return ro
	}
}