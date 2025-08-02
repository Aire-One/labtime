package monitorconfig

import (
	"bytes"
	"context"
	"log"
	"strings"
	"testing"

	"aireone.xyz/labtime/internal/monitors"
	"aireone.xyz/labtime/internal/scheduler"
	"aireone.xyz/labtime/internal/yamlconfig"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

// Mock implementations for testing

// mockTarget implements the monitors.Target interface.
type mockTarget struct {
	name     string
	interval int
}

func (m mockTarget) GetName() string {
	return m.name
}

func (m mockTarget) GetInterval() int {
	return m.interval
}

// mockJob implements the monitors.Job interface.
type mockJob struct {
	id string
}

func (m mockJob) ID() string {
	return m.id
}

func (m mockJob) Run(_ context.Context) error {
	return nil
}

// mockCollector implements prometheus.Collector interface.
type mockCollector struct {
	desc *prometheus.Desc
}

func (m *mockCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- m.desc
}

func (m *mockCollector) Collect(_ chan<- prometheus.Metric) {
	// No metrics to collect for testing
}

// mockMonitorFactory implements monitors.MonitorFactory.
type mockMonitorFactory struct {
	collector            *mockCollector
	metricName           string
	createCollectorCalls int
	createMonitorCalls   int
}

func (m *mockMonitorFactory) CreateCollector() *mockCollector {
	m.createCollectorCalls++
	if m.collector == nil {
		if m.metricName == "" {
			m.metricName = "test_metric"
		}
		m.collector = &mockCollector{
			desc: prometheus.NewDesc(m.metricName, "Test metric", nil, nil),
		}
	}
	return m.collector
}

func (m *mockMonitorFactory) CreateMonitor(target mockTarget, _ *mockCollector, _ *log.Logger) monitors.Job {
	m.createMonitorCalls++
	return &mockJob{id: target.GetName()}
}

// mockTargetProvider implements monitors.TargetProvider.
type mockTargetProvider struct {
	targets     []mockTarget
	shouldError bool
	errorMsg    string
}

func (m *mockTargetProvider) GetTargets(_ *yamlconfig.YamlConfig) ([]mockTarget, error) {
	if m.shouldError {
		return nil, errors.New(m.errorMsg)
	}
	return m.targets, nil
}

func TestMonitorConfig_Setup_Success(t *testing.T) {
	// Create a mock scheduler
	logger := log.New(bytes.NewBuffer(nil), "test: ", log.LstdFlags)
	mockScheduler, err := scheduler.NewScheduler(logger)
	if err != nil {
		t.Fatalf("Failed to create scheduler: %v", err)
	}
	defer func() {
		if err := mockScheduler.Shutdown(); err != nil {
			t.Logf("Error shutting down scheduler: %v", err)
		}
	}()

	// Create mock targets
	targets := []mockTarget{
		{name: "target1", interval: 30},
		{name: "target2", interval: 60},
	}

	// Create mock providers and factory
	provider := &mockTargetProvider{targets: targets}
	factory := &mockMonitorFactory{metricName: "test_metric_success"}

	// Create monitor config
	config := &MonitorConfig[mockTarget, *mockCollector]{
		Factory:  factory,
		Provider: provider,
	}

	// Create a minimal YAML config
	yamlConfig := &yamlconfig.YamlConfig{}

	// Test Setup
	err = config.Setup(mockScheduler, yamlConfig, logger)
	if err != nil {
		t.Errorf("Setup() failed: %v", err)
	}

	// Verify that the collector was created
	if factory.collector == nil {
		t.Error("Expected collector to be created")
	}

	// Verify that CreateCollector was called exactly once
	if factory.createCollectorCalls != 1 {
		t.Errorf("Expected CreateCollector to be called once, got %d calls", factory.createCollectorCalls)
	}

	// Verify that CreateMonitor was called for each target
	expectedMonitorCalls := len(targets)
	if factory.createMonitorCalls != expectedMonitorCalls {
		t.Errorf("Expected CreateMonitor to be called %d times, got %d calls", expectedMonitorCalls, factory.createMonitorCalls)
	}
}

func TestMonitorConfig_Setup_GetTargetsError(t *testing.T) {
	// Create a mock scheduler
	logger := log.New(bytes.NewBuffer(nil), "test: ", log.LstdFlags)
	mockScheduler, err := scheduler.NewScheduler(logger)
	if err != nil {
		t.Fatalf("Failed to create scheduler: %v", err)
	}
	defer func() {
		if err := mockScheduler.Shutdown(); err != nil {
			t.Logf("Error shutting down scheduler: %v", err)
		}
	}()

	// Create mock provider that returns an error
	provider := &mockTargetProvider{
		shouldError: true,
		errorMsg:    "configuration validation failed",
	}
	factory := &mockMonitorFactory{metricName: "test_metric_error"}

	// Create monitor config
	config := &MonitorConfig[mockTarget, *mockCollector]{
		Factory:  factory,
		Provider: provider,
	}

	// Create a minimal YAML config
	yamlConfig := &yamlconfig.YamlConfig{}

	// Test Setup - should return error
	err = config.Setup(mockScheduler, yamlConfig, logger)
	if err == nil {
		t.Error("Expected Setup() to return an error")
	}

	// Verify error message contains expected context
	expectedError := "error getting targets from configuration"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error to contain '%s', got: %v", expectedError, err)
	}

	// Verify original error is preserved
	if !strings.Contains(err.Error(), "configuration validation failed") {
		t.Errorf("Expected error to contain original error message, got: %v", err)
	}
}

