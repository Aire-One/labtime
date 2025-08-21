package labtime

import (
	"flag"
	"log"
	"os"

	"github.com/peterbourgon/ff/v3"
)

const (
	defaultConfigFile = "config.yaml"
)

type Flags struct {
	// Path to the configuration file.
	ConfigFile *string

	// Watch for changes in the configuration file.
	WatchConfigFile *bool
}

func LoadFlag(logger *log.Logger) Flags {
	fs := flag.NewFlagSet("labtime", flag.ContinueOnError)

	cfg := Flags{
		ConfigFile:      fs.String("config", defaultConfigFile, "Path to the configuration file"),
		WatchConfigFile: fs.Bool("watch", false, "Watch for changes in the configuration file"),
	}

	if err := ff.Parse(fs, os.Args[1:], ff.WithEnvVars()); err != nil {
		logger.Fatalf("Error parsing flags: %v", err)
	}

	return cfg
}
