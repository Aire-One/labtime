package monitors

import (
	"context"
	"errors"
	"log"
	"strings"
	"testing"

	"aireone.xyz/labtime/internal/yamlconfig"
	"github.com/docker/docker/api/types/container"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// mockDockerClient is a mock implementation of DockerClient for testing.
type mockDockerClient struct {
	containers []container.Summary
	err        error
}

func (m *mockDockerClient) ContainerList(_ context.Context, _ container.ListOptions) ([]container.Summary, error) {
	return m.containers, m.err
}

func TestDockerTarget_GetName(t *testing.T) {
	const expectedName = "test-container"

	target := DockerTarget{
		Name:          expectedName,
		ContainerName: "nginx",
		Interval:      30,
	}

	if got := target.GetName(); got != expectedName {
		t.Errorf("GetName() = %v, want %v", got, expectedName)
	}
}

func TestDockerTarget_GetInterval(t *testing.T) {
	const expectedInterval = 45

	target := DockerTarget{
		Name:          "test-container",
		ContainerName: "nginx",
		Interval:      expectedInterval,
	}

	if got := target.GetInterval(); got != expectedInterval {
		t.Errorf("GetInterval() = %v, want %v", got, expectedInterval)
	}
}

func TestDockerMonitorFactory_CreateCollector(t *testing.T) {
	factory := DockerMonitorFactory{}
	collector := factory.CreateCollector()

	if collector == nil {
		t.Fatal("CreateCollector() returned nil")
	}
}

func TestDockerMonitorFactory_CreateMonitor(t *testing.T) {
	factory := DockerMonitorFactory{}
	target := DockerTarget{
		Name:          "test-container",
		ContainerName: "nginx",
		Interval:      30,
	}
	collector := factory.CreateCollector()
	logger := log.Default()

	monitor := factory.CreateMonitor(target, collector, logger)

	dockerMonitor, ok := monitor.(*DockerMonitor)
	if !ok {
		t.Fatal("CreateMonitor() did not return a *DockerMonitor")
	}

	if dockerMonitor.Label != target.Name {
		t.Errorf("Monitor label = %v, want %v", dockerMonitor.Label, target.Name)
	}

	if dockerMonitor.ContainerName != target.ContainerName {
		t.Errorf("Monitor container name = %v, want %v", dockerMonitor.ContainerName, target.ContainerName)
	}
}

func TestDockerTargetProvider_GetTargets(t *testing.T) {
	tests := []struct {
		name           string
		config         *yamlconfig.YamlConfig
		expectedTarget DockerTarget
	}{
		{
			name: "explicit values - nginx",
			config: &yamlconfig.YamlConfig{
				DockerMonitors: []struct {
					Name          string `yaml:"name,omitempty" json:"name,omitempty"`
					ContainerName string `yaml:"container_name" json:"container_name"`
					Interval      int    `yaml:"interval,omitempty" json:"interval,omitempty"`
				}{
					{Name: "nginx-container", ContainerName: "nginx", Interval: 30},
				},
			},
			expectedTarget: DockerTarget{Name: "nginx-container", ContainerName: "nginx", Interval: 30},
		},
		{
			name: "explicit values - redis",
			config: &yamlconfig.YamlConfig{
				DockerMonitors: []struct {
					Name          string `yaml:"name,omitempty" json:"name,omitempty"`
					ContainerName string `yaml:"container_name" json:"container_name"`
					Interval      int    `yaml:"interval,omitempty" json:"interval,omitempty"`
				}{
					{Name: "redis-cache", ContainerName: "redis", Interval: 60},
				},
			},
			expectedTarget: DockerTarget{Name: "redis-cache", ContainerName: "redis", Interval: 60},
		},
		{
			name: "with defaults - default name and interval",
			config: &yamlconfig.YamlConfig{
				DockerMonitors: []struct {
					Name          string `yaml:"name,omitempty" json:"name,omitempty"`
					ContainerName string `yaml:"container_name" json:"container_name"`
					Interval      int    `yaml:"interval,omitempty" json:"interval,omitempty"`
				}{
					{ContainerName: "postgres"}, // Test defaults
				},
			},
			expectedTarget: DockerTarget{Name: "postgres", ContainerName: "postgres", Interval: 60}, // Default name is ContainerName, default interval is 60
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := DockerTargetProvider{}
			targets := provider.GetTargets(tt.config)

			if len(targets) != 1 {
				t.Fatalf("Expected 1 target, got %d", len(targets))
			}

			actual := targets[0]
			expected := tt.expectedTarget

			if actual.Name != expected.Name {
				t.Errorf("Name: expected %s, got %s", expected.Name, actual.Name)
			}
			if actual.ContainerName != expected.ContainerName {
				t.Errorf("ContainerName: expected %s, got %s", expected.ContainerName, actual.ContainerName)
			}
			if actual.Interval != expected.Interval {
				t.Errorf("Interval: expected %d, got %d", expected.Interval, actual.Interval)
			}
		})
	}
}

func TestDockerMonitor_ID(t *testing.T) {
	const expectedID = "test-monitor"

	monitor := &DockerMonitor{
		Label: expectedID,
	}

	if got := monitor.ID(); got != expectedID {
		t.Errorf("ID() = %v, want %v", got, expectedID)
	}
}

