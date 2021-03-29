package http_client

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

// RANDOM
var randomGenerator = rand.New(rand.NewSource(time.Now().UnixNano()))

const (
	// HTTP_CLIENT_SIZE
	MaxClientSize = 20
	MaxDelayTime  = 30 * 1000

	// HTTP methods
	GET    = "GET"
	POST   = "POST"
	PUT    = "PUT"
	DELETE = "DELETE"
	PATCH  = "PATCH"
	HEAD   = "HEAD"
	OPTION = "OPTION"
)

type HTTPError struct {
	code    int
	message string
}

func (err *HTTPError) Error() string {
	return err.message
}

func httpError(code int, message string) *HTTPError {
	return &HTTPError{code, message}
}

type HTTPRequest struct {
	Id              string
	Url             string
	Method          string
	Retry           int
	AuthFree        bool
	CustomizeHeader map[string]string
	Awaitable       chan *HTTPResponse
}

type HTTPRequestBuilder struct {
	request *HTTPRequest
}

type IHTTPRequestBuilder interface {
	Id(id string) *HTTPRequestBuilder
	Url(url string) *HTTPRequestBuilder
	Method(method string) *HTTPRequestBuilder
	Retry(retry int) *HTTPRequestBuilder
	AuthFree(authFree bool) *HTTPRequestBuilder
	CustomizeHeader(customizeHeader map[string]string) *HTTPRequestBuilder
	Build() *HTTPRequest
}

func (b *HTTPRequestBuilder) Id(id string) *HTTPRequestBuilder {
	b.request.Id = id
	return b
}

func (b *HTTPRequestBuilder) Url(url string) *HTTPRequestBuilder {
	b.request.Url = url
	return b
}

func (b *HTTPRequestBuilder) Method(method string) *HTTPRequestBuilder {
	b.request.Method = method
	return b
}

func (b *HTTPRequestBuilder) Retry(retry int) *HTTPRequestBuilder {
	b.request.Retry = retry
	return b
}

func (b *HTTPRequestBuilder) AuthFree(authFree bool) *HTTPRequestBuilder {
	b.request.AuthFree = authFree
	return b
}

func (b *HTTPRequestBuilder) CustomizeHeader(customizeHeader map[string]string) *HTTPRequestBuilder {
	if b.request.CustomizeHeader == nil {
		b.request.CustomizeHeader = make(map[string]string)
	}
	for key, val := range customizeHeader {
		b.request.CustomizeHeader[key] = val
	}
	return b
}

func (b *HTTPRequestBuilder) Build() *HTTPRequest {
	b.request.Id = strconv.FormatInt(randomGenerator.Int63n(time.Now().Unix()), 16)
	return b.request
}

func NewHTTPRequestBuilder() *HTTPRequestBuilder {
	request := &HTTPRequest{}
	builder := &HTTPRequestBuilder{request}
	return builder
}

type HTTPResponse struct {
	success bool
	code    int
	header  http.Header // usage just like map, can for each kv or ["headerKey"] gives an array of strings
	body    string
}

func newHTTPResponse(success bool, code int, header http.Header, body string) *HTTPResponse {
	return &HTTPResponse{success, code, header, body}
}

func newErrorHTTPResponse(errCode int, msg string) *HTTPResponse {
	return &HTTPResponse{success: false, code: errCode, body: msg}
}

func toHTTPResponse(resp *http.Response) (*HTTPResponse, error) {
	defer resp.Body.Close()
	statusCode := resp.StatusCode
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	httpResp := newHTTPResponse(statusCode >= 200 && statusCode <= 300, statusCode, resp.Header, string(body[:]))
	return httpResp, nil
}

type requestFilter func(request *HTTPRequest) bool

func defaultRequestFilterFunc(request *HTTPRequest) bool {
	if request.Url == "" {
		return false
	}
	if request.Method == "" {
		return false
	}
	if request.Awaitable == nil {
		request.Awaitable = make(chan *HTTPResponse)
	}
	return true
}

type HTTPRequestQueue struct {
	channel chan *HTTPRequest
	requestFilter
}

type IHTTPRequestQueue interface {
	enqueue(request *HTTPRequest)
	dequeue() *HTTPRequest
}

func (q *HTTPRequestQueue) enqueue(request *HTTPRequest) error {
	if !q.requestFilter(request) {
		return httpError(0, "filter failed")
	}
	q.channel <- request
	return nil
}

func (q *HTTPRequestQueue) dequeue() *HTTPRequest {
	r := <-q.channel
	return r
}

func newHTTPRequestQueue() *HTTPRequestQueue {
	return &HTTPRequestQueue{make(chan *HTTPRequest), defaultRequestFilterFunc}
}

