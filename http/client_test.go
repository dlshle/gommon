package http

import (
	"net/http"
	"testing"
	"time"
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
	/*
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
				t.Logf("request %s failed with %s", at.ID(), at.Response().Body)
			}
		}
		t.Logf("last resp body: %s", respArr[len(respArr)-1].Response().Body)
		t.Logf("num exceeded: %d", client.(*httpClient).numExceeded)
	*/

	t.Logf("test timeout")
	r := client.DoRequest(NewRequestBuilder().Timeout(time.Millisecond * 100).URL("https://www.baidu.com").Method(http.MethodGet).Build())
	t.Logf("request failed %v", r)
	if r.Code != -1 {
		t.Errorf("not timeout!")
	}
	client.Stop()
}
