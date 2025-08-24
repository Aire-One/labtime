package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"aireone.xyz/labtime/internal/apps/labtime"
)

const (
	loggerPrefix = "main:"
)

func main() {
	logger := log.New(os.Stdout, loggerPrefix, log.LstdFlags|log.Lshortfile)
	cfg := labtime.LoadFlag(logger)

	app, err := labtime.NewApp(labtime.Options{
		ConfigFile:      *cfg.ConfigFile,
		WatchConfigFile: *cfg.WatchConfigFile,
	}, logger)
	if err != nil {
		logger.Fatalf("Error creating app: %v", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start app async since app.Start() is blocking and we need to handle
	// signals concurrently.
	go func() {
		if err := app.Start(context.Background()); err != nil {
			logger.Printf("Error starting app: %v", err)
			sigChan <- syscall.SIGTERM
		}
	}()

	sig := <-sigChan
	logger.Printf("Received signal %v, initiating graceful shutdown...", sig)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	// Run shutdown with timeout protection
	shutdownComplete := make(chan error, 1)
	go func() {
		shutdownComplete <- app.Shutdown(shutdownCtx)
	}()

	var exitCode int
	select {
	case err := <-shutdownComplete:
		if err != nil {
			logger.Printf("Error during shutdown: %v", err)
			exitCode = 1
		} else {
			logger.Println("Shutdown completed successfully")
			exitCode = 0
		}
	case <-shutdownCtx.Done():
		logger.Println("Shutdown timeout exceeded")
		exitCode = 1
	}

	cancel()
	os.Exit(exitCode)
}
