package http

import (
	"testing"
)

func TestClient(t *testing.T) {
	c := NewBuilder().MaxConcurrentRequests(5).MaxConnsPerHost(1).TimeoutSec(60).AddInterceptor(CurlInterceptor).Build()
	r := NewRequestBuilder().Method("GET").URL("http://106.14.70.176:8088/superset/sqllab/").Build()
	resp, err := c.DoRequest(r)
	if err != nil {
		t.Errorf("Failed to request: %v", err)
	}
	t.Logf("resp: %+v", resp)
}
