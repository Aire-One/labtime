package monitors

import (
	"bytes"
	"crypto/tls"
	"errors"
	"log"
	"testing"
	"time"

	"aireone.xyz/labtime/internal/yamlconfig"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestTLSTarget_GetName(t *testing.T) {
	const expectedName = "test-domain"

	target := TLSTarget{
		Name:     expectedName,
		Domain:   "example.com",
		Interval: 30,
	}

	if got := target.GetName(); got != expectedName {
		t.Errorf("GetName() = %v, want %v", got, expectedName)
	}
}

func TestTLSTarget_GetInterval(t *testing.T) {
	const expectedInterval = 30

	target := TLSTarget{
		Name:     "test-domain",
		Domain:   "example.com",
		Interval: expectedInterval,
	}

	if got := target.GetInterval(); got != expectedInterval {
		t.Errorf("GetInterval() = %v, want %v", got, expectedInterval)
	}
}

func TestTLSMonitorFactory_CreateCollector(t *testing.T) {
	factory := TLSMonitorFactory{}
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
		"tls_monitor_name": "test-domain",
		"tls_domain_name":  "example.com",
	})

	// Verify we can set a value without panicking (smoke test for label compatibility)
	gauge.Set(86400) // 1 day in seconds
}

func TestTLSMonitorFactory_CreateMonitor(t *testing.T) {
	const (
		expectedLabel  = "test-domain"
		expectedDomain = "example.com"
	)

	factory := TLSMonitorFactory{}
	collector := factory.CreateCollector()
	logger := log.New(bytes.NewBuffer(nil), "", 0)

	target := TLSTarget{
		Name:     expectedLabel,
		Domain:   expectedDomain,
		Interval: 30,
	}

	monitor := factory.CreateMonitor(target, collector, logger)

	tlsMonitor, ok := monitor.(*TLSMonitor)
	if !ok {
		t.Fatal("CreateMonitor() did not return an *TLSMonitor")
	}

	if tlsMonitor.Label != expectedLabel {
		t.Errorf("Expected Label '%s', got '%s'", expectedLabel, tlsMonitor.Label)
	}

	if tlsMonitor.Domain != expectedDomain {
		t.Errorf("Expected Domain '%s', got '%s'", expectedDomain, tlsMonitor.Domain)
	}

	if tlsMonitor.Logger != logger {
		t.Error("Logger was not set correctly")
	}

	if tlsMonitor.ExpiresTimeMonitor != collector {
		t.Error("ExpiresTimeMonitor was not set correctly")
	}
}

func TestTLSTargetProvider_GetTargets(t *testing.T) {
	config := &yamlconfig.YamlConfig{
		TLSMonitors: []struct {
			Name     string `yaml:"name,omitempty" json:"name,omitempty"`
			Domain   string `yaml:"domain" json:"domain"`
			Interval int    `yaml:"interval,omitempty" json:"interval,omitempty"`
		}{
			{Name: "domain1", Domain: "example1.com", Interval: 30},
			{Name: "domain2", Domain: "example2.com", Interval: 60},
		},
	}
	expectedTargets := config.TLSMonitors

	provider := TLSTargetProvider{}
	targets := provider.GetTargets(config)

	expectedTargetCount := len(expectedTargets)
	if len(targets) != expectedTargetCount {
		t.Fatalf("Expected %d targets, got %d", expectedTargetCount, len(targets))
	}

	for i, expected := range expectedTargets {
		target := targets[i]
		if target.Name != expected.Name {
			t.Errorf("Target %d Name: expected %s, got %s", i, expected.Name, target.Name)
		}
		if target.Domain != expected.Domain {
			t.Errorf("Target %d Domain: expected %s, got %s", i, expected.Domain, target.Domain)
		}
		if target.Interval != expected.Interval {
			t.Errorf("Target %d Interval: expected %d, got %d", i, expected.Interval, target.Interval)
		}
	}
}

func TestTLSMonitor_ID(t *testing.T) {
	const expectedID = "test-domain"

	monitor := &TLSMonitor{
		Label: expectedID,
	}

	actualID := monitor.ID()

	if actualID != expectedID {
		t.Errorf("expected ID to be %s, but got %s", expectedID, actualID)
	}
}

func TestTLSMonitor_tlsHandshake_DialError(t *testing.T) {
	var logBuf bytes.Buffer
	logger := log.New(&logBuf, "", 0)

	monitor := &TLSMonitor{
		Label:  "test-domain",
		Domain: "example.com",
		Logger: logger,
		DialFunc: func(_, _ string, _ *tls.Config) (*tls.Conn, error) {
			return nil, errors.New("connection failed")
		},
	}

	data, err := monitor.tlsHandshake()

	if err == nil {
		t.Error("Expected error but got none")
	}

	if data != nil {
		t.Error("Expected nil data on error")
	}
}

func TestTLSMonitor_pushToPrometheus(t *testing.T) {
	const testLabel = "test-domain"
	const testDomain = "example.com"

	// Create collector and register it
	collector := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "labtime_tls_cert_expire_time",
		Help: "The duration (in second) until the TLS certificate expires.",
	}, []string{"tls_monitor_name", "tls_domain_name"})

	reg := prometheus.NewRegistry()
	reg.MustRegister(collector)

	var logBuf bytes.Buffer
	logger := log.New(&logBuf, "", 0)

	monitor := &TLSMonitor{
		Label:              testLabel,
		Domain:             testDomain,
		Logger:             logger,
		ExpiresTimeMonitor: collector,
	}

	// Test with certificate expiring in 1 day
	futureTime := time.Now().Add(24 * time.Hour)
	data := &TLSHealthCheckerData{
		Expires: futureTime,
	}

	monitor.pushToPrometheus(data)

	// Verify the metric was set correctly
	expectedValue := time.Until(futureTime).Seconds()

	// Allow for small timing differences (within 1 second)
	metricValue := testutil.ToFloat64(collector.With(prometheus.Labels{
		"tls_monitor_name": testLabel,
		"tls_domain_name":  testDomain,
	}))

	if abs(metricValue-expectedValue) > 1.0 {
		t.Errorf("Expected metric value around %f, got %f", expectedValue, metricValue)
	}

	// Verify log output contains expected information
	if !bytes.Contains(logBuf.Bytes(), []byte(testLabel)) {
		t.Error("Log output should contain monitor label")
	}
	if !bytes.Contains(logBuf.Bytes(), []byte("expires in")) {
		t.Error("Log output should contain expiration information")
	}
}

// Helper function to calculate absolute value of float64.
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
