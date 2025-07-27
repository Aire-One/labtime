package labtime

import (
	"flag"
	"log"
	"os"
	"testing"
)

func TestLoadFlag(t *testing.T) {
	tests := []struct {
		name         string
		envVars      map[string]string
		args         []string
		expectedFile string
	}{
		{
			name:         "Environment variable set",
			envVars:      map[string]string{"CONFIG": "env-config.yaml"},
			args:         []string{},
			expectedFile: "env-config.yaml",
		},
		{
			name:         "Flag overrides environment variable",
			envVars:      map[string]string{"CONFIG": "env-config.yaml"},
			args:         []string{"-config", "flag-config.yaml"},
			expectedFile: "flag-config.yaml",
		},
		{
			name:         "No env var or flag",
			envVars:      map[string]string{},
			args:         []string{},
			expectedFile: defaultConfigFile,
		},
		{
			name:         "Flag only (no env vars)",
			envVars:      map[string]string{},
			args:         []string{"-config", "flag-only.yaml"},
			expectedFile: "flag-only.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore the original os.Args and environment
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()

			// Make sure we are using a clean flag set
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

			// Set up environment variables (t.Setenv automatically handles cleanup)
			for envVar, envValue := range tt.envVars {
				t.Setenv(envVar, envValue)
			}

			os.Args = append([]string{"cmd"}, tt.args...)

			cfg := LoadFlag(log.Default())

			if *cfg.ConfigFile != tt.expectedFile {
				t.Errorf("got %s, want %s", *cfg.ConfigFile, tt.expectedFile)
			}
		})
	}
}
