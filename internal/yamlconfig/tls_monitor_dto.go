package yamlconfig

// TLSMonitorDTO represents the configuration for TLS certificate monitoring targets.
type TLSMonitorDTO struct {
	// Name of the target. Used to identify the target from Prometheus. Default is the domain name.
	Name string `yaml:"name,omitempty" json:"name,omitempty"`
	// Domain name address of the target. The target should be accessible from the machine running the exporter.
	// The domain should not contain the protocol (http:// or https://) and the port (:443 is mandatory to check the TLS certificate).
	Domain string `yaml:"domain" json:"domain"`
	// Interval to ping the target. Default is 60 seconds.
	Interval int `yaml:"interval,omitempty" json:"interval,omitempty"`
}
