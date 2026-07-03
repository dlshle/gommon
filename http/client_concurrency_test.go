package http

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestClientSuccessfulRequest(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))
	defer ts.Close()

	c := NewBuilder().MaxConcurrentRequests(5).MaxConnsPerHost(10).TimeoutSec(30).Build()
	defer c.Stop()

	req, err := NewRequestBuilder().Method("POST").URL(ts.URL + "/echo").Header(
		NewHeaderMaker().Set("hello", "world").Make(),
	).BytesBody([]byte("hello")).Build()
	if err != nil {
		t.Fatalf("Failed to build request: %v", err)
	}

	resp, err := c.DoRequest(req)
	if err != nil {
		t.Fatalf("Failed to request: %v", err)
	}
	if resp == nil || resp.Code != http.StatusOK || string(resp.Body) != "hello" {
		t.Errorf("Invalid response: %+v", resp)
	}
}

func TestClientStopIsIdempotent(t *testing.T) {
	c := NewHTTPClient(2, 10, 30)
	c.Stop()
	c.Stop() // should not panic
	if c.Status() != PoolStatusStopped {
		t.Errorf("expected status Stopped, got %d", c.Status())
	}
}

func TestClientRequestsAfterStopAreRejected(t *testing.T) {
	c := NewHTTPClient(2, 10, 30)
	c.Stop()

	req, _ := http.NewRequestWithContext(context.Background(), "GET", "http://127.0.0.1:1/", nil)
	resp, err := c.Request(req)
	if err == nil {
		t.Fatal("expected error for request after stop")
	}
	if resp != nil {
		t.Errorf("expected nil response, got %+v", resp)
	}
}

func TestClientQueueFullIsRejected(t *testing.T) {
	blocked := make(chan struct{})
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-blocked
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	c := NewHTTPClient(1, 1, 30)
	defer c.Stop()

	// Occupies the only worker.
	req1, _ := http.NewRequestWithContext(context.Background(), "GET", ts.URL+"/slow", nil)
	ar1 := c.RequestAsync(req1)

	// Wait until the worker has picked up the first request.
	time.Sleep(100 * time.Millisecond)

	// Fills the only queue slot.
	req2, _ := http.NewRequestWithContext(context.Background(), "GET", ts.URL+"/slow", nil)
	ar2 := c.RequestAsync(req2)

	// Must be rejected because both the worker and the queue are busy.
	req3, _ := http.NewRequestWithContext(context.Background(), "GET", ts.URL+"/slow", nil)
	resp3, err3 := c.Request(req3)
	if err3 == nil {
		t.Fatal("expected queue-full error")
	}
	if resp3 != nil {
		t.Errorf("expected nil response, got %+v", resp3)
	}

	// Ensure the first two requests eventually succeed.
	close(blocked)
	if _, err := ar1.Get(); err != nil {
		t.Fatalf("request 1 failed: %v", err)
	}
	if _, err := ar2.Get(); err != nil {
		t.Fatalf("request 2 failed: %v", err)
	}
}

func TestClientConcurrentStopAndSubmit(t *testing.T) {
	for i := 0; i < 10; i++ {
		c := NewHTTPClient(2, 10, 30)
		var wg sync.WaitGroup
		var sent int32
		for j := 0; j < 20; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				req, _ := http.NewRequestWithContext(context.Background(), "GET", "http://127.0.0.1:1/", nil)
				c.RequestAsync(req)
				atomic.AddInt32(&sent, 1)
			}()
		}
		go c.Stop()
		wg.Wait()
		_ = atomic.LoadInt32(&sent)
	}
}

func TestAwaitableResponseNoLostWakeup(t *testing.T) {
	// Regression test for the lost-wakeup bug: resolve called concurrently with Wait.
	var wg sync.WaitGroup
	for i := 0; i < 500; i++ {
		ar := newAwaitableResponse()
		wg.Add(1)
		go func() {
			defer wg.Done()
			ar.Wait()
		}()
		ar.resolve(&Response{Code: 200})
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("lost wakeup: Wait goroutine blocked forever")
	}
}

func TestClientDrainsQueueOnStop(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "ok")
	}))
	defer ts.Close()

	c := NewHTTPClient(2, 100, 30)
	responses := make([]AwaitableResponse, 10)
	for i := range responses {
		req, _ := http.NewRequestWithContext(context.Background(), "GET", ts.URL+"/ok", nil)
		responses[i] = c.RequestAsync(req)
	}

	c.Stop()

	for i, ar := range responses {
		resp, err := ar.Get()
		if err != nil {
			t.Fatalf("request %d failed after stop: %v", i, err)
		}
		if resp == nil || resp.Code != http.StatusOK {
			t.Errorf("request %d invalid response: %+v", i, resp)
		}
	}
}
