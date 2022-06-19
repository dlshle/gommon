package http

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	urlpkg "net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dlshle/gommon/logger"
)

// New version, need to deprecate http_client

// Globals

// TODO, need a customized request capable of retry
// TODO, Go http client already has connection reusability built-in, so no need to use the producer/consumer pattern here unless request limitation is required
type Request = http.Request

// logger
var globalLogger = logger.New(os.Stdout, "[NetworkClient]", true)

// request status error message_dispatcher
var requestStatusErrorStringMap map[int32]string
var requestStatusErrorCodeMap map[int32]int

// trackableRequest Status
const (
	RequestStatusIdle       = 0
	RequestStatusWaiting    = 1
	RequestStatusInProgress = 2
	RequestStatusCancelled  = 9
	RequestStatusDone       = 10
)

// Rand utils
var randomGenerator = rand.New(rand.NewSource(time.Now().UnixNano()))

// init
func initRequestStatusErrorMaps() {
	requestStatusErrorCodeMap = make(map[int32]int)
	requestStatusErrorStringMap = make(map[int32]string)
	requestStatusErrorStringMap[RequestStatusInProgress] = "Handle is in progress"
	requestStatusErrorCodeMap[RequestStatusInProgress] = ErrRequestInProgress
	requestStatusErrorStringMap[RequestStatusCancelled] = "Handle is cancelled"
	requestStatusErrorCodeMap[RequestStatusCancelled] = ErrRequestCancelled
	requestStatusErrorStringMap[RequestStatusDone] = "Handle is finished"
	requestStatusErrorCodeMap[RequestStatusDone] = ErrRequestFinished
}

func init() {
	initRequestStatusErrorMaps()
	initPoolStatusStringMap()
}

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
}

type RequestBuilder interface {
	Build() *http.Request
	Method(method string) RequestBuilder
	URL(url string) RequestBuilder
	Header(header http.Header) RequestBuilder
	Body(body io.ReadCloser) RequestBuilder
	StringBody(body string) RequestBuilder
}

func NewRequestBuilder() RequestBuilder {
	return &requestBuilder{&http.Request{}}
}

func (b *requestBuilder) Build() *http.Request {
	if b.request.Method == "" {
		b.request.Method = "GET"
	}
	return b.request
}

func (b *requestBuilder) Method(method string) RequestBuilder {
	b.request.Method = method
	return b
}

