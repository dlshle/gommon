package http

import (
	"io"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/dlshle/gommon/utils"
)

// Awaitable Response
type Response struct {
	Code   int
	Header http.Header // usage just like map, can for each kv or ["headerKey"] gives an array of strings
	Body   []byte
	URI    string
}

// response util
func ParseJSONResponseBody[T any](resp *Response) (holder T, err error) {
	return utils.UnmarshalJSONEntity[T](resp.Body)
}

func fromRawResponse(resp *http.Response) (*Response, error) {
	defer resp.Body.Close() // very important for reusing connections in go http client
	uri := resp.Request.URL.Path
	statusCode := resp.StatusCode
	body, err := io.ReadAll(resp.Body)
	return &Response{statusCode, resp.Header, body, uri}, err
}

type awaitableResponse struct {
	response *Response
	err      error
	cond     *sync.Cond
	isClosed atomic.Value
}

type AwaitableResponse interface {
	Wait()
	Get() (*Response, error)
}

func newAwaitableResponse() *awaitableResponse {
	var isClosed atomic.Value
	isClosed.Store(false)
	return &awaitableResponse{nil, nil, sync.NewCond(&sync.Mutex{}), isClosed}
}

func (ar *awaitableResponse) Wait() {
	if !ar.isClosed.Load().(bool) {
		ar.cond.L.Lock()
		ar.cond.Wait()
		ar.cond.L.Unlock()
	}
}

func (ar *awaitableResponse) Get() (*Response, error) {
	ar.Wait()
	return ar.response, ar.err
}

func (ar *awaitableResponse) resolve(resp *Response) {
	if !ar.isClosed.Load().(bool) {
		ar.response = resp
		ar.cond.Broadcast()
		ar.isClosed.Store(true)
	}
}

func (ar *awaitableResponse) reject(err error) {
	if !ar.isClosed.Load().(bool) {
		ar.err = err
		ar.cond.Broadcast()
		ar.isClosed.Store(true)
	}
}
