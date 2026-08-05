package http

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dlshle/gommon/retry"
)

func TestClient(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))
	defer ts.Close()

	c := NewBuilder().MaxConcurrentRequests(5).MaxConnsPerHost(1).TimeoutSec(60).AddInterceptor(CurlInterceptor).Build()
	defer c.Stop()

	r, e := NewRequestBuilder().Method("POST").URL(ts.URL + "/echo").Header(NewHeaderMaker().Set("hello", "world").Make()).BytesBody([]byte("hello")).Build()
	if e != nil {
		t.Fatalf("Failed to build request: %v", e)
	}
	resp, err := c.DoRequest(r)
	if err != nil {
		t.Fatalf("Failed to request: %v", err)
	}
	if resp == nil || string(resp.Body) != "hello" {
		t.Errorf("Invalid response: %v", resp)
	}
}

func TestRequestBuilder_NoBodyDoesNotPanic(t *testing.T) {
	req, err := NewRequestBuilder().Method("GET").URL("http://example.com").Build()
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	if req.Body == nil {
		t.Fatal("expected non-nil request body")
	}
	if req.GetBody == nil {
		t.Fatal("expected non-nil GetBody")
	}
}

func TestInterceptorMutationIsSent(t *testing.T) {
	expectedHeader := "x-interceptor-test"
	expectedValue := "mutated"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(expectedHeader) != expectedValue {
			http.Error(w, "missing mutated header", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	mutatingInterceptor := func(req *Request, next func(*Request) (*Response, error)) (*Response, error) {
		req.Header.Set(expectedHeader, expectedValue)
		return next(req)
	}

	c := NewBuilder().MaxConcurrentRequests(1).MaxConnsPerHost(1).TimeoutSec(5).AddInterceptor(mutatingInterceptor).Build()
	defer c.Stop()

	req, err := NewRequestBuilder().Method("GET").URL(ts.URL).Header(http.Header{}).Build()
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}

	resp, err := c.DoRequest(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.Code)
	}
}

func TestRetryInterceptorPreservesBody(t *testing.T) {
	attempts := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		body, _ := io.ReadAll(r.Body)
		if attempts < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		if string(body) != "retry-body" {
			http.Error(w, "missing body on retry", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	retryInterceptor := RetryInterceptor(
		&retry.RetryOptions{MaxRetries: 3, Interval: 10 * time.Millisecond},
		map[int]bool{http.StatusServiceUnavailable: true},
	)

	c := NewBuilder().MaxConcurrentRequests(1).MaxConnsPerHost(1).TimeoutSec(5).AddInterceptor(retryInterceptor).Build()
	defer c.Stop()

	req, err := NewRequestBuilder().Method("POST").URL(ts.URL).StringBody("retry-body").Build()
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}

	resp, err := c.DoRequest(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.Code)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

func TestResponseBodySizeLimit(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello world"))
	}))
	defer ts.Close()

	c := NewBuilder().MaxConcurrentRequests(1).MaxConnsPerHost(1).TimeoutSec(5).MaxResponseBodySize(5).Build()
	defer c.Stop()

	req, err := NewRequestBuilder().Method("GET").URL(ts.URL).Build()
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}

	resp, err := c.DoRequest(req)
	if err == nil {
		t.Fatal("expected error for oversized response body")
	}
	if resp != nil {
		t.Fatalf("expected nil response on body-limit error, got %+v", resp)
	}
}

func TestBuilderProducesIndependentClients(t *testing.T) {
	b := NewBuilder().MaxConcurrentRequests(1).MaxQueueSize(1).TimeoutSec(5)

	c1 := b.Build()
	c2 := b.Build()

	if c1 == c2 {
		t.Fatal("Build() returned the same client instance")
	}
	if c1.Status() != PoolStatusRunning || c2.Status() != PoolStatusRunning {
		t.Fatal("expected both clients to be running")
	}

	c1.Stop()
	if c1.Status() != PoolStatusStopped {
		t.Fatalf("expected c1 stopped, got status %d", c1.Status())
	}
	if c2.Status() != PoolStatusRunning {
		t.Fatalf("expected c2 still running after c1 stopped, got status %d", c2.Status())
	}
	c2.Stop()
}

func TestCurlInterceptorDoesNotBreakRetries(t *testing.T) {
	attempts := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		body, _ := io.ReadAll(r.Body)
		if attempts < 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		if string(body) != "curl-retry-body" {
			http.Error(w, "missing body on retry after curl interceptor", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	retryInterceptor := RetryInterceptor(
		&retry.RetryOptions{MaxRetries: 2, Interval: 10 * time.Millisecond},
		map[int]bool{http.StatusServiceUnavailable: true},
	)

	// CurlInterceptor runs first, then retry interceptor, then executor.
	c := NewBuilder().MaxConcurrentRequests(1).MaxConnsPerHost(1).TimeoutSec(5).
		AddInterceptor(CurlInterceptor).
		AddInterceptor(retryInterceptor).
		Build()
	defer c.Stop()

	req, err := NewRequestBuilder().Method("POST").URL(ts.URL).StringBody("curl-retry-body").Build()
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}

	resp, err := c.DoRequest(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.Code)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
}

func TestBodyMethodIsRetrySafe(t *testing.T) {
	attempts := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		body, _ := io.ReadAll(r.Body)
		if attempts < 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		if string(body) != "body-method-data" {
			http.Error(w, "missing body on retry after Body()", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	retryInterceptor := RetryInterceptor(
		&retry.RetryOptions{MaxRetries: 2, Interval: 10 * time.Millisecond},
		map[int]bool{http.StatusServiceUnavailable: true},
	)

	c := NewBuilder().MaxConcurrentRequests(1).MaxConnsPerHost(1).TimeoutSec(5).AddInterceptor(retryInterceptor).Build()
	defer c.Stop()

	req, err := NewRequestBuilder().Method("POST").URL(ts.URL).Body(io.NopCloser(bytes.NewReader([]byte("body-method-data")))).Build()
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}

	resp, err := c.DoRequest(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.Code)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
}
