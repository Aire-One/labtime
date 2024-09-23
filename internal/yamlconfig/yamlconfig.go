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
}