func (b *requestBuilder) URL(url string) RequestBuilder {
	u, err := urlpkg.Parse(url)
	if err != nil {
		globalLogger.Printf("Unable to parse url(%s), fallback to original url(%s).\n", url, b.request.URL.String())
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
		rc = ioutil.NopCloser(bodyReader)
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

func fromRawResponse(resp *http.Response) (*Response, error) {
	defer resp.Body.Close() // very important for reusing connections in go http client
	uri := resp.Request.URL.Path
	statusCode := resp.StatusCode
	body, err := ioutil.ReadAll(resp.Body)
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
	id       string
	status   int32
	request  *http.Request
	response *awaitableResponse
}

type TrackableRequest interface {
	Id() string
	Status() int32
	Update(request *http.Request) error
	Cancel() error
	Response() *Response
	getRequest() *http.Request
	setStatus(status int32)
}

func NewTrackableRequest(request *http.Request) TrackableRequest {
	id := strconv.FormatInt(randomGenerator.Int63n(time.Now().Unix()), 16)
	return &trackableRequest{id, RequestStatusIdle, request, newAwaitableResponse()}
}

func (tr *trackableRequest) Id() string {
	return tr.id
}

func (tr *trackableRequest) Status() int32 {
	return atomic.LoadInt32(&tr.status)
}

func (tr *trackableRequest) setStatus(status int32) {
	atomic.StoreInt32(&tr.status, status)
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
	if status <= RequestStatusWaiting {
		tr.status = RequestStatusCancelled
		tr.response.resolve(cancelledResponse())
		return nil
	}
	return NewClientError("Unable to update request due to "+requestStatusErrorStringMap[status], requestStatusErrorCodeMap[status])
}

func (tr *trackableRequest) Response() *Response {
	return tr.response.Get()
}

// Pool
// constants
const (
	PoolStatusIdle        = 0
	PoolStatusStarting    = 1
	PoolStatusRunning     = 2
	PoolStatusTerminating = 3
	PoolStatusStopped     = 4
)

var poolStatusStringMap map[int]string

func initPoolStatusStringMap() {
	poolStatusStringMap = make(map[int]string)
	poolStatusStringMap[PoolStatusIdle] = "Idle"
	poolStatusStringMap[PoolStatusStarting] = "Starting"
	poolStatusStringMap[PoolStatusRunning] = "Running"
	poolStatusStringMap[PoolStatusTerminating] = "Terminating"
	poolStatusStringMap[PoolStatusStopped] = "Stopped"
}

type httpClient struct {
	ctx         context.Context
	cancelFunc  func()
	id          string
	queue       chan TrackableRequest
	logger      *logger.SimpleLogger
	status      int
	rwMutex     *sync.RWMutex
	workerSize  int
	numWorkers  int32
	baseClient  *http.Client
	stopWg      sync.WaitGroup
	numExceeded int32
}

type HTTPClient interface {
	Id() string
	request(request *http.Request) TrackableRequest
	DoRequest(request *http.Request) *Response
	Request(request *http.Request) *Response
	RequestAsync(request *http.Request) TrackableRequest
	batchRequest(requests []*http.Request) []TrackableRequest
	BatchRequest(requests []*http.Request) []*Response
	Verbose(use bool)
	BatchRequestAsync(requests []*http.Request) []TrackableRequest
	Status() int
	setStatus(status int)
	Stop()
}

func numWithinRange(value, min, max int) int {
	if value < min {
		value = min
	} else if value > max {
		value = max
	}
	return value
}

func newHTTPClient(timeout int) *http.Client {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConns = 100
	t.MaxConnsPerHost = 100
	t.MaxIdleConnsPerHost = 100
	return &http.Client{
		Timeout:   time.Second * time.Duration(timeout),
		Transport: t,
	}
}

func New(id string, maxConcurrentRequests, maxQueueSize, timeoutInSec int) HTTPClient {
	ctx, cancelFunc := context.WithCancel(context.Background())
	var stopWg sync.WaitGroup
	maxConcurrentRequests = numWithinRange(maxConcurrentRequests, 1, 2048)
	maxQueueSize = numWithinRange(maxQueueSize, 1, 4096)
	rawClients := make([]*http.Client, maxConcurrentRequests)
	for i := 0; i < maxConcurrentRequests; i++ {
		rawClients[i] = newHTTPClient(timeoutInSec)
	}
	client := &httpClient{
		ctx,
		cancelFunc,
		id,
		make(chan TrackableRequest, maxQueueSize),
		logger.New(os.Stdout, fmt.Sprintf("HttpClient[%s]", id), false),
		PoolStatusIdle,
		new(sync.RWMutex),
		maxConcurrentRequests,
		0,
		newHTTPClient(timeoutInSec),
		stopWg,
		0,
	}
	client.status = PoolStatusRunning
	return client
}

func (c *httpClient) incrementAndGetWorkerCount() int32 {
	return atomic.AddInt32(&c.numWorkers, 1)
}

func (c *httpClient) decrementWorkerCount() {
	atomic.AddInt32(&c.numWorkers, -1)
}

func (c *httpClient) pendingRequests() int {
	return len(c.queue)
}

func (c *httpClient) maxQueueSize() int {
	return cap(c.queue)
}

func (c *httpClient) workerCount() int {
	return int(atomic.LoadInt32(&c.numWorkers))
}

func (c *httpClient) isQueueSizeExceeded() bool {
	return c.pendingRequests() >= c.maxQueueSize()
}

func (c *httpClient) tryToStartNewWorker() {
	if c.pendingRequests() > 0 && c.workerCount() < c.workerSize {
		c.stopWg.Add(1)
		go c.workerFunc(int(c.incrementAndGetWorkerCount()))
	}
}

func (c *httpClient) workerFunc(id int) {
	numRequests := 0
	numSuccess := 0
	numFailed := 0
	taggedLogger := c.logger.WithPrefix(fmt.Sprintf("[Worker-%d]", id))
	taggedLogger.Printf("worker has started.")
	shouldContinue := true
	for shouldContinue {
		select {
		case req, isOpen := <-c.queue:
			if !isOpen {
				shouldContinue = false
				break
			}
			request := req.(*trackableRequest)
			if request.Status() != RequestStatusWaiting {
				taggedLogger.Printf("skip request(%s) due to invalid status(%d).", request.Id(), request.Status())
			} else {
				numRequests++
				success := c.executeRequest(taggedLogger, request)
				if success {
					numSuccess++
				} else {
					numFailed++
				}
			}
			if c.pendingRequests() == 0 {
				shouldContinue = false
			}
		case <-c.ctx.Done():
			shouldContinue = false
		}
	}
	taggedLogger.Printf("worker lifecycle is finished, numRequests: %d, numSuccessRequests: %d, numFailedRequests: %d", numRequests, numSuccess, numFailed)
	c.decrementWorkerCount()
	c.stopWg.Done()
}

func (c *httpClient) executeRequest(taggedLogger *logger.SimpleLogger, request *trackableRequest) (success bool) {
	request.setStatus(RequestStatusInProgress)
	taggedLogger.Printf("worker has acquired request(%s, %d) with rawRequest %+v.", request.id, request.Status(), request.getRequest())
	rawResponse, err := c.baseClient.Do(request.getRequest())
	if err != nil || rawResponse == nil {
		taggedLogger.Printf("request failed due to %s, will resolve it with invalid response(-1).", err.Error())
		request.response.resolve(invalidResponse("failed: "+err.Error(), -1))
		success = false
	} else {
		response, err := fromRawResponse(rawResponse)
		if err != nil {
			taggedLogger.Printf("unable to parse response body of %+v.\n", rawResponse)
			success = false
		} else {
			success = true
		}
		request.response.resolve(response)
		taggedLogger.Printf("request(%s) has been resolved. Response: %+v.\n", request.id, response)
	}
	return
}

func (c *httpClient) Stop() {
	c.cancelFunc()
	c.setStatus(PoolStatusTerminating)
	close(c.queue)
	c.stopWg.Wait()
}

func (c *httpClient) setStatus(status int) {
	c.rwMutex.Lock()
	defer c.rwMutex.Unlock()
	oldStatus := c.status
	c.status = status
	c.logger.Printf("Switched pool status from %s to %s\n", poolStatusStringMap[oldStatus], poolStatusStringMap[status])
}

func (c *httpClient) Id() string {
	return c.id
}

func (c *httpClient) Status() int {
	c.rwMutex.RLock()
	defer c.rwMutex.RUnlock()
	return c.status
}

func (c *httpClient) request(request *http.Request) TrackableRequest {
	loggerTag := "[Handle]"
	c.logger.Printf("%s New request received: %+v\nCurrent queue size: %d\n", loggerTag, request, len(c.queue))
	tRequest := NewTrackableRequest(request)
	tRequest.setStatus(RequestStatusWaiting)
	if c.Status() != PoolStatusRunning {
		tRequest.(*trackableRequest).response.resolve(invalidResponse("client is closed", -1))
		atomic.AddInt32(&c.numExceeded, 1)
		return tRequest
	}
	if c.isQueueSizeExceeded() {
		c.executeRequest(c.logger.WithPrefix("[direct]"), tRequest.(*trackableRequest))
		return tRequest
	}
	c.queue <- tRequest
	c.tryToStartNewWorker()
	return tRequest
}

func (c *httpClient) DoRequest(request *http.Request) *Response {
	tRequest := NewTrackableRequest(request)
	tRequest.setStatus(RequestStatusWaiting)
	c.executeRequest(c.logger.WithPrefix("[direct]"), tRequest.(*trackableRequest))
	return tRequest.Response()
}

func (c *httpClient) Request(request *http.Request) *Response {
	tr := c.request(request)
	return tr.Response()
}

func (c *httpClient) RequestAsync(request *http.Request) TrackableRequest {
	return c.request(request)
}

func (c *httpClient) batchRequest(requests []*http.Request) []TrackableRequest {
	res := make([]TrackableRequest, len(requests))
	for i, req := range requests {
		res[i] = c.request(req)
	}
	return res
}

func (c *httpClient) BatchRequest(requests []*http.Request) []*Response {
	responses := make([]*Response, len(requests))
	trs := c.batchRequest(requests)
	for i, tr := range trs {
		if tr == nil {
			responses[i] = nil
		} else {
			responses[i] = tr.Response()
		}
	}
	return responses
}

func (c *httpClient) BatchRequestAsync(requests []*http.Request) []TrackableRequest {
	return c.batchRequest(requests)
}

func (c *httpClient) Verbose(use bool) {
	c.logger.Verbose(use)
}
