package http

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/dlshle/gommon/logging"
)

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

	if req.GetBody != nil || req.Body != nil {
		bodyBuf := new(bytes.Buffer)
		var (
			bodyReader io.ReadCloser = nil
			err        error         = nil
		)
		if req.GetBody != nil {
			bodyReader, err = req.GetBody()
			if err != nil {
				return "", err
			}
		} else {
			// body is not nil
			bodyReader = req.Body
		}
		if bodyReader != nil {
			_, err = bodyBuf.ReadFrom(bodyReader)
			if err != nil {
				return "", err
			}
			if req.Body != nil {
				req.Body = io.NopCloser(bytes.NewBuffer(bodyBuf.Bytes()))
			}
			if bodyBuf.Len() > 0 {
				bodyStr := strings.ReplaceAll(bodyBuf.String(), "'", "'\\''")
				curlCmd.WriteString(fmt.Sprintf(" -d '%s'", bodyStr))
			}
		}
	}

	curlCmd.WriteString(" '" + req.URL.String() + "'")

	return curlCmd.String(), nil
}