func TestMonitorConfig_Setup_EmptyTargets(t *testing.T) {
	// Create a mock scheduler
	logger := log.New(bytes.NewBuffer(nil), "test: ", log.LstdFlags)
	mockScheduler, err := scheduler.NewScheduler(logger)
	if err != nil {
		t.Fatalf("Failed to create scheduler: %v", err)
	}
	defer func() {
		if err := mockScheduler.Shutdown(); err != nil {
			t.Logf("Error shutting down scheduler: %v", err)
		}
	}()

	// Create mock provider with no targets
	provider := &mockTargetProvider{targets: []mockTarget{}}
	factory := &mockMonitorFactory{metricName: "test_metric_empty"}

	// Create monitor config
	config := &MonitorConfig[mockTarget, *mockCollector]{
		Factory:  factory,
		Provider: provider,
	}

	// Create a minimal YAML config
	yamlConfig := &yamlconfig.YamlConfig{}

	// Test Setup - should succeed with no targets
	err = config.Setup(mockScheduler, yamlConfig, logger)
	if err != nil {
		t.Errorf("Setup() with empty targets failed: %v", err)
	}

	// Verify that the collector was still created and registered
	if factory.collector == nil {
		t.Error("Expected collector to be created even with no targets")
	}

	// Verify that CreateCollector was called exactly once
	if factory.createCollectorCalls != 1 {
		t.Errorf("Expected CreateCollector to be called once, got %d calls", factory.createCollectorCalls)
	}

	// Verify that CreateMonitor was not called (no targets)
	if factory.createMonitorCalls != 0 {
		t.Errorf("Expected CreateMonitor to not be called with no targets, got %d calls", factory.createMonitorCalls)
	}
}

func TestMonitorConfig_Setup_MultipleTargets(t *testing.T) {
	// Create a mock scheduler
	logger := log.New(bytes.NewBuffer(nil), "test: ", log.LstdFlags)
	mockScheduler, err := scheduler.NewScheduler(logger)
	if err != nil {
		t.Fatalf("Failed to create scheduler: %v", err)
	}
	defer func() {
		if err := mockScheduler.Shutdown(); err != nil {
			t.Logf("Error shutting down scheduler: %v", err)
		}
	}()

	// Create multiple mock targets with different intervals
	targets := []mockTarget{
		{name: "target1", interval: 30},
		{name: "target2", interval: 60},
		{name: "target3", interval: 120},
	}

	// Create mock providers and factory
	provider := &mockTargetProvider{targets: targets}
	factory := &mockMonitorFactory{metricName: "test_metric_multiple"}

	// Create monitor config
	config := &MonitorConfig[mockTarget, *mockCollector]{
		Factory:  factory,
		Provider: provider,
	}

	// Create a minimal YAML config
	yamlConfig := &yamlconfig.YamlConfig{}

	// Test Setup
	err = config.Setup(mockScheduler, yamlConfig, logger)
	if err != nil {
		t.Errorf("Setup() with multiple targets failed: %v", err)
	}

	// Verify that the collector was created
	if factory.collector == nil {
		t.Error("Expected collector to be created")
	}

	// Verify that CreateCollector was called exactly once
	if factory.createCollectorCalls != 1 {
		t.Errorf("Expected CreateCollector to be called once, got %d calls", factory.createCollectorCalls)
	}

	// Verify that CreateMonitor was called for each target (3 targets)
	expectedMonitorCalls := 3
	if factory.createMonitorCalls != expectedMonitorCalls {
		t.Errorf("Expected CreateMonitor to be called %d times, got %d calls", expectedMonitorCalls, factory.createMonitorCalls)
	}
}
