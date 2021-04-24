package http

import (
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	urlpkg "net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dlshle/gommon/logger"
)

// Globals

// TODO, need a customized request capable of retry
type Request = http.Request

// logger
var globalLogger = logger.New(os.Stdout, "[NetworkClient]", true)

// request status error messages
var requestStatusErrorStringMap map[int]string
var requestStatusErrorCodeMap map[int]int

// TrackableRequest Status
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
	requestStatusErrorCodeMap = make(map[int]int)
	requestStatusErrorStringMap = make(map[int]string)
	requestStatusErrorStringMap[RequestStatusInProgress] = "Request is in progress"
	requestStatusErrorCodeMap[RequestStatusInProgress] = ErrRequestInProgress
	requestStatusErrorStringMap[RequestStatusCancelled] = "Request is cancelled"
	requestStatusErrorCodeMap[RequestStatusCancelled] = ErrRequestCancelled
	requestStatusErrorStringMap[RequestStatusDone] = "Request is finished"
	requestStatusErrorCodeMap[RequestStatusDone] = ErrRequestFinished
}

func init() {
	initRequestStatusErrorMaps()
	initPoolStatusStringMap()
	initGlobalPool()
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
type HeaderMaker struct {
	header http.Header
}

type IHeaderMaker interface {
	Set(key string, value string) *HeaderMaker
	Remove(key string) *HeaderMaker
	Make() *http.Header
}

func (m *HeaderMaker) Set(key string, value string) *HeaderMaker {
	m.header.Set(key, value)
	return m
}

func (m *HeaderMaker) Remove(key string) *HeaderMaker {
	m.header.Del(key)
	return m
}

func (m *HeaderMaker) Make() http.Header {
	return m.header
}

func NewHeaderMaker() *HeaderMaker {
	return &HeaderMaker{http.Header{}}
}

// HTTP Body
func BuildBodyFrom(body string) io.Reader {
	return strings.NewReader(body)
}

// HTTP Request
type RequestBuilder struct {
	request *http.Request
}

type IRequestBuilder interface {
	Build() *http.Request
	Method(method string) *RequestBuilder
	URL(url string) *RequestBuilder
	Header(header http.Header) *RequestBuilder
	Body(body io.ReadCloser) *RequestBuilder
	StringBody(body string) *RequestBuilder
}

func NewRequestBuilder() IRequestBuilder {
	return &RequestBuilder{&http.Request{}}
}

func (b *RequestBuilder) Build() *http.Request {
	if b.request.Method == "" {
		b.request.Method = "GET"
	}
	return b.request
}

func (b *RequestBuilder) Method(method string) *RequestBuilder {
	b.request.Method = method
	return b
}

func (b *RequestBuilder) URL(url string) *RequestBuilder {
	u, err := urlpkg.Parse(url)
	if err != nil {
		globalLogger.Printf("Unable to parse url(%s), fallback to original url(%s).\n", url, b.request.URL.String())
		return b
	}
	b.request.URL = u
	return b
}

func (b *RequestBuilder) Header(header http.Header) *RequestBuilder {
	b.request.Header = header
	return b
}

func (b *RequestBuilder) Body(body io.ReadCloser) *RequestBuilder {
	b.request.Body = body
	return b
}

func (b *RequestBuilder) StringBody(body string) *RequestBuilder {
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
	success bool
	code    int
	header  http.Header // usage just like map, can for each kv or ["headerKey"] gives an array of strings
	body    string
}

func fromRawResponse(resp *http.Response) (*Response, error) {
	defer resp.Body.Close()
	statusCode := resp.StatusCode
	body, err := ioutil.ReadAll(resp.Body)
	var bodyString string
	if err != nil {
		bodyString = err.Error()
	} else {
		bodyString = string(body[:])
	}
	return &Response{statusCode >= 200 && statusCode <= 300, statusCode, resp.Header, bodyString}, err
}

// Invalid response builder
func invalidResponse(status string, statusCode int) *Response {
	return &Response{false, statusCode, nil, status}
}

type AwaitableResponse struct {
	response *Response
	cond     *sync.Cond
	isClosed atomic.Value
}

type IAwaitableResponse interface {
	Wait()
	Get() *http.Response
	Cancel() bool
	resolve(resp *http.Response)
}

func newAwaitableResponse() *AwaitableResponse {
	var isClosed atomic.Value
	isClosed.Store(false)
	return &AwaitableResponse{nil, sync.NewCond(&sync.Mutex{}), isClosed}
}

func (ar *AwaitableResponse) Wait() {
	if !ar.isClosed.Load().(bool) {
		ar.cond.L.Lock()
		ar.cond.Wait()
		ar.cond.L.Unlock()
	}
}

func (ar *AwaitableResponse) Get() *Response {
	ar.Wait()
	return ar.response
}

func (ar *AwaitableResponse) resolve(resp *Response) {
	if !ar.isClosed.Load().(bool) {
		ar.response = resp
		ar.isClosed.Store(true)
		ar.cond.Broadcast()
	}
}

// Trackable Request
// canceled response
func cancelledResponse() *Response {
	return invalidResponse("Cancelled", -4)
}

type TrackableRequest struct {
	id       string
	status   int
	request  *http.Request
	response *AwaitableResponse
	rwMutex  *sync.RWMutex
}

type ITrackableRequest interface {
	Id() string
	Status() int
	Update(request *http.Request) error
	Cancel() error
	Response() *Response
	getRequest() *http.Request
	setStatus(status int)
}

func NewTrackableRequest(request *http.Request) *TrackableRequest {
	id := strconv.FormatInt(randomGenerator.Int63n(time.Now().Unix()), 16)
	return &TrackableRequest{id, RequestStatusIdle, request, newAwaitableResponse(), new(sync.RWMutex)}
}

func (tr *TrackableRequest) Id() string {
	return tr.id
}

func (tr *TrackableRequest) Status() int {
	tr.rwMutex.RLock()
	defer tr.rwMutex.RUnlock()
	return tr.status
}

func (tr *TrackableRequest) setStatus(status int) {
	tr.rwMutex.Lock()
	defer tr.rwMutex.Unlock()
	tr.status = status
}

func (tr *TrackableRequest) getRequest() *http.Request {
	tr.rwMutex.RLock()
	defer tr.rwMutex.RUnlock()
	return tr.request
}

func (tr *TrackableRequest) Update(request *http.Request) error {
	status := tr.Status()
	if status <= RequestStatusWaiting {
		tr.rwMutex.Lock()
		tr.request = request
		tr.rwMutex.Unlock()
		return nil
	}
	return NewClientError("Unable to update request due to "+requestStatusErrorStringMap[status], requestStatusErrorCodeMap[status])
}

func (tr *TrackableRequest) Cancel() error {
	status := tr.Status()
	if status <= RequestStatusWaiting {
		tr.rwMutex.Lock()
		tr.status = RequestStatusCancelled
		tr.response.resolve(cancelledResponse())
		tr.rwMutex.Unlock()
		return nil
	}
	return NewClientError("Unable to update request due to "+requestStatusErrorStringMap[status], requestStatusErrorCodeMap[status])
}

func (tr *TrackableRequest) Response() *Response {
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

type ClientPool struct {
	id      string
	clients []*http.Client
	queue   chan *TrackableRequest
	logger  *logger.SimpleLogger
	status  int
	rwMutex *sync.RWMutex
}

type IClientPool interface {
	Id() string
	request(request *http.Request) *TrackableRequest
	Request(request *http.Request) *Response
	RequestAsync(request *http.Request) *TrackableRequest
	batchRequest(requests []*http.Request) []*TrackableRequest
	BatchRequest(requests []*http.Request) []*Response
	Verbose(use bool)
	// or channel?
	BatchRequestAsync(requests []*http.Request) []*TrackableRequest
	Status() int
	setStatus(status int)
	start()
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
	return &http.Client{Timeout: time.Second * time.Duration(timeout)}
}

func NewPool(id string, numClients, maxQueueSize, timeoutInSec int) *ClientPool {
	numClients = numWithinRange(numClients, 1, 2048)
	maxQueueSize = numWithinRange(maxQueueSize, 1, 4096)
	rawClients := make([]*http.Client, numClients)
	for i := 0; i < numClients; i++ {
		rawClients[i] = newHTTPClient(timeoutInSec)
	}
	pool := &ClientPool{
		id,
		rawClients,
		make(chan *TrackableRequest, maxQueueSize),
		logger.New(os.Stdout, fmt.Sprintf("HttpClient[%s]", id), false),
		PoolStatusIdle,
		new(sync.RWMutex),
	}
	pool.start()
	return pool
}

func (c *ClientPool) start() {
	if c.Status() != PoolStatusIdle {
		return
	}
	c.logger.Printf("Starting the client...\n")
	go func() {
		var wg sync.WaitGroup
		c.setStatus(PoolStatusStarting)
		for i, clientItr := range c.clients {
			wg.Add(1)
			go func(id int, client *http.Client) {
				numRequests := 0
				numSuccess := 0
				numFailed := 0
				loggerTag := fmt.Sprintf("[Client-%d]", id)
				c.logger.Printf("%s client has started.\n", loggerTag)
				for c.Status() == PoolStatusRunning {
					// actual worker logic
					request := <-c.queue
					if request != nil {
						if request.Status() != RequestStatusWaiting {
							c.logger.Printf("%s skip request(%s) due to invalid status(%d).\n", loggerTag, request.Id(), request.Status())
						}
						numRequests++
						request.setStatus(RequestStatusInProgress)
						c.logger.Printf("%s client has acquired request(%s, %d) with rawRequest %+v.\n", loggerTag, request.id, request.Status(), request.getRequest())
						rawResponse, err := client.Do(request.getRequest())
						if err != nil && rawResponse == nil {
							c.logger.Printf("%s request failed due to %s, will resolve it with invalid response(-1).\n", loggerTag, err.Error())
							request.response.resolve(invalidResponse(fmt.Sprintf("Failed(%s)", err.Error()), -1))
							numFailed++
						} else {
							response, err := fromRawResponse(rawResponse)
							if err != nil {
								c.logger.Printf("%s unable to parse response body of %+v.\n", loggerTag, rawResponse)
								numSuccess++
							} else {
								numFailed++
							}
							request.response.resolve(response)
							c.logger.Printf("%s request(%s) has been resolved. Response: %+v.\n", loggerTag, request.id, response)
						}
					}
				}
				c.logger.Printf("%s client has stopped.\n", loggerTag)
				c.logger.Printf("%s client performance: [%d, %d, %d].\n", loggerTag, numRequests, numSuccess, numFailed)
				wg.Done()
			}(i, clientItr)
		}
		c.setStatus(PoolStatusRunning)
		wg.Wait()
		c.setStatus(PoolStatusStopped)
		c.logger.Printf("Client has stopped.")
	}()
}

func (c *ClientPool) Stop() {
	c.setStatus(PoolStatusTerminating)
	close(c.queue)
	for c.Status() != PoolStatusStopped {
	}
}

func (c *ClientPool) setStatus(status int) {
	c.rwMutex.Lock()
	defer c.rwMutex.Unlock()
	oldStatus := c.status
	c.status = status
	c.logger.Printf("Switched pool status from %s to %s\n", poolStatusStringMap[oldStatus], poolStatusStringMap[status])
}

func (c *ClientPool) Id() string {
	return c.id
}

func (c *ClientPool) Status() int {
	c.rwMutex.RLock()
	defer c.rwMutex.RUnlock()
	return c.status
}

func (c *ClientPool) request(request *http.Request) *TrackableRequest {
	loggerTag := "[Request]"
	c.logger.Printf("%s New request received: %+v\nCurrent queue size: %d\n", loggerTag, request, len(c.queue))
	trackableRequest := NewTrackableRequest(request)
	trackableRequest.setStatus(RequestStatusWaiting)
	c.queue <- trackableRequest
	return trackableRequest
}

func (c *ClientPool) Request(request *http.Request) *Response {
	tr := c.request(request)
	return tr.Response()
}

func (c *ClientPool) RequestAsync(request *http.Request) *TrackableRequest {
	return c.request(request)
}

func (c *ClientPool) batchRequest(requests []*http.Request) []*TrackableRequest {
	res := make([]*TrackableRequest, len(requests))
	for i, req := range requests {
		res[i] = c.request(req)
	}
	return res
}

func (c *ClientPool) BatchRequest(requests []*http.Request) []*Response {
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

func (c *ClientPool) BatchRequestAsync(requests []*http.Request) []*TrackableRequest {
	return c.batchRequest(requests)
}

func (c *ClientPool) Verbose(use bool) {
	c.logger.Verbose(use)
}

// global client
var globalPool *ClientPool

func initGlobalPool() {
	numCpu := runtime.NumCPU()
	globalPool = NewPool("global", numCpu, numCpu*16, 30)
}

func DoRequest(request *http.Request) *Response {
	return globalPool.Request(request)
}

func DoRequestAsync(request *http.Request) *TrackableRequest {
	return globalPool.RequestAsync(request)
}

func DoBatchRequest(requests []*http.Request) []*Response {
	return globalPool.BatchRequest(requests)
}

func DoBatchRequestAsync(requests []*http.Request) []*TrackableRequest {
	return globalPool.BatchRequestAsync(requests)
}

func Status() int {
	return globalPool.Status()
}
