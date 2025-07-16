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
	Build() HTTPClient
}

type httpClientBuilder struct {
	transport  *http.Transport
	baseClient *http.Client
	client     *httpClient
}

func (h *httpClientBuilder) Id(id string) HTTPClientBuilder {
	h.client.id = id
	h.client.logger.Prefix(id)
	return h
}

func (h *httpClientBuilder) Logger(logger logging.Logger) HTTPClientBuilder {
	h.client.logger = logger
	return h
}

func (h *httpClientBuilder) TimeoutSec(timeout int) HTTPClientBuilder {
	h.baseClient.Timeout = time.Duration(timeout) * time.Second
	return h
}

func (h *httpClientBuilder) AddInterceptor(interceptor Interceptor) HTTPClientBuilder {
	if h.client.interceptors == nil {
		h.client.interceptors = make([]Interceptor, 0)
	}
	h.client.interceptors = append(h.client.interceptors, interceptor)
	return h
}

func (h *httpClientBuilder) WithInterceptors(interceptors ...Interceptor) HTTPClientBuilder {
	h.client.interceptors = interceptors
	return h
}

func (h *httpClientBuilder) MaxConcurrentRequests(n int) HTTPClientBuilder {
	h.client.workerSize = numWithinRange(n, 1, runtime.NumCPU()*32)
	return h
}

func (h *httpClientBuilder) MaxQueueSize(n int) HTTPClientBuilder {
	h.client.queue = make(chan TrackableRequest, numWithinRange(n, 1, runtime.NumCPU()*64))
	return h
}

func (h *httpClientBuilder) MaxConnsPerHost(n int) HTTPClientBuilder {
	numMaxConnsPerHost := numWithinRange(n, 1, runtime.NumCPU()*8)
	h.transport.MaxConnsPerHost = numMaxConnsPerHost
	h.transport.MaxIdleConnsPerHost = numMaxConnsPerHost
	h.transport.MaxIdleConns = numMaxConnsPerHost
	return h
}

func (h *httpClientBuilder) Build() HTTPClient {
	stopWg := new(sync.WaitGroup)
	h.baseClient.Transport = h.transport
	h.client.baseClient = h.baseClient
	h.client.stopWg = stopWg
	h.client.numWorkers = 0
	h.client.status = PoolStatusRunning
	return h.client
}

func NewBuilder() HTTPClientBuilder {
	ctx, cancelFunc := context.WithCancel(context.Background())
	return &httpClientBuilder{
		transport: http.DefaultTransport.(*http.Transport).Clone(),
		baseClient: &http.Client{
			Timeout: time.Minute,
		},
		client: &httpClient{
			ctx:        ctx,
			cancelFunc: cancelFunc,
			id:         "http_client",
			queue:      make(chan TrackableRequest, 128),
			logger:     logging.GlobalLogger.WithPrefix("http_client"),
			status:     PoolStatusIdle,
			rwMutex:    new(sync.RWMutex),
			workerSize: 5,
		},
	}
}
