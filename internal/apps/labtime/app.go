package labtime

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"aireone.xyz/labtime/internal/scheduler"
	"aireone.xyz/labtime/internal/yamlconfig"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type App struct {
	scheduler            *scheduler.Scheduler
	prometheusHTTPServer *http.Server

	logger *log.Logger
}

func NewApp(flag *FlagConfig, logger *log.Logger) (*App, error) {
	config, err := loadYamlConfig(flag.configFile)
	if err != nil {
		return nil, errors.Wrap(err, "error loading yaml config")
	}

	scheduler, err := createScheduler(config, logger)
	if err != nil {
		return nil, errors.Wrap(err, "error creating scheduler")
	}

	return &App{
		logger:    logger,
		scheduler: scheduler,
		prometheusHTTPServer: &http.Server{
			Addr:         ":2112",
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  15 * time.Second,
		},
	}, nil
}

func loadYamlConfig(configFile string) (*yamlconfig.YamlConfig, error) {
	file, err := os.Open(configFile)
	if err != nil {
		return nil, errors.Wrap(err, "error opening config file")
	}

	defer file.Close()

	return yamlconfig.NewYamlConfig(file)
}

func createScheduler(config *yamlconfig.YamlConfig, logger *log.Logger) (*scheduler.Scheduler, error) {
	scheduler, err := scheduler.NewScheduler(logger)
	if err != nil {
		return nil, errors.Wrap(err, "error creating scheduler")
	}

	// Dictionary of monitor configurations
	monitorConfigs := getMonitorConfigs()

	// Setup monitors using the dictionary
	for monitorType, monitorConfig := range monitorConfigs {
		if err := monitorConfig.Setup(scheduler, config, logger); err != nil {
			return nil, errors.Wrapf(err, "error setting up %s monitor", monitorType)
		}
	}

	return scheduler, nil
}

func (a *App) Start() error {
	a.scheduler.Start()

	// Serve Prometheus metrics
	http.Handle("/metrics", promhttp.Handler())
	if err := a.prometheusHTTPServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return errors.Wrap(err, "error starting prometheus http server")
	}

	return nil
}

func (a *App) Shutdown() error {
	if err := a.scheduler.Shutdown(); err != nil {
		return errors.Wrap(err, "error shutting down scheduler")
	}

	if err := a.prometheusHTTPServer.Shutdown(context.TODO()); err != nil {
		return errors.Wrap(err, "error shutting down prometheus http server")
	}

	return nil
}
