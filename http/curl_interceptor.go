package http

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/dlshle/gommon/logging"
)

// CurlInterceptor logs an equivalent curl command for the request before executing it.
// WARNING: this logs the full request body and headers, which may include sensitive
// data (credentials, PII, tokens). Use with care in production environments.
func CurlInterceptor(request *Request, next func(*Request) (*Response, error)) (*Response, error) {
	curl, err := requestToCurl(request)
	if err != nil {
		logging.GlobalLogger.Warnf(context.Background(), "Failed to convert request into curl command: %s", err)
	} else {
		logging.GlobalLogger.Info(context.Background(), "Curl command: "+curl)
	}
	return next(request)
}

func requestToCurl(req *Request) (string, error) {
	var curlCmd bytes.Buffer

	curlCmd.WriteString("curl -X " + req.Method)

	for key, values := range req.Header {
		for _, value := range values {
			curlCmd.WriteString(fmt.Sprintf(" -H '%s: %s'", key, value))
		}
	}

	bodyBuf, err := snapshotRequestBody(req)
	if err != nil {
		return "", err
	}
	if bodyBuf.Len() > 0 {
		bodyStr := strings.ReplaceAll(bodyBuf.String(), "'", "'\\''")
		curlCmd.WriteString(fmt.Sprintf(" -d '%s'", bodyStr))
	}

	curlCmd.WriteString(" '" + req.URL.String() + "'")

	return curlCmd.String(), nil
}

// snapshotRequestBody reads the request body for curl logging and leaves the
// request with a rewindable body so subsequent interceptors/retries can read it.
func snapshotRequestBody(req *Request) (*bytes.Buffer, error) {
	bodyBuf := new(bytes.Buffer)

	if req.GetBody != nil {
		bodyReader, err := req.GetBody()
		if err != nil {
			return bodyBuf, err
		}
		defer bodyReader.Close()
		if _, err := bodyBuf.ReadFrom(bodyReader); err != nil {
			return bodyBuf, err
		}
		// Restore req.Body with a fresh copy and keep GetBody usable.
		restored, err := req.GetBody()
		if err != nil {
			return bodyBuf, err
		}
		req.Body = restored
		return bodyBuf, nil
	}

	if req.Body != nil {
		if _, err := bodyBuf.ReadFrom(req.Body); err != nil {
			_ = req.Body.Close()
			return bodyBuf, err
		}
		_ = req.Body.Close()
		data := bodyBuf.Bytes()
		req.Body = io.NopCloser(bytes.NewReader(data))
		req.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(data)), nil
		}
	}

	return bodyBuf, nil
}
