package monitors

import (
	"context"
	"log"
	"net/http"

	aireoneHttp "aireone.xyz/labtime/internal/http"
	"aireone.xyz/labtime/internal/yamlconfig"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

// HTTPTarget represents an HTTP monitoring target.
type HTTPTarget struct {
	Name     string `yaml:"name"`
	URL      string `yaml:"url"`
	Method   string `yaml:"method"`
	Interval int    `yaml:"interval,omitempty"`
}

// GetName implements the Target interface.
func (h HTTPTarget) GetName() string {
	return h.Name
}

// GetInterval implements the Target interface.
func (h HTTPTarget) GetInterval() int {
	return h.Interval
}

// HTTPMonitorFactory implements MonitorFactory for HTTP monitoring.
type HTTPMonitorFactory struct{}

// CreateCollector creates a Prometheus GaugeVec for HTTP monitoring.
func (h HTTPMonitorFactory) CreateCollector() *prometheus.GaugeVec {
	return prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "labtime_http_site_status_code",
		Help: "The status code of the site.",
	}, []string{"http_monitor_site_name", "http_site_url"})
}

// CreateMonitor creates an HTTP monitor instance.
func (h HTTPMonitorFactory) CreateMonitor(target HTTPTarget, collector *prometheus.GaugeVec, logger *log.Logger) Job {
	return &HTTPMonitor{
		Label:                 target.Name,
		URL:                   target.URL,
		Method:                target.Method,
		Logger:                logger,
		SiteStatusCodeMonitor: collector,
		Client: &http.Client{
			Transport: aireoneHttp.NewLoggerMiddleware(logger, http.DefaultTransport),
		},
	}
}

// HTTPTargetProvider implements TargetProvider for HTTP targets.
type HTTPTargetProvider struct{}

// GetTargets extracts HTTP targets from the configuration.
func (h HTTPTargetProvider) GetTargets(config *yamlconfig.YamlConfig) ([]HTTPTarget, error) {
	targets := make([]HTTPTarget, len(config.HTTPStatusCode))
	for i, t := range config.HTTPStatusCode {
		name := t.Name
		if name == "" {
			name = t.URL
		}
		interval := t.Interval
		if interval == 0 {
			interval = 60
		}
		method := t.Method
		if method == "" {
			method = http.MethodHead
		} else if !isValidHTTPMethod(method) {
			return nil, errors.Wrapf(errors.New("invalid HTTP method"), "invalid method '%s' for target '%s'", method, name)
		}
		targets[i] = HTTPTarget{
			Name:     name,
			URL:      t.URL,
			Method:   method,
			Interval: interval,
		}
	}
	return targets, nil
}

// isValidHTTPMethod checks if the provided method is a valid HTTP method.
func isValidHTTPMethod(method string) bool {
	switch method {
	case http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodDelete,
		http.MethodHead,
		http.MethodOptions,
		http.MethodPatch,
		http.MethodConnect,
		http.MethodTrace:
		return true
	default:
		return false
	}
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type HTTPMonitor struct {
	Label  string
	URL    string
	Method string

	Logger *log.Logger

	SiteStatusCodeMonitor *prometheus.GaugeVec

	Client HTTPClient
}

func (h *HTTPMonitor) ID() string {
	return h.Label
}

func (h *HTTPMonitor) Run(ctx context.Context) error {
	d, err := h.httpHealthCheck(ctx)
	if err != nil {
		return errors.Wrap(err, "error running http health check")
	}

	h.pushToPrometheus(d)

	return nil
}

type HTTPHealthCheckerData struct {
	StatusCode int
}

func (h *HTTPMonitor) httpHealthCheck(ctx context.Context) (*HTTPHealthCheckerData, error) {
	req, err := http.NewRequestWithContext(ctx, h.Method, h.URL, http.NoBody)
	if err != nil {
		return nil, errors.Wrap(err, "error creating http request")
	}
	req = req.WithContext(ctx)

	resp, err := h.Client.Do(req)
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
