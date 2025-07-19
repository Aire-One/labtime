package monitors

import (
	"context"
	"log"

	"aireone.xyz/labtime/internal/yamlconfig"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

// DockerTarget represents a Docker container monitoring target.
type DockerTarget struct {
	Name          string `yaml:"name"`
	ContainerName string `yaml:"container_name"`
	Interval      int    `yaml:"interval,omitempty"`
}

// GetName implements the Target interface.
func (d DockerTarget) GetName() string {
	return d.Name
}

// GetInterval implements the Target interface.
func (d DockerTarget) GetInterval() int {
	return d.Interval
}

// DockerMonitorFactory implements MonitorFactory for Docker monitoring.
type DockerMonitorFactory struct{}

// CreateCollector creates a Prometheus GaugeVec for Docker monitoring.
func (d DockerMonitorFactory) CreateCollector() *prometheus.GaugeVec {
	return prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "labtime_docker_container_status",
		Help: "The status of the Docker container (1 = running, 0 = not running).",
	}, []string{"docker_monitor_name", "container_name"})
}

// CreateMonitor creates a Docker monitor instance.
func (d DockerMonitorFactory) CreateMonitor(target DockerTarget, collector *prometheus.GaugeVec, logger *log.Logger) Job {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		logger.Printf("Failed to create Docker client for monitor '%s': %v", target.Name, err)
		// Return a monitor with nil client - it will fail gracefully during Run()
		cli = nil
	}

	return &DockerMonitor{
		Label:                  target.Name,
		ContainerName:          target.ContainerName,
		Logger:                 logger,
		ContainerStatusMonitor: collector,
		client:                 cli,
	}
}

// DockerTargetProvider implements TargetProvider for Docker targets.
type DockerTargetProvider struct{}

// GetTargets extracts Docker targets from the configuration.
func (d DockerTargetProvider) GetTargets(config *yamlconfig.YamlConfig) []DockerTarget {
	targets := make([]DockerTarget, len(config.DockerMonitors))
	for i, monitor := range config.DockerMonitors {
		targets[i] = DockerTarget{
			Name:          monitor.Name,
			ContainerName: monitor.ContainerName,
			Interval:      monitor.Interval,
		}
	}
	return targets
}

// DockerClient interface for testing purposes.
type DockerClient interface {
	ContainerList(ctx context.Context, options container.ListOptions) ([]container.Summary, error)
}

type DockerMonitor struct {
	Label         string
	ContainerName string

	Logger *log.Logger

	ContainerStatusMonitor *prometheus.GaugeVec

	client DockerClient
}

func (d *DockerMonitor) ID() string {
	return d.Label
}

func (d *DockerMonitor) Run(ctx context.Context) error {
	status, err := d.checkContainerStatus(ctx)
	if err != nil {
		return errors.Wrap(err, "error checking Docker container status")
	}

	d.pushToPrometheus(status)

	return nil
}

type DockerHealthCheckerData struct {
	IsRunning bool
}

func (d *DockerMonitor) checkContainerStatus(ctx context.Context) (*DockerHealthCheckerData, error) {
	// Check if Docker client is available
	if d.client == nil {
		return nil, errors.New("Docker client not available")
	}

	// List all containers (including stopped ones)
	containers, err := d.client.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list containers")
	}

	// Search for the target container
	for _, c := range containers {
		for _, name := range c.Names {
			// Container names in Docker API start with '/'
			containerName := name
			if len(name) > 0 && name[0] == '/' {
				containerName = name[1:]
			}

			if containerName == d.ContainerName {
				isRunning := c.State == "running"
				d.Logger.Printf("Container '%s' found with state: %s", d.ContainerName, c.State)
				return &DockerHealthCheckerData{IsRunning: isRunning}, nil
			}
		}
	}

	// Container not found
	d.Logger.Printf("Container '%s' not found", d.ContainerName)
	return &DockerHealthCheckerData{IsRunning: false}, nil
}

func (d *DockerMonitor) pushToPrometheus(data *DockerHealthCheckerData) {
	var statusValue float64
	if data.IsRunning {
		statusValue = 1
	} else {
		statusValue = 0
	}

	d.ContainerStatusMonitor.WithLabelValues(d.Label, d.ContainerName).Set(statusValue)
	d.Logger.Printf("Docker monitor '%s' for container '%s': status = %v", d.Label, d.ContainerName, data.IsRunning)
}
