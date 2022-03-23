package http

import (
	"net/http"
	"testing"
)

func TestClient(t *testing.T) {
	client := NewBuilder().
		Id("test").
		MaxConcurrentRequests(5).
		MaxQueueSize(128).
		TimeoutSec(30).
		MaxConnsPerHost(5).
		Build()
	requestToBaidu := NewRequestBuilder().URL("https://www.baidu.com").Method(http.MethodGet).Build()
	resp := client.RequestAsync(requestToBaidu)
	resp1 := client.RequestAsync(requestToBaidu)
	if resp.Response().Body != resp1.Response().Body {
		t.FailNow()
	}
	requestToBing := NewRequestBuilder().URL("https://www.bing.com").Method(http.MethodGet).Build()
	var respArr []TrackableRequest
	for i := 0; i < 100; i++ {
		respArr = append(respArr, client.RequestAsync(requestToBing))
	}
	t.Log("===================================================================================")
	t.Logf("resp size: %d", len(respArr))
	for _, at := range respArr {
		t.Log(at.Response().Code)
		if !at.Response().Success {
			t.FailNow()
			t.Logf("request %s failed with %s", at.Id(), at.Response().Body)
		}
	}
	t.Logf("last resp body: %s", respArr[len(respArr)-1].Response().Body)
	t.Logf("num exceeded: %d", client.(*httpClient).numExceeded)
	client.Stop()
}
