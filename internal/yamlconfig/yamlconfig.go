package yamlconfig

import (
	"io"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

var ErrYAMLDecode = errors.New("error decoding YAML file")

type YamlConfig struct {
	// List of targets to ping.
	Targets []struct {
		// Name of the target. Used to identify the target from Prometheus.
		Name string `yaml:"name"`
		// URL of the target. The target should be accessible from the machine running the exporter.
		// The URL should contain the protocol (http:// or https://) and the port if it's not the default one.
		URL string `yaml:"url"`
		// Interval to ping the target. Default is 5 seconds.
		Interval int `yaml:"interval,omitempty"`
	} `yaml:"targets"`
	// List of TLS targets to monitor.
	TLSMonitors []struct {
		// Name of the target. Used to identify the target from Prometheus. Default is the domain name.
		Name string `yaml:"name"`
		// Domain name address of the target. The target should be accessible from the machine running the exporter.
		// The domain should not contain the protocol (http:// or https://) and the port (:443 is mandatory to check the TLS certificate).
		Domain string `yaml:"domain"`
		// Interval to ping the target. Default is 36000 seconds (once a day).
		Interval int `yaml:"interval,omitempty"`
	} `yaml:"tls_monitors"`
}

func NewYamlConfig(r io.Reader) (*YamlConfig, error) {
	d := yaml.NewDecoder(r)
	config := &YamlConfig{}

	if err := d.Decode(&config); err != nil {
		return nil, errors.Wrap(ErrYAMLDecode, err.Error())
	}

	applyDefault(config)

	return config, nil
}

func applyDefault(config *YamlConfig) {
	// Set default interval to 5 seconds if not provided
	for i := range config.Targets {
		if config.Targets[i].Interval == 0 {
			config.Targets[i].Interval = 5
		}
	}

	// Set default interval to 5 seconds if not provided
	for i := range config.TLSMonitors {
		if config.TLSMonitors[i].Name == "" {
			config.TLSMonitors[i].Name = config.TLSMonitors[i].Domain
		}
		if config.TLSMonitors[i].Interval == 0 {
			config.TLSMonitors[i].Interval = 36000
		}
	}
}
