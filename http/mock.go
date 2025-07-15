package http

import (
	"errors"
	"net/http"
	"sync"
)

// MockHTTPClient is a test double for HTTPClient that allows predefined responses based on requests
type MockHTTPClient struct {
	responses    map[string][]*Response // Map of request keys to response sequences
	requestCount map[string]int         // Tracks how many times each request has been made
	mutex        sync.Mutex
}

// NewMockHTTPClient creates a new instance of MockHTTPClient
func NewMockHTTPClient() *MockHTTPClient {
	return &MockHTTPClient{
		responses:    make(map[string][]*Response),
		requestCount: make(map[string]int),
	}
}

// DoRequest returns a predefined response based on the request and its invocation count
func (m *MockHTTPClient) DoRequest(request *http.Request) (*Response, error) {
	m.mutex.Lock()
	key := m.generateRequestKey(request)
	count := m.requestCount[key]

	// If no predefined response exists, return a default error response
	if _, exists := m.responses[key]; !exists {
		m.mutex.Unlock()
		return nil, errors.New("no predefined response for request")
	}

	// If the request has been called more times than responses available, use the last response
	respIndex := count
	if respIndex >= len(m.responses[key]) {
		respIndex = len(m.responses[key]) - 1
	}

	// Increment request count and unlock
	m.requestCount[key]++
	response := m.responses[key][respIndex]
	m.mutex.Unlock()

	return response, nil
}

// Request is implemented to satisfy the HTTPClient interface but simply delegates to DoRequest
func (m *MockHTTPClient) Request(request *http.Request) (*Response, error) {
	return m.DoRequest(request)
}

func (m *MockHTTPClient) RequestAsync(request *http.Request) AwaitableResponse {
	ar := newAwaitableResponse()
	r, e := m.Request(request)
	if e != nil {
		ar.reject(e)
	} else {
		ar.resolve(r)
	}
	return ar
}

// Id returns a fixed ID for the mock client
func (m *MockHTTPClient) Id() string {
	return "mock-client"
}

// Status returns a fixed status for the mock client
func (m *MockHTTPClient) Status() int {
	return 0
}

// Stop is a no-op for the mock client
func (m *MockHTTPClient) Stop() {
	// No-op
}

// Verbose is a no-op for the mock client
func (m *MockHTTPClient) Verbose(use bool) {
	// No-op
}

// SetResponseForRequest configures the mock client to return a specific response for a given request
func (m *MockHTTPClient) SetResponseForRequest(request *http.Request, response *Response) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	key := m.generateRequestKey(request)
	m.responses[key] = []*Response{response}
	m.requestCount[key] = 0
}

// SetResponsesForRequest configures the mock client to return a sequence of responses for a given request
func (m *MockHTTPClient) SetResponsesForRequest(request *http.Request, responses []*Response) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	key := m.generateRequestKey(request)
	m.responses[key] = responses
	m.requestCount[key] = 0
}

// generateRequestKey creates a unique key for a request based on its method and URL
func (m *MockHTTPClient) generateRequestKey(request *http.Request) string {
	return request.Method + ":" + request.URL.String()
}

// Reset clears all configured responses and request counts
func (m *MockHTTPClient) Reset() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.responses = make(map[string][]*Response)
	m.requestCount = make(map[string]int)
}
