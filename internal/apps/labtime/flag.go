package labtime

import "flag"

const (
	defaultConfigFile = "config.yaml"
)

type Flags struct {
	// Path to the configuration file.
	ConfigFile string
}

func LoadFlag() *Flags {
	cfg := Flags{}
	flag.StringVar(&cfg.ConfigFile, "config", defaultConfigFile, "Path to the configuration file")
	flag.Parse()
	return &cfg
}
