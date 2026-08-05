package http

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Request = http.Request

// requestTimeoutKey is a private context key used to communicate a per-request
// timeout from the request builder to the client. A typed key avoids collisions
// with other packages using the string "timeout".
type requestTimeoutKey struct{}

func requestTimeoutContextValue(ctx context.Context) (time.Duration, bool) {
	v := ctx.Value(requestTimeoutKey{})
	if v == nil {
		return 0, false
	}
	d, ok := v.(time.Duration)
	return d, ok
}

// HTTP Header
type headerMaker struct {
	header http.Header
}

type HeaderMaker interface {
	Set(key string, value string) *headerMaker
	Remove(key string) *headerMaker
	Make() http.Header
}

func (m *headerMaker) Set(key string, value string) *headerMaker {
	m.header.Set(key, value)
	return m
}

func (m *headerMaker) Remove(key string) *headerMaker {
	m.header.Del(key)
	return m
}

func (m *headerMaker) Make() http.Header {
	return m.header
}

func NewHeaderMaker() HeaderMaker {
	return &headerMaker{http.Header{}}
}

// HTTP Body
func BuildBodyFrom(body string) io.Reader {
	return strings.NewReader(body)
}

// HTTP Request
type requestBuilder struct {
	ctx        context.Context
	method     string
	url        string
	header     http.Header
	bodyGetter func() (io.ReadCloser, error)
	timeout    time.Duration
}

type RequestBuilder interface {
	Build() (*http.Request, error)
	Context(ctc context.Context) RequestBuilder
	// this will set timeout context when the request is handled(not built)
	// timeout set by this method will not be applied to the net/http http client
	Timeout(timeout time.Duration) RequestBuilder
	Method(method string) RequestBuilder
	URL(url string) RequestBuilder
	Header(header http.Header) RequestBuilder
	Body(body io.ReadCloser) RequestBuilder
	BytesBody(body []byte) RequestBuilder
	StringBody(body string) RequestBuilder
}

func NewRequestBuilder() RequestBuilder {
	return &requestBuilder{
		method:  "GET",
		timeout: time.Duration(0),
		bodyGetter: func() (io.ReadCloser, error) {
			return http.NoBody, nil
		},
	}
}

func (b *requestBuilder) Build() (*http.Request, error) {
	return b.build()
}

func (b *requestBuilder) build() (*http.Request, error) {
	if b.ctx == nil {
		b.ctx = context.Background()
	}
	body, err := b.bodyGetter()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(b.ctx, strings.ToUpper(b.method), b.url, body)
	if err != nil {
		return nil, err
	}
	req.Header = b.header
	req.GetBody = b.bodyGetter
	if b.timeout > 0 {
		req = req.WithContext(context.WithValue(req.Context(), requestTimeoutKey{}, b.timeout))
	}
	return req, nil
}

func (b *requestBuilder) Timeout(timeout time.Duration) RequestBuilder {
	b.timeout = timeout
	return b
}

func (b *requestBuilder) Context(ctx context.Context) RequestBuilder {
	b.ctx = ctx
	return b
}

func (b *requestBuilder) Method(method string) RequestBuilder {
	b.method = strings.ToUpper(method)
	return b
}

func (b *requestBuilder) URL(url string) RequestBuilder {
	b.url = url
	return b
}

func (b *requestBuilder) Header(header http.Header) RequestBuilder {
	b.header = header
	return b
}

func (b *requestBuilder) Body(body io.ReadCloser) RequestBuilder {
	if body == nil {
		b.bodyGetter = func() (io.ReadCloser, error) {
			return http.NoBody, nil
		}
		return b
	}
	// Buffer the provided body so retries can safely rewind it.
	data, err := io.ReadAll(body)
	_ = body.Close()
	if err != nil {
		b.bodyGetter = func() (io.ReadCloser, error) {
			return nil, fmt.Errorf("read request body: %w", err)
		}
		return b
	}
	b.bodyGetter = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(data)), nil
	}
	return b
}

func (b *requestBuilder) BytesBody(body []byte) RequestBuilder {
	b.bodyGetter = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}
	return b
}

func (b *requestBuilder) StringBody(body string) RequestBuilder {
	b.bodyGetter = func() (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader(body)), nil
	}
	return b
}
