package yamlconfig

// HTTPMonitorDTO represents the configuration for HTTP status code monitoring targets.
type HTTPMonitorDTO struct {
	// Name of the target. Used to identify the target from Prometheus. Default is the URL.
	Name string `yaml:"name,omitempty" json:"name,omitempty"`
	// URL of the target. The target should be accessible from the machine running the exporter.
	// The URL should contain the protocol (http:// or https://) and the port if it's not the default one.
	URL string `yaml:"url" json:"url"`
	// Interval to ping the target. Default is 60 seconds.
	Interval int `yaml:"interval,omitempty" json:"interval,omitempty"`
}
