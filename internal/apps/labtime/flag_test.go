package labtime

import (
	"flag"
	"log"
	"os"
	"testing"
)

func TestLoadFlag(t *testing.T) {
	tests := []struct {
		name          string
		envVars       map[string]string
		args          []string
		expectedFile  string
		expectedWatch bool
	}{
		{
			name:          "Environment variable set",
			envVars:       map[string]string{"CONFIG": "env-config.yaml"},
			args:          []string{},
			expectedFile:  "env-config.yaml",
			expectedWatch: false,
		},
		{
			name:          "Flag overrides environment variable",
			envVars:       map[string]string{"CONFIG": "env-config.yaml"},
			args:          []string{"-config", "flag-config.yaml"},
			expectedFile:  "flag-config.yaml",
			expectedWatch: false,
		},
		{
			name:          "No env var or flag (default values)",
			envVars:       map[string]string{},
			args:          []string{},
			expectedFile:  defaultConfigFile,
			expectedWatch: false,
		},
		{
			name:          "Flag only (no env vars)",
			envVars:       map[string]string{},
			args:          []string{"-config", "flag-only.yaml"},
			expectedFile:  "flag-only.yaml",
			expectedWatch: false,
		},
		{
			name:          "Watch flag enabled",
			envVars:       map[string]string{},
			args:          []string{"-watch"},
			expectedFile:  defaultConfigFile,
			expectedWatch: true,
		},
		{
			name:          "Watch flag with config file",
			envVars:       map[string]string{},
			args:          []string{"-config", "test.yaml", "-watch"},
			expectedFile:  "test.yaml",
			expectedWatch: true,
		},
		{
			name:          "Watch env var enabled",
			envVars:       map[string]string{"WATCH": "true"},
			args:          []string{},
			expectedFile:  defaultConfigFile,
			expectedWatch: true,
		},
		{
			name:          "Flag overrides watch env var",
			envVars:       map[string]string{"WATCH": "true"},
			args:          []string{"-watch=false"},
			expectedFile:  defaultConfigFile,
			expectedWatch: false,
		},
		{
			name:          "All env vars set",
			envVars:       map[string]string{"CONFIG": "env-config.yaml", "WATCH": "true"},
			args:          []string{},
			expectedFile:  "env-config.yaml",
			expectedWatch: true,
		},
		{
			name:          "All flags override all env vars",
			envVars:       map[string]string{"CONFIG": "env-config.yaml", "WATCH": "true"},
			args:          []string{"-config", "flag-config.yaml", "-watch=false"},
			expectedFile:  "flag-config.yaml",
			expectedWatch: false,
		},
		{
			name:          "Long flag names",
			envVars:       map[string]string{},
			args:          []string{"--config", "long-flag.yaml", "--watch"},
			expectedFile:  "long-flag.yaml",
			expectedWatch: true,
		},
		{
			name:          "Mixed short and long flags",
			envVars:       map[string]string{},
			args:          []string{"-config", "mixed.yaml", "--watch"},
			expectedFile:  "mixed.yaml",
			expectedWatch: true,
		},
		{
			name:          "Empty config env var uses default",
			envVars:       map[string]string{"CONFIG": ""},
			args:          []string{},
			expectedFile:  defaultConfigFile,
			expectedWatch: false,
		},
		{
			name:          "Empty watch env var uses default",
			envVars:       map[string]string{"WATCH": ""},
			args:          []string{},
			expectedFile:  defaultConfigFile,
			expectedWatch: false,
		},
		{
			name:          "Empty config flag",
			envVars:       map[string]string{},
			args:          []string{"-config", ""},
			expectedFile:  "",
			expectedWatch: false,
		},
		{
			name:          "Explicitly cleared env vars",
			envVars:       map[string]string{"CONFIG": "", "WATCH": ""},
			args:          []string{},
			expectedFile:  defaultConfigFile,
			expectedWatch: false,
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
				t.Errorf("ConfigFile: got %s, want %s", *cfg.ConfigFile, tt.expectedFile)
			}

			if *cfg.WatchConfigFile != tt.expectedWatch {
				t.Errorf("WatchConfigFile: got %t, want %t", *cfg.WatchConfigFile, tt.expectedWatch)
			}
		})
	}
}