func TestDockerMonitor_Run_ContainerRunning(t *testing.T) {
	// Create a mock client that returns a running container
	mockClient := &mockDockerClient{
		containers: []container.Summary{
			{
				Names: []string{"/nginx"},
				State: "running",
			},
		},
	}

	// Create Prometheus gauge
	gauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "test_docker_container_status",
		Help: "Test metric",
	}, []string{"docker_monitor_name", "container_name"})

	monitor := &DockerMonitor{
		Label:                  "test-container",
		ContainerName:          "nginx",
		Logger:                 log.Default(),
		ContainerStatusMonitor: gauge,
		client:                 mockClient,
	}

	err := monitor.Run(t.Context())
	if err != nil {
		t.Errorf("Run() returned error: %v", err)
	}

	// Verify the metric value was set to 1 for running container
	expectedValue := 1.0
	actualValue := testutil.ToFloat64(gauge.WithLabelValues("test-container", "nginx"))
	if actualValue != expectedValue {
		t.Errorf("Expected metric value %v, got %v", expectedValue, actualValue)
	}
}

func TestDockerMonitor_Run_ContainerNotRunning(t *testing.T) {
	// Create a mock client that returns a stopped container
	mockClient := &mockDockerClient{
		containers: []container.Summary{
			{
				Names: []string{"/nginx"},
				State: "exited",
			},
		},
	}

	// Create Prometheus gauge
	gauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "test_docker_container_status",
		Help: "Test metric",
	}, []string{"docker_monitor_name", "container_name"})

	monitor := &DockerMonitor{
		Label:                  "test-container",
		ContainerName:          "nginx",
		Logger:                 log.Default(),
		ContainerStatusMonitor: gauge,
		client:                 mockClient,
	}

	err := monitor.Run(t.Context())
	if err != nil {
		t.Errorf("Run() returned error: %v", err)
	}

	// Verify the metric value was set to 0 for stopped container
	expectedValue := 0.0
	actualValue := testutil.ToFloat64(gauge.WithLabelValues("test-container", "nginx"))
	if actualValue != expectedValue {
		t.Errorf("Expected metric value %v, got %v", expectedValue, actualValue)
	}
}

func TestDockerMonitor_Run_ContainerNotFound(t *testing.T) {
	// Create a mock client that returns no containers
	mockClient := &mockDockerClient{
		containers: []container.Summary{},
	}

	// Create Prometheus gauge
	gauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "test_docker_container_status",
		Help: "Test metric",
	}, []string{"docker_monitor_name", "container_name"})

	monitor := &DockerMonitor{
		Label:                  "test-container",
		ContainerName:          "nginx",
		Logger:                 log.Default(),
		ContainerStatusMonitor: gauge,
		client:                 mockClient,
	}

	err := monitor.Run(t.Context())
	if err != nil {
		t.Errorf("Run() returned error: %v", err)
	}

	// Verify the metric value was set to 0 for container not found
	expectedValue := 0.0
	actualValue := testutil.ToFloat64(gauge.WithLabelValues("test-container", "nginx"))
	if actualValue != expectedValue {
		t.Errorf("Expected metric value %v, got %v", expectedValue, actualValue)
	}
}

func TestDockerMonitor_Run_NilClient(t *testing.T) {
	// Create Prometheus gauge
	gauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "test_docker_container_status",
		Help: "Test metric",
	}, []string{"docker_monitor_name", "container_name"})

	monitor := &DockerMonitor{
		Label:                  "test-container",
		ContainerName:          "nginx",
		Logger:                 log.Default(),
		ContainerStatusMonitor: gauge,
		client:                 nil, // No client
	}

	err := monitor.Run(t.Context())
	if err == nil {
		t.Error("Run() should return error when client is nil")
	}

	expectedErrMsg := "Docker client not available"
	if !strings.Contains(err.Error(), expectedErrMsg) {
		t.Errorf("Expected error to contain '%s', got: %v", expectedErrMsg, err)
	}
}

func TestDockerMonitor_Run_ClientError(t *testing.T) {
	// Create a mock client that returns an error
	mockClient := &mockDockerClient{
		containers: nil,
		err:        errors.New("failed to connect to Docker daemon"),
	}

	// Create Prometheus gauge
	gauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "test_docker_container_status",
		Help: "Test metric",
	}, []string{"docker_monitor_name", "container_name"})

	monitor := &DockerMonitor{
		Label:                  "test-container",
		ContainerName:          "nginx",
		Logger:                 log.Default(),
		ContainerStatusMonitor: gauge,
		client:                 mockClient,
	}

	err := monitor.Run(t.Context())
	if err == nil {
		t.Error("Run() should return error when Docker client fails")
	}

	expectedErrMsg := "error checking Docker container status"
	if !strings.Contains(err.Error(), expectedErrMsg) {
		t.Errorf("Expected error to contain '%s', got: %v", expectedErrMsg, err)
	}
}

func TestDockerMonitor_Run_ContainerNameWithoutSlash(t *testing.T) {
	// Create a mock client that returns a container with name without leading slash
	mockClient := &mockDockerClient{
		containers: []container.Summary{
			{
				Names: []string{"nginx"}, // No leading slash
				State: "running",
			},
		},
	}

	// Create Prometheus gauge
	gauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "test_docker_container_status",
		Help: "Test metric",
	}, []string{"docker_monitor_name", "container_name"})

	monitor := &DockerMonitor{
		Label:                  "test-container",
		ContainerName:          "nginx",
		Logger:                 log.Default(),
		ContainerStatusMonitor: gauge,
		client:                 mockClient,
	}

	err := monitor.Run(t.Context())
	if err != nil {
		t.Errorf("Run() returned error: %v", err)
	}

	// Verify the metric value was set to 1 for running container
	expectedValue := 1.0
	actualValue := testutil.ToFloat64(gauge.WithLabelValues("test-container", "nginx"))
	if actualValue != expectedValue {
		t.Errorf("Expected metric value %v, got %v", expectedValue, actualValue)
	}
}
