package http

import (
	"net/http"
	"testing"
)

func TestMockHTTPClient_SingleResponse(t *testing.T) {
	// Create mock client and test request
	client := NewMockHTTPClient()
	req, _ := http.NewRequest("GET", "http://example.com", nil)

	// Create test response
	expectedResp := &Response{
		Code: 200,
		Body: []byte("test response"),
	}

	// Set response and get actual response
	client.SetResponseForRequest(req, expectedResp)
	actualResp, err := client.DoRequest(req)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify response matches expected values
	if actualResp.Code != 200 || string(actualResp.Body) != "test response" {
		t.Errorf("Expected success=true, code=200, body='test response', got %+v", actualResp)
	}
}

func TestMockHTTPClient_ResponseSequence(t *testing.T) {
	// Create mock client and test request
	client := NewMockHTTPClient()
	req, _ := http.NewRequest("GET", "http://example.com/sequence", nil)

	// Create test responses
	resp1 := &Response{Code: 200, Body: []byte("first")}
	resp2 := &Response{Code: 200, Body: []byte("second")}
	resp3 := &Response{Code: 200, Body: []byte("third")}

	// Set response sequence
	client.SetResponsesForRequest(req, []*Response{resp1, resp2, resp3})

	// Get responses for multiple requests
	respA, err := client.DoRequest(req)
	if err != nil {
		t.Error(err)
	}
	respB, err := client.DoRequest(req)
	if err != nil {
		t.Error(err)
	}
	respC, err := client.DoRequest(req)
	if err != nil {
		t.Error(err)
	}

	// Verify response sequence
	if string(respA.Body) != "first" || string(respB.Body) != "second" || string(respC.Body) != "third" {
		t.Errorf("Expected response sequence first -> second -> third, got %s -> %s -> %s",
			string(respA.Body), string(respB.Body), string(respC.Body))
	}

	// Test cycling behavior - should keep returning last response
	respD, err := client.DoRequest(req)
	if err != nil {
		t.Error(err)
	}
	if string(respD.Body) != "third" {
		t.Errorf("Expected continued use of last response, got %s", string(respD.Body))
	}
}

func TestMockHTTPClient_DifferentRequests(t *testing.T) {
	// Create mock client
	client := NewMockHTTPClient()

	// Create different requests and responses
	req1, _ := http.NewRequest("GET", "http://example.com/one", nil)
	req2, _ := http.NewRequest("GET", "http://example.com/two", nil)

	resp1 := &Response{Code: 200, Body: []byte("one")}
	resp2 := &Response{Code: 200, Body: []byte("two")}

	// Set different responses for different requests
	client.SetResponseForRequest(req1, resp1)
	client.SetResponseForRequest(req2, resp2)

	// Test each request independently
	respA, err := client.DoRequest(req1)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	respB, err := client.DoRequest(req2)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if string(respA.Body) != "one" || string(respB.Body) != "two" {
		t.Errorf("Expected different responses for different requests, got %s and %s",
			string(respA.Body), string(respB.Body))
	}
}

func TestMockHTTPClient_UnconfiguredRequest(t *testing.T) {
	// Create mock client and test request
	client := NewMockHTTPClient()
	req, _ := http.NewRequest("GET", "http://example.com/unconfigured", nil)

	// Test unconfigured request
	_, err := client.DoRequest(req)
	if err == nil {
		t.Errorf("Expected error!")
	}

	if err.Error() != "no predefined response for request" {
		t.Errorf("Expected error message: %s, but got: %s", "no predefined response for request", err.Error())
	}
}
