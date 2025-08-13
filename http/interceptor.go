package http

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
