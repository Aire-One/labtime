package yamlconfig

// DockerMonitorDTO represents the configuration for Docker container monitoring targets.
type DockerMonitorDTO struct {
	// Name of the target. Used to identify the target from Prometheus. Default is the container name.
	Name string `yaml:"name,omitempty" json:"name,omitempty"`
	// Container name to monitor. Should match the exact container name in Docker.
	ContainerName string `yaml:"container_name" json:"container_name"`
	// Interval to check the container status. Default is 60 seconds.
	Interval int `yaml:"interval,omitempty" json:"interval,omitempty"`
}
