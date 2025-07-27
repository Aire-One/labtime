package monitors

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"aireone.xyz/labtime/internal/yamlconfig"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// mockHTTPClient is a mock implementation of HTTPClient for testing.
type mockHTTPClient struct {
	doFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.doFunc(req)
}

// Helper function to create a mock response.
func createMockResponse(statusCode int) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader("")),
		Header:     make(http.Header),
	}
}

func TestHTTPTarget_GetName(t *testing.T) {
	const expectedName = "test-site"

	target := HTTPTarget{
		Name:     expectedName,
		URL:      "https://example.com",
		Interval: 30,
	}

	if got := target.GetName(); got != expectedName {
		t.Errorf("GetName() = %v, want %v", got, expectedName)
	}
}

func TestHTTPTarget_GetInterval(t *testing.T) {
	const expectedInterval = 30

	target := HTTPTarget{
		Name:     "test-site",
		URL:      "https://example.com",
		Interval: expectedInterval,
	}

	if got := target.GetInterval(); got != expectedInterval {
		t.Errorf("GetInterval() = %v, want %v", got, expectedInterval)
	}
}

func TestHTTPMonitorFactory_CreateCollector(t *testing.T) {
	factory := HTTPMonitorFactory{}
	collector := factory.CreateCollector()

	if collector == nil {
		t.Fatal("CreateCollector() returned nil")
	}

	// Test that the collector can be registered (basic smoke test)
	reg := prometheus.NewRegistry()
	err := reg.Register(collector)
	if err != nil {
		t.Fatalf("Failed to register collector: %v", err)
	}

	// Test that the collector accepts the expected label structure without panicking
	gauge := collector.With(prometheus.Labels{
		"http_monitor_site_name": "test-site",
		"http_site_url":          "http://test.com",
	})

	// Verify we can set a value without panicking (smoke test for label compatibility)
	gauge.Set(200)
}

func TestHTTPMonitorFactory_CreateMonitor(t *testing.T) {
	const (
		expectedLabel = "test-site"
		expectedURL   = "https://example.com"
	)

	factory := HTTPMonitorFactory{}
	collector := factory.CreateCollector()
	logger := log.New(bytes.NewBuffer(nil), "", 0)

	target := HTTPTarget{
		Name:     expectedLabel,
		URL:      expectedURL,
		Interval: 30,
	}

	monitor := factory.CreateMonitor(target, collector, logger)

	httpMonitor, ok := monitor.(*HTTPMonitor)
	if !ok {
		t.Fatal("CreateMonitor() did not return an *HTTPMonitor")
	}

	if httpMonitor.Label != expectedLabel {
		t.Errorf("Expected Label '%s', got '%s'", expectedLabel, httpMonitor.Label)
	}

	if httpMonitor.URL != expectedURL {
		t.Errorf("Expected URL '%s', got '%s'", expectedURL, httpMonitor.URL)
	}

	if httpMonitor.Logger != logger {
		t.Error("Logger was not set correctly")
	}

	if httpMonitor.Client == nil {
		t.Error("Client was not set")
	}

	if httpMonitor.SiteStatusCodeMonitor != collector {
		t.Error("SiteStatusCodeMonitor was not set correctly")
	}
}

