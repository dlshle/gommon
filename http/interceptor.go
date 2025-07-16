package http

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/dlshle/gommon/logging"
)

type Interceptor = func(request *Request, next func(*Request) (*Response, error)) (*Response, error)

func intercept(interceptors []Interceptor, request *Request, requestExecutor func(*Request) (*Response, error)) (*Response, error) {
	if len(interceptors) == 0 {
		return requestExecutor(request)
	}
	return interceptors[0](request, func(currReq *Request) (*Response, error) {
		select {
		case <-currReq.Context().Done():
			return nil, currReq.Context().Err()
		default:
		}
		if len(interceptors) > 1 {
			return intercept(interceptors[1:], currReq, requestExecutor)
		}
		return requestExecutor(currReq)
	})
}

func CurlInterceptor(request *Request, next func(*Request) (*Response, error)) (*Response, error) {
	curl, err := requestToCurl(request)
	if err != nil {
		logging.GlobalLogger.Warnf(context.Background(), "Failed to convert request into curl command: %s", err)
	} else {
		logging.GlobalLogger.Info(context.Background(), "Curl command: "+curl)
	}
	return next(request)
}

func requestToCurl(req *http.Request) (string, error) {
	var curlCmd bytes.Buffer

	curlCmd.WriteString("curl -X " + req.Method)

	for key, values := range req.Header {
		for _, value := range values {
			curlCmd.WriteString(fmt.Sprintf(" -H '%s: %s'", key, value))
		}
	}

	if req.Body != nil {
		bodyBuf := new(bytes.Buffer)
		_, err := bodyBuf.ReadFrom(req.Body)
		if err != nil {
			return "", err
		}
		req.Body = io.NopCloser(bytes.NewBuffer(bodyBuf.Bytes()))
		if bodyBuf.Len() > 0 {
			bodyStr := strings.ReplaceAll(bodyBuf.String(), "'", "'\\''")
			curlCmd.WriteString(fmt.Sprintf(" -d '%s'", bodyStr))
		}
	}

	curlCmd.WriteString(" '" + req.URL.String() + "'")

	return curlCmd.String(), nil
}
