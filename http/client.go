package http

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dlshle/gommon/errors"
	"github.com/dlshle/gommon/logging"
	"github.com/dlshle/gommon/utils"
)

// TODO, need a customized request capable of retry
func init() {
	initPoolStatusStringMap()
}

type httpClient struct {
	ctx          context.Context
	cancelFunc   func()
	id           string
	interceptors []Interceptor
	queue        chan TrackableRequest
	logger       logging.Logger
	status       int
	rwMutex      *sync.RWMutex
	workerSize   int
	numWorkers   int32
	baseClient   *http.Client
	stopWg       *sync.WaitGroup
	numExceeded  int32
}

// deprecated
type HTTPClient interface {
	Id() string
	DoRequest(request *http.Request) (*Response, error)
	Request(request *http.Request) (*Response, error)
	RequestAsync(request *http.Request) AwaitableResponse
	Verbose(use bool)
	Status() int
	Stop()
}

type Client = HTTPClient

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

func NewHTTPClient(maxConcurrentRequests, maxQueueSize, timeoutInSec int) Client {
	return New(utils.RandomStringWithSize(5), maxConcurrentRequests, maxQueueSize, timeoutInSec)
}

func New(id string, maxConcurrentRequests, maxQueueSize, timeoutInSec int) Client {
	ctx, cancelFunc := context.WithCancel(context.Background())
	stopWg := new(sync.WaitGroup)
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
		[]Interceptor{},
		make(chan TrackableRequest, maxQueueSize),
		logging.GlobalLogger.WithPrefix("http-" + id).WithWaterMark(logging.FATAL),
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

func (c *httpClient) maybeStartNewWorker() {
	if c.pendingRequests() > 0 && c.workerCount() < c.workerSize {
		c.stopWg.Add(1)
		go c.workerRoutine(int(c.incrementAndGetWorkerCount()))
	}
}

func (c *httpClient) workerRoutine(id int) {
	defer c.completeWorker()
	numRequests := 0
	numSuccess := 0
	numFailed := 0
	taggedLogger := c.logger.WithPrefix(fmt.Sprintf("[Worker-%d]", id))
	taggedLogger.Debugf(c.ctx, "worker has started.")
	shouldContinue := true
	for shouldContinue {
		select {
		case req, isOpen := <-c.queue:
			if !isOpen {
				shouldContinue = false
				break
			}
			request := req.(*trackableRequest)
			numRequests++
			success := c.executeRequest(request)
			if success {
				numSuccess++
			} else {
				numFailed++
			}
			if c.pendingRequests() == 0 {
				shouldContinue = false
			}
		case <-c.ctx.Done():
			shouldContinue = false
		}
	}
	taggedLogger.Debugf(c.ctx, "worker lifecycle is finished, numRequests: %d, numSuccessRequests: %d, numFailedRequests: %d", numRequests, numSuccess, numFailed)
}

func (c *httpClient) completeWorker() {
	if recovered := recover(); recovered != nil {
		c.logger.Errorf(c.ctx, "worker has crashed with error: %v", recovered)
	}
	c.decrementWorkerCount()
	c.stopWg.Done()
}

func (c *httpClient) executeRequest(request *trackableRequest) (success bool) {
	defer request.complete()
	c.logger.Debugf(c.ctx, "worker has acquired request(%s) with rawRequest %+v.", request.id, request.getRequest())
	resp, err := intercept(c.interceptors, request.getRequest(), func(req *Request) (*Response, error) {
		rawResponse, err := c.baseClient.Do(request.getRequest())
		if err != nil || rawResponse == nil {
			c.logger.Debugf(c.ctx, "request failed due to %s, will resolve it with invalid response(-1).", err.Error())
			return nil, err
		} else {
			response, err := fromRawResponse(rawResponse)
			if err != nil {
				c.logger.Debugf(c.ctx, "unable to parse response body of %+v.\n", rawResponse)
				return nil, err
			} else {
				return response, nil
			}
		}
	})
	if err != nil {
		request.response.reject(err)
		success = false
	} else {
		request.response.resolve(resp)
		success = true
		c.logger.Debugf(c.ctx, "request(%s) has been resolved. Response: %+v.\n", request.id, resp)
	}
	return
}

func (c *httpClient) Stop() {
	c.cancelFunc()
	c.setStatus(PoolStatusTerminating)
	close(c.queue)
	c.stopWg.Wait()
	c.setStatus(PoolStatusStopped)
}

func (c *httpClient) setStatus(status int) {
	c.rwMutex.Lock()
	defer c.rwMutex.Unlock()
	oldStatus := c.status
	c.status = status
	c.logger.Debugf(c.ctx, "Switched pool status from %s to %s\n", poolStatusStringMap[oldStatus], poolStatusStringMap[status])
}

func (c *httpClient) Id() string {
	return c.id
}

func (c *httpClient) Status() int {
	c.rwMutex.RLock()
	defer c.rwMutex.RUnlock()
	return c.status
}

func (c *httpClient) request(request *http.Request) *awaitableResponse {
	c.logger.Debugf(c.ctx, "New request received: %+v\nCurrent queue size: %d\n", request, len(c.queue))
	tRequest := newTrackableRequest(request)
	if c.Status() != PoolStatusRunning {
		tRequest.response.reject(errors.Error("client is closed"))
		atomic.AddInt32(&c.numExceeded, 1)
		resp := tRequest.response
		tRequest.complete()
		return resp
	}
	if c.isQueueSizeExceeded() {
		c.executeRequest(tRequest)
		return tRequest.response
	}
	c.queue <- tRequest
	c.maybeStartNewWorker()
	return tRequest.response
}

func (c *httpClient) DoRequest(request *http.Request) (*Response, error) {
	tRequest := newTrackableRequest(request)
	c.executeRequest(tRequest)
	return tRequest.WaitAndGetResponse()
}

func (c *httpClient) Request(request *http.Request) (*Response, error) {
	return c.request(request).Get()
}

func (c *httpClient) RequestAsync(request *http.Request) AwaitableResponse {
	return c.request(request)
}

func (c *httpClient) Verbose(use bool) {
	if !use {
		c.logger.SetWaterMark(logging.FATAL)
	} else {
		c.logger.SetWaterMark(logging.DEBUG)
	}
}
