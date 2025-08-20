package main

import (
	"context"
	"log"
	"os"

	"aireone.xyz/labtime/internal/apps/labtime"
)

const (
	loggerPrefix = "main:"
)

func main() {
	logger := log.New(os.Stdout, loggerPrefix, log.LstdFlags|log.Lshortfile)
	cfg := labtime.LoadFlag(logger)

	app, err := labtime.NewApp(labtime.Options{ConfigFile: *cfg.ConfigFile}, logger)
	if err != nil {
		logger.Fatalf("Error creating app: %v", err)
	}

	if err := app.Start(context.TODO()); err != nil {
		logger.Fatalf("Error starting app: %v", err)
	}
}
