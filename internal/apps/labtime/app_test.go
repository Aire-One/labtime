package labtime

import (
	"log"
	"os"
	"testing"
)

func TestNewApp(t *testing.T) {
	flag := &FlagConfig{
		configFile: "../../../configs/example-config.yaml", // we shouldn't rely on the actual file
	}
	logger := log.New(os.Stdout, "", 0) // silent?

	app, err := NewApp(flag, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if app.logger != logger {
		t.Errorf("expected logger to be set")
	}

	if app.scheduler == nil {
		t.Errorf("expected scheduler to be initialized")
	}

	if app.prometheusHTTPServer == nil {
		t.Errorf("expected prometheusHTTPServer to be initialized")
	}
}
