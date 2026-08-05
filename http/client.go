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

func init() {
	initPoolStatusStringMap()
}

type httpClient struct {
	ctx                 context.Context
	cancelFunc          func()
	id                  string
	interceptors        []Interceptor
	queue               chan TrackableRequest
	logger              logging.Logger
	status              int
	rwMutex             *sync.RWMutex
	workerSize          int
	numWorkers          int32
	baseClient          *http.Client
	stopWg              *sync.WaitGroup
	maxResponseBodySize int64
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
	timeoutDuration := time.Duration(0)
	if timeout > 0 {
		timeoutDuration = time.Second * time.Duration(timeout)
	}
	transport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		// DefaultTransport has been replaced; use it as-is. Connection limits won't be tuned.
		return &http.Client{
			Timeout: timeoutDuration,
		}
	}
	t := transport.Clone()
	t.MaxIdleConns = 100
	t.MaxConnsPerHost = 100
	t.MaxIdleConnsPerHost = 100
	return &http.Client{
		Timeout:   timeoutDuration,
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
	client := &httpClient{
		ctx:          ctx,
		cancelFunc:   cancelFunc,
		id:           id,
		interceptors: []Interceptor{},
		queue:        make(chan TrackableRequest, maxQueueSize),
		logger:       logging.GlobalLogger.WithPrefix("http-" + id).WithWaterMark(logging.FATAL),
		status:       PoolStatusRunning,
		rwMutex:      new(sync.RWMutex),
		workerSize:   maxConcurrentRequests,
		numWorkers:   0,
		baseClient:   newHTTPClient(timeoutInSec),
		stopWg:       stopWg,
	}
	client.startWorkers()
	return client
}

func (c *httpClient) startWorkers() {
	for i := 0; i < c.workerSize; i++ {
		c.stopWg.Add(1)
		workerLogger := c.logger.WithPrefix(fmt.Sprintf("[Worker-%d]", i+1))
		go c.workerRoutine(i+1, workerLogger)
		atomic.AddInt32(&c.numWorkers, 1)
	}
}

func (c *httpClient) decrementWorkerCount() {
	atomic.AddInt32(&c.numWorkers, -1)
}

func (c *httpClient) workerCount() int {
	return int(atomic.LoadInt32(&c.numWorkers))
}

func (c *httpClient) workerRoutine(id int, logger logging.Logger) {
	defer c.completeWorker()
	logger.Debugf(c.ctx, "worker has started.")
	for {
		select {
		case req := <-c.queue:
			request := req.(*trackableRequest)
			c.executeRequest(request, logger)
		case <-c.ctx.Done():
			logger.Debugf(c.ctx, "worker is exiting because client context is done; draining remaining queue.")
			for {
				select {
				case req := <-c.queue:
					request := req.(*trackableRequest)
					c.executeRequest(request, logger)
				default:
					return
				}
			}
		}
	}
}

func (c *httpClient) completeWorker() {
	if recovered := recover(); recovered != nil {
		c.logger.Errorf(c.ctx, "worker has crashed with error: %v", recovered)
	}
	c.decrementWorkerCount()
	c.stopWg.Done()
}

func (c *httpClient) executeRequest(request *trackableRequest, logger logging.Logger) (success bool) {
	defer request.complete()
	logger.Debugf(c.ctx, "worker has acquired request(%s).", request.id)
	resp, err := intercept(c.interceptors, request.getRequest(), func(req *Request) (*Response, error) {
		// Reset body from GetBody so retries and interceptor chains see a fresh body each time.
		if req.GetBody != nil {
			body, bodyErr := req.GetBody()
			if bodyErr != nil {
				return nil, bodyErr
			}
			req.Body = body
		}
		rawResponse, err := c.baseClient.Do(req)
		if err != nil {
			logger.Debugf(c.ctx, "request(%s) failed: %v", request.id, err)
			return nil, err
		}
		response, err := fromRawResponse(rawResponse, c.maxResponseBodySize)
		if err != nil {
			logger.Debugf(c.ctx, "request(%s) unable to parse response body: %v", request.id, err)
			return nil, err
		}
		return response, nil
	})
	if err != nil {
		request.response.reject(err)
		success = false
	} else {
		request.response.resolve(resp)
		success = true
		logger.Debugf(c.ctx, "request(%s) has been resolved with code %d.", request.id, resp.Code)
	}
	return
}

func (c *httpClient) Stop() {
	c.rwMutex.Lock()
	if c.status != PoolStatusRunning {
		c.rwMutex.Unlock()
		return
	}
	c.status = PoolStatusTerminating
	c.rwMutex.Unlock()

	c.cancelFunc()
	c.stopWg.Wait()
	c.setStatus(PoolStatusStopped)
}

func (c *httpClient) setStatus(status int) {
	c.rwMutex.Lock()
	oldStatus := c.status
	c.status = status
	c.rwMutex.Unlock()
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
	tRequest := newTrackableRequest(request)
	c.rwMutex.Lock()
	if c.status != PoolStatusRunning {
		c.rwMutex.Unlock()
		tRequest.response.reject(errors.Error("client is closed"))
		tRequest.complete()
		return tRequest.response
	}
	select {
	case c.queue <- tRequest:
		c.rwMutex.Unlock()
	default:
		c.rwMutex.Unlock()
		tRequest.response.reject(errors.Error("request queue is full"))
		tRequest.complete()
	}
	return tRequest.response
}

func (c *httpClient) DoRequest(request *http.Request) (*Response, error) {
	return c.request(request).Get()
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