type FutureHTTPResponse struct {
	channel  chan *HTTPResponse
	response *HTTPResponse
}

type AwaitableHTTPResponse interface {
	Await() *HTTPResponse
}

func (f *FutureHTTPResponse) Await() *HTTPResponse {
	channelResult := <-f.channel
	if channelResult != nil {
		f.response = channelResult
		close(f.channel)
	}
	return f.response
}

type RequestProcessor func(request *HTTPRequest) *HTTPRequest

type HTTPClient struct {
	rwLock            *sync.RWMutex
	isStarted         bool
	isTerminated      bool
	BaseUrl           string
	clients           []*http.Client
	requestQueue      *HTTPRequestQueue
	requestProcessors []RequestProcessor
	delayTime         int
}

type IHTTPClient interface {
	request(request *HTTPRequest) chan *HTTPResponse
	Request(request *HTTPRequest) *HTTPResponse
	AsyncRequest(request *HTTPRequest) *FutureHTTPResponse
	AddRequestProcessor(processor RequestProcessor)
	requestInPool(requests []*HTTPRequest) chan *HTTPResponse
	RequestInPool(requests []*HTTPRequest) *[]HTTPResponse
	toRawRequest(request *HTTPRequest) (*http.Request, error)
	hasStarted() bool
	start()
	hasTerminated() bool
	terminate()
}

func (c *HTTPClient) hasStarted() bool {
	c.rwLock.RLock()
	defer c.rwLock.RUnlock()
	return c.isStarted
}

func (c *HTTPClient) hasTerminated() bool {
	c.rwLock.RLock()
	defer c.rwLock.RUnlock()
	return c.isTerminated
}

func (c *HTTPClient) start() {
	if !c.hasStarted() {
		// set start to true
		c.rwLock.Lock()
		defer c.rwLock.Unlock()
		c.isStarted = true

		// start sequence where using goroutines to consume requests
		for _, client := range c.clients {
			go func() {
				// idx := strconv.FormatInt(randomGenerator.Int63n(time.Now().Unix()), 16)
				for !c.hasTerminated() {
					req := c.requestQueue.dequeue()
					awaitableChan := req.Awaitable
					rawRequest, toRawRequestErr := c.toRawRequest(req)
					if toRawRequestErr != nil {
						awaitableChan <- newErrorHTTPResponse(-1, toRawRequestErr.Error())
						continue
					}
					// fmt.Printf("client %s on request(%s) %+v\n", idx, req.Id, rawRequest)
					resp, err := client.Do(rawRequest)
					if err != nil {
						awaitableChan <- newErrorHTTPResponse(-1, err.Error())
					} else {
						httpResp, transformErr := toHTTPResponse(resp)
						if transformErr != nil {
							awaitableChan <- newErrorHTTPResponse(-1, err.Error())
						} else {
							awaitableChan <- httpResp
						}
					}
					time.Sleep(time.Duration(c.delayTime) * time.Millisecond)
				}
			}()
		}
	}
}

func (c *HTTPClient) terminate() {
	c.rwLock.Lock()
	defer c.rwLock.Unlock()
	if !c.isTerminated {
		c.isTerminated = true
	}
}

func (c *HTTPClient) request(request *HTTPRequest) chan *HTTPResponse {
	if !c.isStarted {
		c.start()
	}
	c.requestQueue.enqueue(request)
	return request.Awaitable
}

func (c *HTTPClient) Request(request *HTTPRequest) *HTTPResponse {
	channel := c.request(request)
	defer close(channel)
	response := <-channel
	return response
}

func (c *HTTPClient) requestInPool(requests []*HTTPRequest) chan *HTTPResponse {
	responseChannel := make(chan *HTTPResponse)
	for _, request := range requests {
		func(r *HTTPRequest) {
			go func() {
				responseChannel <- c.Request(r)
			}()
		}(request)
	}
	return responseChannel
}

func (c *HTTPClient) RequestInPool(requests []*HTTPRequest) []*HTTPResponse {
	size := len(requests)
	channel := c.requestInPool(requests)
	results := make([]*HTTPResponse, size, size)
	defer close(channel)
	for i := 0; i < size; i++ {
		response := <-channel
		results[i] = response
		// results = append(results, response)
	}
	return results
}

func (c *HTTPClient) AsyncRequest(request *HTTPRequest) *FutureHTTPResponse {
	respChannel := c.request(request)
	return &FutureHTTPResponse{respChannel, nil}
}