func TestHTTPTargetProvider_GetTargets(t *testing.T) {
	tests := []struct {
		name           string
		config         *yamlconfig.YamlConfig
		expectedTarget HTTPTarget
	}{
		{
			name: "explicit values - site1",
			config: &yamlconfig.YamlConfig{
				HTTPStatusCode: []struct {
					Name     string `yaml:"name,omitempty" json:"name,omitempty"`
					URL      string `yaml:"url" json:"url"`
					Interval int    `yaml:"interval,omitempty" json:"interval,omitempty"`
				}{
					{Name: "site1", URL: "https://example1.com", Interval: 30},
				},
			},
			expectedTarget: HTTPTarget{Name: "site1", URL: "https://example1.com", Interval: 30},
		},
		{
			name: "explicit values - site2",
			config: &yamlconfig.YamlConfig{
				HTTPStatusCode: []struct {
					Name     string `yaml:"name,omitempty" json:"name,omitempty"`
					URL      string `yaml:"url" json:"url"`
					Interval int    `yaml:"interval,omitempty" json:"interval,omitempty"`
				}{
					{Name: "site2", URL: "https://example2.com", Interval: 60},
				},
			},
			expectedTarget: HTTPTarget{Name: "site2", URL: "https://example2.com", Interval: 60},
		},
		{
			name: "with defaults - default name and interval",
			config: &yamlconfig.YamlConfig{
				HTTPStatusCode: []struct {
					Name     string `yaml:"name,omitempty" json:"name,omitempty"`
					URL      string `yaml:"url" json:"url"`
					Interval int    `yaml:"interval,omitempty" json:"interval,omitempty"`
				}{
					{URL: "https://example.com"}, // Test defaults
				},
			},
			expectedTarget: HTTPTarget{Name: "https://example.com", URL: "https://example.com", Interval: 60}, // Default name is URL, default interval is 60
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := HTTPTargetProvider{}
			targets := provider.GetTargets(tt.config)

			if len(targets) != 1 {
				t.Fatalf("Expected 1 target, got %d", len(targets))
			}

			actual := targets[0]
			expected := tt.expectedTarget

			if actual.Name != expected.Name {
				t.Errorf("Name: expected %s, got %s", expected.Name, actual.Name)
			}
			if actual.URL != expected.URL {
				t.Errorf("URL: expected %s, got %s", expected.URL, actual.URL)
			}
			if actual.Interval != expected.Interval {
				t.Errorf("Interval: expected %d, got %d", expected.Interval, actual.Interval)
			}
		})
	}
}

func TestHTTPMonitor_ID(t *testing.T) {
	const expectedID = "test-site"

	monitor := &HTTPMonitor{
		Label: expectedID,
	}

	actualID := monitor.ID()

	if actualID != expectedID {
		t.Errorf("expected ID to be %s, but got %s", expectedID, actualID)
	}
}

func TestHTTPMonitor_Run(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		expectError  bool
		expectMetric bool
		metricValue  float64
	}{
		{
			name:         "successful request with 200",
			statusCode:   200,
			expectError:  false,
			expectMetric: true,
			metricValue:  200,
		},
		{
			name:         "client error with 404",
			statusCode:   404,
			expectError:  false,
			expectMetric: true,
			metricValue:  404,
		},
		{
			name:         "server error with 500",
			statusCode:   500,
			expectError:  false,
			expectMetric: true,
			metricValue:  500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			const testLabel = "test-site"

			// Create a test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodHead {
					t.Errorf("Expected HEAD request, got %s", r.Method)
				}
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			// Create logger
			var logBuf bytes.Buffer
			logger := log.New(&logBuf, "", 0)

			// Create collector and register it
			collector := prometheus.NewGaugeVec(prometheus.GaugeOpts{
				Name: "labtime_http_site_status_code",
				Help: "The status code of the site.",
			}, []string{"http_monitor_site_name", "http_site_url"})

			reg := prometheus.NewRegistry()
			reg.MustRegister(collector)

			// Create monitor
			monitor := &HTTPMonitor{
				Label:                 testLabel,
				URL:                   server.URL,
				Logger:                logger,
				Client:                &http.Client{}, // Use real client for integration test
				SiteStatusCodeMonitor: collector,
			}

			// Run the monitor
			err := monitor.Run(t.Context())

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if tt.expectMetric {
				// Check that the metric was set correctly
				gauge := collector.With(prometheus.Labels{
					"http_monitor_site_name": testLabel,
					"http_site_url":          server.URL,
				})
				metricValue := testutil.ToFloat64(gauge)

				if metricValue != tt.metricValue {
					t.Errorf("Expected metric value %f, got %f", tt.metricValue, metricValue)
				}

				// Check that the log message was written
				logOutput := logBuf.String()
				expectedLog := fmt.Sprintf("HTTP health check for %s: status code %d", testLabel, tt.statusCode)
				if logOutput == "" {
					t.Error("Expected log output but got none")
				} else if !bytes.Contains(logBuf.Bytes(), []byte(expectedLog)) {
					t.Errorf("Expected log to contain '%s', got '%s'", expectedLog, logOutput)
				}
			}
		})
	}
}

