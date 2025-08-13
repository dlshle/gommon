package http

import (
	"context"
	"net/http"
	"time"

	"github.com/dlshle/gommon/utils"
)

// Trackable Request
type trackableRequest struct {
	id         string
	cancelFunc func()
	request    *http.Request
	response   *awaitableResponse
}

type TrackableRequest interface {
	ID() string
	WaitAndGetResponse() (*Response, error)
}

func newTrackableRequest(request *http.Request) *trackableRequest {
	var (
		ctx        context.Context
		cancelFunc func()
	)
	if timeoutVal := request.Context().Value("timeout"); timeoutVal != nil {
		ctx, cancelFunc = context.WithTimeout(request.Context(), timeoutVal.(time.Duration))
	} else {
		ctx, cancelFunc = context.WithCancel(request.Context())
	}
	request = request.WithContext(ctx)
	id := utils.RandomStringWithSize(12)
	return &trackableRequest{id, cancelFunc, request, newAwaitableResponse()}
}

func (tr *trackableRequest) ID() string {
	return tr.id
}

func (tr *trackableRequest) complete() {
	// invoke cancel func to relase timeout context timer
	tr.cancelFunc()
	tr.cancelFunc = nil
}

func (tr *trackableRequest) getRequest() *http.Request {
	return tr.request
}

func (tr *trackableRequest) WaitAndGetResponse() (*Response, error) {
	return tr.response.Get()
}