func (c *HTTPClient) AddRequestProcessor(processor RequestProcessor) {
	c.rwLock.RLock()
	defer c.rwLock.RUnlock()
	c.requestProcessors = append(c.requestProcessors, processor)
}

func (c *HTTPClient) toRawRequest(request *HTTPRequest) (*http.Request, error) {
	for _, processor := range c.requestProcessors {
		request = processor(request)
	}
	rawRequest, err := http.NewRequest(request.Method, c.BaseUrl+request.Url, nil)
	if err != nil {
		return nil, err
	}
	if request.CustomizeHeader != nil {
		for key, val := range request.CustomizeHeader {
			rawRequest.Header.Set(key, val)
		}
	}
	return rawRequest, nil
}

func NewHTTPClient(baseUrl string, numClients int, timeoutInSec int, delayTime int) *HTTPClient {
	if numClients > MaxClientSize {
		numClients = MaxClientSize
	}
	if delayTime < 0 {
		delayTime = 0
	}
	if delayTime > MaxDelayTime {
		delayTime = MaxDelayTime
	}
	rawClients := make([]*http.Client, numClients)
	for i := 0; i < numClients; i++ {
		rawClients[i] = newHTTPClient(timeoutInSec)
	}
	return &HTTPClient{new(sync.RWMutex), false, false, baseUrl, rawClients, newHTTPRequestQueue(), make([]RequestProcessor, 0, 5), delayTime}
}

func newHTTPClient(timeout int) *http.Client {
	return &http.Client{Timeout: time.Second * time.Duration(timeout)}
}

// Tests
func copyOne(request *HTTPRequest) *HTTPRequest {
	cpy := NewHTTPRequestBuilder().Url(request.Url).Method(request.Method).CustomizeHeader(request.CustomizeHeader).Build()
	return cpy
}

func copyRequest(request *HTTPRequest, amount int) []*HTTPRequest {
	list := make([]*HTTPRequest, amount)
	for i := 0; i < amount; i++ {
		list[i] = copyOne(request)
	}
	return list
}

func buildRCClient(baseUrl string, numClients int, delayTime int) *HTTPClient {
	return NewHTTPClient(baseUrl, numClients, 5, delayTime)
}

func buildRCRequestWithToken(url string, method string, token string) *HTTPRequest {
	customizeHeader := make(map[string]string)
	customizeHeader["Accept"] = "*/*"
	customizeHeader["Accept-Encoding"] = "gzip, deflate, br"
	customizeHeader["User-Agent"] = "PostmanRuntime/7.26.8"
	var actualToken string
	if token != "" {
		if token[:7] == "Bearer " {
			actualToken = token
		} else {
			actualToken = "Bearer " + token
		}
		customizeHeader["Authorization"] = actualToken
	}
	return NewHTTPRequestBuilder().Url(url).Method(method).CustomizeHeader(customizeHeader).Build()
}

var baseFlag = flag.String("b", "https://api-xmnup.lab.nordigy.ru", "need to specify the base Url for the client")
var cFlag = flag.Int("c", 10, "need to specify the number of concurrent running clients")
var delayTimeFlag = flag.Int("d", 0, "need to specify the delay time for each request")
var urlFlag = flag.String("u", "", "need to specify the Url for the request")
var methodFlag = flag.String("m", "GET", "need to specify the Method for the request")
var tokenFlag = flag.String("t", "", "need to specify the token for the request")
var nFlag = flag.Int("n", 10, "need to specify how many requests do you want to send")

func runClientTest() {
	flag.Parse()
	baseUrl := *baseFlag
	num_concurrency := *cFlag
	delayTime := *delayTimeFlag
	url := *urlFlag
	method := *methodFlag
	token := *tokenFlag
	num_requests := *nFlag
	client := buildRCClient(baseUrl, num_concurrency, delayTime)
	requestInstance := buildRCRequestWithToken(url, method, token)
	requests := copyRequest(requestInstance, num_requests)

	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Are you sure you want to send request to %s with %d clients and %d requests?\nY/n", (baseUrl + url), num_concurrency, num_requests)
	text, input_err := reader.ReadString('\n')
	if input_err != nil {
		fmt.Println("Input error, terminating the client...")
		return
	}
	if text == "Y\n" {
		responses := client.RequestInPool(requests)
		num_all := 0
		num_success := 0
		for _, res := range responses {
			num_all += 1
			if res.success {
				fmt.Println("success: ", res.success)
				num_success += 1
			} else {
				fmt.Printf("error code: %d err body: %s\n", res.code, res.body)
			}
		}
		fmt.Printf("success: %d / %d failed: %d / %d\n", num_success, num_all, (num_all - num_success), num_all)
	}
}
