package labtime

import (
	"flag"
	"log"
	"os"
	"testing"
)

func TestLoadFlag(t *testing.T) {
	tests := []struct {
		name                string
		envVars             map[string]string
		args                []string
		expectedFile        string
		expectedWatch       bool
		expectedWatchDocker bool
	}{
		{
			name:                "Environment variable set",
			envVars:             map[string]string{"CONFIG": "env-config.yaml"},
			args:                []string{},
			expectedFile:        "env-config.yaml",
			expectedWatch:       false,
			expectedWatchDocker: false,
		},
		{
			name:                "Flag overrides environment variable",
			envVars:             map[string]string{"CONFIG": "env-config.yaml"},
			args:                []string{"-config", "flag-config.yaml"},
			expectedFile:        "flag-config.yaml",
			expectedWatch:       false,
			expectedWatchDocker: false,
		},
		{
			name:                "No env var or flag (default values)",
			envVars:             map[string]string{},
			args:                []string{},
			expectedFile:        defaultConfigFile,
			expectedWatch:       false,
			expectedWatchDocker: false,
		},
		{
			name:                "Flag only (no env vars)",
			envVars:             map[string]string{},
			args:                []string{"-config", "flag-only.yaml"},
			expectedFile:        "flag-only.yaml",
			expectedWatch:       false,
			expectedWatchDocker: false,
		},
		{
			name:                "Watch flag enabled",
			envVars:             map[string]string{},
			args:                []string{"-watch"},
			expectedFile:        defaultConfigFile,
			expectedWatch:       true,
			expectedWatchDocker: false,
		},
		{
			name:                "Watch flag with config file",
			envVars:             map[string]string{},
			args:                []string{"-config", "test.yaml", "-watch"},
			expectedFile:        "test.yaml",
			expectedWatch:       true,
			expectedWatchDocker: false,
		},
		{
			name:                "Watch env var enabled",
			envVars:             map[string]string{"WATCH": "true"},
			args:                []string{},
			expectedFile:        defaultConfigFile,
			expectedWatch:       true,
			expectedWatchDocker: false,
		},
		{
			name:                "Flag overrides watch env var",
			envVars:             map[string]string{"WATCH": "true"},
			args:                []string{"-watch=false"},
			expectedFile:        defaultConfigFile,
			expectedWatch:       false,
			expectedWatchDocker: false,
		},
		{
			name:                "All env vars set",
			envVars:             map[string]string{"CONFIG": "env-config.yaml", "WATCH": "true"},
			args:                []string{},
			expectedFile:        "env-config.yaml",
			expectedWatch:       true,
			expectedWatchDocker: false,
		},
		{
			name:                "All flags override all env vars",
			envVars:             map[string]string{"CONFIG": "env-config.yaml", "WATCH": "true"},
			args:                []string{"-config", "flag-config.yaml", "-watch=false"},
			expectedFile:        "flag-config.yaml",
			expectedWatch:       false,
			expectedWatchDocker: false,
		},
		{
			name:                "Long flag names",
			envVars:             map[string]string{},
			args:                []string{"--config", "long-flag.yaml", "--watch"},
			expectedFile:        "long-flag.yaml",
			expectedWatch:       true,
			expectedWatchDocker: false,
		},
		{
			name:                "Mixed short and long flags",
			envVars:             map[string]string{},
			args:                []string{"-config", "mixed.yaml", "--watch"},
			expectedFile:        "mixed.yaml",
			expectedWatch:       true,
			expectedWatchDocker: false,
		},
		{
			name:                "Empty config env var uses default",
			envVars:             map[string]string{"CONFIG": ""},
			args:                []string{},
			expectedFile:        defaultConfigFile,
			expectedWatch:       false,
			expectedWatchDocker: false,
		},
		{
			name:                "Empty watch env var uses default",
			envVars:             map[string]string{"WATCH": ""},
			args:                []string{},
			expectedFile:        defaultConfigFile,
			expectedWatch:       false,
			expectedWatchDocker: false,
		},
		{
			name:                "Empty config flag",
			envVars:             map[string]string{},
			args:                []string{"-config", ""},
			expectedFile:        "",
			expectedWatch:       false,
			expectedWatchDocker: false,
		},
		{
			name:                "Explicitly cleared env vars",
			envVars:             map[string]string{"CONFIG": "", "WATCH": ""},
			args:                []string{},
			expectedFile:        defaultConfigFile,
			expectedWatch:       false,
			expectedWatchDocker: false,
		},
		{
			name:                "Dynamic Docker Monitoring flag enabled",
			envVars:             map[string]string{},
			args:                []string{"-dynamic-docker-monitoring"},
			expectedFile:        defaultConfigFile,
			expectedWatch:       false,
			expectedWatchDocker: true,
		},
		{
			name:                "Dynamic Docker Monitoring flag with config file",
			envVars:             map[string]string{},
			args:                []string{"-config", "test.yaml", "-dynamic-docker-monitoring"},
			expectedFile:        "test.yaml",
			expectedWatch:       false,
			expectedWatchDocker: true,
		},
		{
			name:                "Dynamic Docker Monitoring env var enabled",
			envVars:             map[string]string{"DYNAMIC_DOCKER_MONITORING": "true"},
			args:                []string{},
			expectedFile:        defaultConfigFile,
			expectedWatch:       false,
			expectedWatchDocker: true,
		},
		{
			name:                "Flag overrides Dynamic Docker Monitoring env var",
			envVars:             map[string]string{"DYNAMIC_DOCKER_MONITORING": "true"},
			args:                []string{"-dynamic-docker-monitoring=false"},
			expectedFile:        defaultConfigFile,
			expectedWatch:       false,
			expectedWatchDocker: false,
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

			if *cfg.DynamicDockerMonitoring != tt.expectedWatchDocker {
				t.Errorf("DynamicDockerMonitoring: got %t, want %t", *cfg.DynamicDockerMonitoring, tt.expectedWatchDocker)
			}
		})
	}
}
