package labtime

import (
	"flag"
	"os"
	"testing"
)

func TestLoadFlag(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		expectedFile string
	}{
		{
			name:         "No flag provided",
			args:         []string{},
			expectedFile: defaultConfigFile,
		},
		{
			name:         "Flag provided",
			args:         []string{"-config", "custom.yaml"},
			expectedFile: "custom.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore the original os.Args
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()

			// Make sure we are using a clean flag set
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

			os.Args = append([]string{"cmd"}, tt.args...)

			cfg := LoadFlag()

			if cfg.ConfigFile != tt.expectedFile {
				t.Errorf("got %s, want %s", cfg.ConfigFile, tt.expectedFile)
			}
		})
	}
}
