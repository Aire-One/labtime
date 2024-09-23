package http

import (
	"log"
	"net/http"
)

type RoundTripperMiddleware struct {
	Proxied http.RoundTripper

	OnBefore func(req *http.Request)
	OnAfter  func(res *http.Response)
}

func (m RoundTripperMiddleware) RoundTrip(req *http.Request) (res *http.Response, err error) {
	m.OnBefore(req)
	res, err = m.Proxied.RoundTrip(req)
	m.OnAfter(res)

	return res, err
}

func NewLoggerMiddleware(logger *log.Logger, proxied http.RoundTripper) *RoundTripperMiddleware {
	return &RoundTripperMiddleware{
		Proxied: proxied,
		OnBefore: func(req *http.Request) {
			logger.Printf("Request: %s %s", req.Method, req.URL.String())
		},
		OnAfter: func(res *http.Response) {
			logger.Printf("Response: %d", res.StatusCode)
		},
	}
}
