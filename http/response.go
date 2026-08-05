package http

import (
	"fmt"
	"io"
	"net/http"
	"sync"

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

func fromRawResponse(resp *http.Response, maxBodySize int64) (*Response, error) {
	defer resp.Body.Close() // very important for reusing connections in go http client
	uri := resp.Request.URL.Path
	statusCode := resp.StatusCode

	var body []byte
	var err error
	if maxBodySize > 0 {
		body, err = io.ReadAll(io.LimitReader(resp.Body, maxBodySize+1))
		if err == nil && int64(len(body)) > maxBodySize {
			err = fmt.Errorf("response body exceeds maximum allowed size of %d bytes", maxBodySize)
		}
	} else {
		body, err = io.ReadAll(resp.Body)
	}
	return &Response{statusCode, resp.Header, body, uri}, err
}

type awaitableResponse struct {
	response *Response
	err      error
	done     chan struct{}
	once     sync.Once
}

type AwaitableResponse interface {
	Wait()
	Get() (*Response, error)
}

func newAwaitableResponse() *awaitableResponse {
	return &awaitableResponse{done: make(chan struct{})}
}

func (ar *awaitableResponse) Wait() {
	<-ar.done
}

func (ar *awaitableResponse) Get() (*Response, error) {
	<-ar.done
	return ar.response, ar.err
}

func (ar *awaitableResponse) resolve(resp *Response) {
	ar.once.Do(func() {
		ar.response = resp
		close(ar.done)
	})
}

func (ar *awaitableResponse) reject(err error) {
	ar.once.Do(func() {
		ar.err = err
		close(ar.done)
	})
}
