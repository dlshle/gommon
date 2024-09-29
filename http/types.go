package http

import (
	"context"
	"io"
	"net/http"
	urlpkg "net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dlshle/gommon/utils"
)

// Errors

// error codes
const (
	ErrInvalidRequest       = 0
	ErrInvalidRequestUrl    = 1
	ErrInvalidRequestMethod = 2
	ErrRequestInProgress    = 3
	ErrRequestCancelled     = 4
	ErrRequestFinished      = 5
)

type ClientError struct {
	msg  string
	code int
}

func (e *ClientError) Error() string {
	return e.msg
}

func NewClientError(msg string, code int) *ClientError {
	return &ClientError{msg, code}
}

func DefaultClientError(msg string) *ClientError {
	return NewClientError(msg, 0)
}

type Request = http.Request

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
	request *http.Request
	timeout time.Duration
}

type RequestBuilder interface {
	Build() *http.Request
	Context(ctc context.Context) RequestBuilder
	// this will set timeout context when the request is handled(not built)
	// timeout set by this method will not be applied to the net/http http client
	Timeout(timeout time.Duration) RequestBuilder
	Method(method string) RequestBuilder
	URL(url string) RequestBuilder
	Header(header http.Header) RequestBuilder
	Body(body io.ReadCloser) RequestBuilder
	StringBody(body string) RequestBuilder
}

func NewRequestBuilder() RequestBuilder {
	req := requestPool.Get().(*http.Request)
	return &requestBuilder{
		request: req,
		timeout: time.Duration(0),
	}
}

func (b *requestBuilder) Build() *http.Request {
	if b.request.Method == "" {
		b.request.Method = "GET"
	}
	if b.timeout > 0 {
		return b.request.WithContext(context.WithValue(b.request.Context(), "timeout", b.timeout))
	}
	return b.request
}

func (b *requestBuilder) Timeout(timeout time.Duration) RequestBuilder {
	b.timeout = timeout
	return b
}

func (b *requestBuilder) Context(ctx context.Context) RequestBuilder {
	b.request = b.request.WithContext(ctx)
	return b
}

func (b *requestBuilder) Method(method string) RequestBuilder {
	b.request.Method = strings.ToUpper(method)
	return b
}

func (b *requestBuilder) URL(url string) RequestBuilder {
	u, err := urlpkg.Parse(url)
	if err != nil {
		// globalLogger.Printf("Unable to parse url(%s), fallback to original url(%s).\n", url, b.request.URL.String())
		return b
	}
	b.request.URL = u
	return b
}

func (b *requestBuilder) Header(header http.Header) RequestBuilder {
	b.request.Header = header
	return b
}

func (b *requestBuilder) Body(body io.ReadCloser) RequestBuilder {
	b.request.Body = body
	return b
}

func (b *requestBuilder) StringBody(body string) RequestBuilder {
	bodyReader := BuildBodyFrom(body)
	rc, ok := bodyReader.(io.ReadCloser)
	if !ok && bodyReader != nil {
		rc = io.NopCloser(bodyReader)
	}
	b.request.Body = rc
	return b
}

// Awaitable Response
type Response struct {
	Success bool
	Code    int
	Header  http.Header // usage just like map, can for each kv or ["headerKey"] gives an array of strings
	Body    string
	URI     string
}

// response util
func ParseJSONResponseBody[T any](resp *Response) (holder T, err error) {
	return utils.UnmarshalJSONEntity[T]([]byte(resp.Body))
}

func fromRawResponse(resp *http.Response) (*Response, error) {
	defer resp.Body.Close() // very important for reusing connections in go http client
	uri := resp.Request.URL.Path
	statusCode := resp.StatusCode
	body, err := io.ReadAll(resp.Body)
	var bodyString string
	if err != nil {
		bodyString = err.Error()
	} else {
		bodyString = string(body[:])
	}
	return &Response{statusCode >= 200 && statusCode <= 300, statusCode, resp.Header, bodyString, uri}, err
}

// Invalid response builder
func invalidResponse(status string, statusCode int) *Response {
	return &Response{false, statusCode, nil, status, ""}
}

type awaitableResponse struct {
	response *Response
	cond     *sync.Cond
	isClosed atomic.Value
}

type AwaitableResponse interface {
	Wait()
	Get() *http.Response
	Cancel() bool
	resolve(resp *http.Response)
}

func newAwaitableResponse() *awaitableResponse {
	var isClosed atomic.Value
	isClosed.Store(false)
	return &awaitableResponse{nil, sync.NewCond(&sync.Mutex{}), isClosed}
}

func (ar *awaitableResponse) Wait() {
	if !ar.isClosed.Load().(bool) {
		ar.cond.L.Lock()
		ar.cond.Wait()
		ar.cond.L.Unlock()
	}
}

func (ar *awaitableResponse) Get() *Response {
	ar.Wait()
	return ar.response
}

func (ar *awaitableResponse) resolve(resp *Response) {
	if !ar.isClosed.Load().(bool) {
		ar.response = resp
		ar.cond.Broadcast()
		ar.isClosed.Store(true)
	}
}

// Trackable Request
// canceled response
func cancelledResponse() *Response {
	return invalidResponse("Cancelled", -4)
}

type trackableRequest struct {
	id         string
	cancelFunc func()
	status     int32
	request    *http.Request
	response   *awaitableResponse
}

type TrackableRequest interface {
	ID() string
	Status() int32
	Update(request *http.Request) error
	Cancel() error
	Response() *Response
	getRequest() *http.Request
	setStatus(status int32)
}

func newTrackableRequest(request *http.Request) *trackableRequest {
	var (
		ctx        context.Context
		cancelFunc func()
	)
	if timeoutVal := request.Context().Value("timeout"); timeoutVal != nil {
		ctx, cancelFunc = context.WithTimeout(request.Context(), timeoutVal.(time.Duration))
	} else {
		ctx, cancelFunc = context.WithCancel(request.Context())
	}
	request = request.WithContext(ctx)
	id := strconv.FormatInt(randomGenerator.Int63n(time.Now().Unix()), 16)
	return &trackableRequest{id, cancelFunc, RequestStatusIdle, request, newAwaitableResponse()}
}

func (tr *trackableRequest) ID() string {
	return tr.id
}

func (tr *trackableRequest) Status() int32 {
	return atomic.LoadInt32(&tr.status)
}

func (tr *trackableRequest) setStatus(status int32) {
	atomic.StoreInt32(&tr.status, status)
}

func (tr *trackableRequest) complete() {
	// invoke cancel func to relase timeout context timer
	tr.cancelFunc()
	tr.setStatus(RequestStatusDone)
	requestPool.Put(tr.request)
}

func (tr *trackableRequest) getRequest() *http.Request {
	return tr.request
}

func (tr *trackableRequest) Update(request *http.Request) error {
	status := tr.Status()
	if status <= RequestStatusWaiting {
		tr.request = request
		return nil
	}
	return NewClientError("Unable to update request due to "+requestStatusErrorStringMap[status], requestStatusErrorCodeMap[status])
}

func (tr *trackableRequest) Cancel() error {
	status := tr.Status()
	if status < RequestStatusDone {
		tr.cancelFunc()
		tr.status = RequestStatusCancelled
		tr.response.resolve(cancelledResponse())
		return nil
	}
	return NewClientError("Unable to update request due to "+requestStatusErrorStringMap[status], requestStatusErrorCodeMap[status])
}

func (tr *trackableRequest) Response() *Response {
	return tr.response.Get()
}
