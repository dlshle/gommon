package http

import (
	"context"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/dlshle/gommon/logging"
)

type HTTPClientBuilder interface {
	Id(id string) HTTPClientBuilder
	Logger(logger logging.Logger) HTTPClientBuilder
	AddInterceptor(interceptor Interceptor) HTTPClientBuilder
	WithInterceptors(interceptors ...Interceptor) HTTPClientBuilder
	TimeoutSec(timeout int) HTTPClientBuilder
	MaxConcurrentRequests(n int) HTTPClientBuilder
	MaxQueueSize(n int) HTTPClientBuilder
	MaxConnsPerHost(n int) HTTPClientBuilder
	MaxResponseBodySize(n int64) HTTPClientBuilder
	Build() Client
}

type httpClientBuilder struct {
	id                    string
	logger                logging.Logger
	customLogger          bool
	interceptors          []Interceptor
	timeout               time.Duration
	maxConcurrentRequests int
	maxQueueSize          int
	maxConnsPerHost       int
	maxResponseBodySize   int64
	transport             *http.Transport
}

func (h *httpClientBuilder) Id(id string) HTTPClientBuilder {
	h.id = id
	return h
}

func (h *httpClientBuilder) Logger(logger logging.Logger) HTTPClientBuilder {
	h.logger = logger
	h.customLogger = true
	return h
}

func (h *httpClientBuilder) TimeoutSec(timeout int) HTTPClientBuilder {
	if timeout < 0 {
		timeout = 0
	}
	h.timeout = time.Duration(timeout) * time.Second
	return h
}

func (h *httpClientBuilder) AddInterceptor(interceptor Interceptor) HTTPClientBuilder {
	if h.interceptors == nil {
		h.interceptors = make([]Interceptor, 0)
	}
	h.interceptors = append(h.interceptors, interceptor)
	return h
}

func (h *httpClientBuilder) WithInterceptors(interceptors ...Interceptor) HTTPClientBuilder {
	h.interceptors = interceptors
	return h
}

func (h *httpClientBuilder) MaxConcurrentRequests(n int) HTTPClientBuilder {
	h.maxConcurrentRequests = n
	return h
}

func (h *httpClientBuilder) MaxQueueSize(n int) HTTPClientBuilder {
	h.maxQueueSize = n
	return h
}

func (h *httpClientBuilder) MaxConnsPerHost(n int) HTTPClientBuilder {
	h.maxConnsPerHost = n
	return h
}

func (h *httpClientBuilder) MaxResponseBodySize(n int64) HTTPClientBuilder {
	h.maxResponseBodySize = n
	return h
}

func (h *httpClientBuilder) Build() Client {
	ctx, cancelFunc := context.WithCancel(context.Background())

	queueSize := numWithinRange(h.maxQueueSize, 1, runtime.NumCPU()*64)
	workerSize := numWithinRange(h.maxConcurrentRequests, 1, runtime.NumCPU()*32)
	numMaxConnsPerHost := numWithinRange(h.maxConnsPerHost, 1, runtime.NumCPU()*8)

	transport := h.transport.Clone()
	transport.MaxConnsPerHost = numMaxConnsPerHost
	transport.MaxIdleConnsPerHost = numMaxConnsPerHost
	transport.MaxIdleConns = numMaxConnsPerHost

	baseClient := &http.Client{
		Timeout:   h.timeout,
		Transport: transport,
	}

	logger := h.logger
	if !h.customLogger {
		logger = logging.GlobalLogger.WithPrefix(h.id)
	}

	client := &httpClient{
		ctx:                 ctx,
		cancelFunc:          cancelFunc,
		id:                  h.id,
		interceptors:        append([]Interceptor(nil), h.interceptors...),
		queue:               make(chan TrackableRequest, queueSize),
		logger:              logger,
		status:              PoolStatusRunning,
		rwMutex:             new(sync.RWMutex),
		workerSize:          workerSize,
		numWorkers:          0,
		baseClient:          baseClient,
		stopWg:              new(sync.WaitGroup),
		maxResponseBodySize: h.maxResponseBodySize,
	}
	client.startWorkers()
	return client
}

func NewBuilder() HTTPClientBuilder {
	var transport *http.Transport
	if defaultTransport, ok := http.DefaultTransport.(*http.Transport); ok {
		transport = defaultTransport.Clone()
	} else {
		transport = &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			MaxConnsPerHost:     100,
		}
	}

	return &httpClientBuilder{
		id:                    "http_client",
		interceptors:          make([]Interceptor, 0),
		timeout:               time.Minute,
		maxConcurrentRequests: 5,
		maxQueueSize:          128,
		maxConnsPerHost:       100,
		maxResponseBodySize:   0,
		transport:             transport,
	}
}
