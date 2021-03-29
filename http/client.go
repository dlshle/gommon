package http

import (
	"fmt"
	"io"
	"log"
	"net/http"
	urlpkg "net/url"
	"os"
	"strings"
	"sync"
)

// Globals

// logger
var globalLogger = log.New(os.Stdout, "[Performance]", log.Ldate|log.Ltime|log.Lshortfile)

// request validators
var requestValidators = []func(r *http.Request) error{
	// url validator
	func(r *http.Request) error {
		if r.URL == nil || r.URL.String() == "" {
			return NewClientError("Invalid request url(%s).", ErrInvalidRequestUrl)
		}
		return nil
	},
	// method validator
	func(r *http.Request) error {
		if len(r.Method) == 0 {
			return NewClientError("Request method not set.", ErrInvalidRequestMethod)
		}
		return nil
	},
}
var requestValidationHandlers = []func(errors []error) error{
	// 0 error
	func(errors []error) error {
		return nil
	},
	// 1 error
	func(errors []error) error {
		for _, err := range errors {
			if err != nil {
				return err
			}
		}
		return nil
	},
	// > 1 errors
	func(errors []error) error {
		err_msg := ""
		for _, err := range errors {
			if err != nil {
				err_msg = fmt.Sprintf("%s\n%s", err_msg, err.Error())
			}
		}
		return NewClientError(err_msg, ErrInvalidRequest)
	},
}

// init
func init() {
}

// Errors

// error codes
const (
	ErrInvalidRequest       = 0
	ErrInvalidRequestUrl    = 1
	ErrInvalidRequestMethod = 2
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
	Build() (*http.Request, error)
	Method(method string) *RequestBuilder
	URL(url string) *RequestBuilder
	Header(header http.Header) *RequestBuilder
	Body(body io.ReadCloser) *RequestBuilder
	StringBody(body string) *RequestBuilder
}

// RequestBuilder helpers
func validateRequest(request *http.Request) error {
	errors := make([]error, len(requestValidators))
	numErrors := 0
	for i, validator := range requestValidators {
		err := validator(request)
		if err != nil {
			numErrors += 1
			errors[i] = validator(request)
		}
	}
	return requestValidationHandlers[numErrors](errors)
}

func (b *RequestBuilder) Build() (*http.Request, error) {
	err := validateRequest(b.request)
	if err != nil {
		return nil, err
	}
	return b.request, nil
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

// TODO Response and Awaitable
