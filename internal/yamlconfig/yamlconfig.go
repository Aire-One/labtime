package yamlconfig

import (
	"io"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

var ErrYAMLDecode = errors.New("error decoding YAML file")

type YamlConfig struct {
	// List of targets to ping.
	HTTPStatusCode []HTTPMonitorDTO `yaml:"http_status_code" json:"http_status_code"`
	// List of TLS targets to monitor.
	TLSMonitors []TLSMonitorDTO `yaml:"tls_monitors" json:"tls_monitors"`
	// List of Docker containers to monitor.
	DockerMonitors []DockerMonitorDTO `yaml:"docker_monitors" json:"docker_monitors"`
}

func NewYamlConfig(r io.Reader) (*YamlConfig, error) {
	d := yaml.NewDecoder(r)
	config := &YamlConfig{}

	if err := d.Decode(&config); err != nil {
		return nil, errors.Wrap(ErrYAMLDecode, err.Error())
	}

	return config, nil
}
