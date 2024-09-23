package labtime

import "flag"

const (
	defaultConfigFile = "config.yaml"
)

type FlagConfig struct {
	configFile string
}

func LoadFlag() *FlagConfig {
	cfg := FlagConfig{}
	flag.StringVar(&cfg.configFile, "config", defaultConfigFile, "Path to the configuration file")
	flag.Parse()
	return &cfg
}