func TestHTTPMonitor_Run_NetworkError(t *testing.T) {
	const testLabel = "test-site"

	// Create logger
	var logBuf bytes.Buffer
	logger := log.New(&logBuf, "", 0)

	// Create collector
	collector := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "labtime_http_site_status_code",
		Help: "The status code of the site.",
	}, []string{"http_monitor_site_name", "http_site_url"})

	// Create mock HTTP client that simulates network error
	mockClient := &mockHTTPClient{
		doFunc: func(_ *http.Request) (*http.Response, error) {
			return nil, fmt.Errorf("network error: connection refused")
		},
	}

	// Create monitor with mock client
	monitor := &HTTPMonitor{
		Label:                 testLabel,
		URL:                   "http://test.example.com",
		Logger:                logger,
		Client:                mockClient,
		SiteStatusCodeMonitor: collector,
	}

	// Run the monitor
	err := monitor.Run(t.Context())

	if err == nil {
		t.Error("Expected network error but got none")
	}

	// Error should be wrapped with context
	if err.Error() == "" {
		t.Error("Expected error message but got empty string")
	}

	// Verify the error contains our mock error message
	if !strings.Contains(err.Error(), "network error") {
		t.Errorf("Expected error to contain 'network error', got: %s", err.Error())
	}
}

func TestHTTPMonitor_httpHealthCheck(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{"status 200", 200},
		{"status 404", 404},
		{"status 500", 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock HTTP client
			mockClient := &mockHTTPClient{
				doFunc: func(_ *http.Request) (*http.Response, error) {
					return createMockResponse(tt.statusCode), nil
				},
			}

			monitor := &HTTPMonitor{
				Label:  "test",
				URL:    "http://test.example.com",
				Logger: log.New(bytes.NewBuffer(nil), "", 0),
				Client: mockClient,
			}

			data, err := monitor.httpHealthCheck(t.Context())
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if data.StatusCode != tt.statusCode {
				t.Errorf("Expected status code %d, got %d", tt.statusCode, data.StatusCode)
			}
		})
	}
}

func TestHTTPMonitor_pushToPrometheus(t *testing.T) {
	const (
		testLabel       = "test-site"
		testURL         = "https://example.com"
		testStatusCode  = 200
		expectedLogLine = "HTTP health check for test-site: status code 200\n"
	)

	var logBuf bytes.Buffer
	logger := log.New(&logBuf, "", 0)

	collector := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "labtime_http_site_status_code",
		Help: "The status code of the site.",
	}, []string{"http_monitor_site_name", "http_site_url"})

	reg := prometheus.NewRegistry()
	reg.MustRegister(collector)

	monitor := &HTTPMonitor{
		Label:                 testLabel,
		URL:                   testURL,
		Logger:                logger,
		Client:                nil,
		SiteStatusCodeMonitor: collector,
	}

	data := &HTTPHealthCheckerData{
		StatusCode: testStatusCode,
	}

	monitor.pushToPrometheus(data)

	// Check metric value
	gauge := collector.With(prometheus.Labels{
		"http_monitor_site_name": testLabel,
		"http_site_url":          testURL,
	})
	metricValue := testutil.ToFloat64(gauge)

	if metricValue != testStatusCode {
		t.Errorf("Expected metric value %d, got %f", testStatusCode, metricValue)
	}

	// Check log output
	logOutput := logBuf.String()
	if logOutput != expectedLogLine {
		t.Errorf("Expected log '%s', got '%s'", expectedLogLine, logOutput)
	}
}
