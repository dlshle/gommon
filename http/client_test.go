package http

import (
	"testing"
)

func TestClient(t *testing.T) {
	c := NewBuilder().MaxConcurrentRequests(5).MaxConnsPerHost(1).TimeoutSec(60).AddInterceptor(CurlInterceptor).Build()
	r, e := NewRequestBuilder().Method("POST").URL("http://154.44.25.103:8080/echo").BytesBody([]byte("hello")).Build()
	if e != nil {
		t.Errorf("Failed to build request: %v", e)
	}
	resp, err := c.DoRequest(r)
	if err != nil {
		t.Errorf("Failed to request: %v", err)
	}
	if string(resp.Body) != "hello" {
		t.Errorf("Invalid response: %s", string(resp.Body))
	}
}
