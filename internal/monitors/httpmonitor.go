package monitors

import (
	"context"
	"log"
	"net/http"
	"time"

	aireoneHttp "aireone.xyz/labtime/internal/http"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

var ErrInvalidStatusCode = errors.New("expected status code 200")

type HTTPMonitor struct {
	Label string
	URL   string

	Logger *log.Logger

	SiteStatusCodeMonitor *prometheus.GaugeVec
}

func (h *HTTPMonitor) ID() string {
	return h.Label
}

func (h *HTTPMonitor) Run() error {
	d, err := h.httpHealthCheck()
	if err != nil {
		return errors.Wrap(err, "error running http health check")
	}

	h.pushToPrometheus(d)

	return nil
}

func newHTTPDurationMiddleware(duration *time.Duration, proxied http.RoundTripper) *aireoneHttp.RoundTripperMiddleware {
	var t time.Time

	return &aireoneHttp.RoundTripperMiddleware{
		Proxied: proxied,
		OnBefore: func(_ *http.Request) {
			t = time.Now()
		},
		OnAfter: func(_ *http.Response) {
			*duration = time.Since(t)
		},
	}
}

type HTTPHealthCheckerData struct {
	StatusCode int
}

func (h *HTTPMonitor) httpHealthCheck() (*HTTPHealthCheckerData, error) {
	r, err := http.NewRequest(http.MethodHead, h.URL, http.NoBody)
	if err != nil {
		return nil, errors.Wrap(err, "error creating http request")
	}
	req := r.WithContext(context.TODO())

	var duration time.Duration
	client := &http.Client{
		Transport: aireoneHttp.NewLoggerMiddleware(h.Logger, newHTTPDurationMiddleware(&duration, http.DefaultTransport)),
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return &HTTPHealthCheckerData{
		StatusCode: resp.StatusCode,
	}, nil
}

func (h *HTTPMonitor) pushToPrometheus(d *HTTPHealthCheckerData) {
	h.Logger.Printf("HTTP health check for %s: status code %d", h.Label, d.StatusCode)
	h.SiteStatusCodeMonitor.
		With(prometheus.Labels{
			"http_monitor_site_name": h.Label,
			"http_site_url":          h.URL,
		}).
		Set(float64(d.StatusCode))
}
