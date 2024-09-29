package http

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dlshle/gommon/logging"
	"github.com/dlshle/gommon/utils"
)

// TODO, need a customized request capable of retry
// TODO, Go http client already has connection reusability built-in, so no need to use the producer/consumer pattern here unless request limitation is required

// init
func init() {
	initRequestStatusErrorMaps()
	initPoolStatusStringMap()
}

type httpClient struct {
	ctx         context.Context
	cancelFunc  func()
	id          string
	queue       chan TrackableRequest
	logger      logging.Logger
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

func NewHTTPClient(maxConcurrentRequests, maxQueueSize, timeoutInSec int) HTTPClient {
	return New(utils.RandomStringWithSize(5), maxConcurrentRequests, maxQueueSize, timeoutInSec)
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

func (c *httpClient) tryToStartNewWorker() {
	if c.pendingRequests() > 0 && c.workerCount() < c.workerSize {
		c.stopWg.Add(1)
		go c.workerFunc(int(c.incrementAndGetWorkerCount()))
	}
}

func (c *httpClient) workerFunc(id int) {
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
			if request.Status() != RequestStatusWaiting {
				taggedLogger.Debugf(c.ctx, "skip request(%s) due to invalid status(%d).", request.ID(), request.Status())
			} else {
				numRequests++
				success := c.executeRequest(request)
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
	request.setStatus(RequestStatusInProgress)
	c.logger.Debugf(c.ctx, "worker has acquired request(%s, %d) with rawRequest %+v.", request.id, request.Status(), request.getRequest())
	rawResponse, err := c.baseClient.Do(request.getRequest())
	request.complete()
	if err != nil || rawResponse == nil {
		c.logger.Debugf(c.ctx, "request failed due to %s, will resolve it with invalid response(-1).", err.Error())
		request.response.resolve(invalidResponse("failed: "+err.Error(), -1))
		success = false
	} else {
		response, err := fromRawResponse(rawResponse)
		if err != nil {
			c.logger.Debugf(c.ctx, "unable to parse response body of %+v.\n", rawResponse)
			success = false
		} else {
			success = true
		}
		request.response.resolve(response)
		c.logger.Debugf(c.ctx, "request(%s) has been resolved. Response: %+v.\n", request.id, response)
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

func (c *httpClient) request(request *http.Request) TrackableRequest {
	loggerTag := "[Handle]"
	c.logger.Debugf(c.ctx, "%s New request received: %+v\nCurrent queue size: %d\n", loggerTag, request, len(c.queue))
	tRequest := newTrackableRequest(request)
	tRequest.setStatus(RequestStatusWaiting)
	if c.Status() != PoolStatusRunning {
		tRequest.response.resolve(invalidResponse("client is closed", -1))
		atomic.AddInt32(&c.numExceeded, 1)
		return tRequest
	}
	if c.isQueueSizeExceeded() {
		c.executeRequest(tRequest)
		return tRequest
	}
	c.queue <- tRequest
	c.tryToStartNewWorker()
	return tRequest
}

func (c *httpClient) DoRequest(request *http.Request) *Response {
	tRequest := newTrackableRequest(request)
	tRequest.setStatus(RequestStatusWaiting)
	c.executeRequest(tRequest)
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
	if !use {
		c.logger.SetWaterMark(logging.FATAL)
	} else {
		c.logger.SetWaterMark(logging.DEBUG)
	}
}
