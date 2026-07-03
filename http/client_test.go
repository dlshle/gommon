package http

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
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
