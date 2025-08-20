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
	"golang.org/x/sync/errgroup"
)

type Options struct {
	ConfigFile string
}

type App struct {
	options              Options
	monitorConfigs       MonitorConfigs
	scheduler            *scheduler.Scheduler
	prometheusHTTPServer *http.Server

	logger *log.Logger
}

func NewApp(options Options, logger *log.Logger) (*App, error) {
	monitorConfigs := getMonitorConfigs()

	scheduler, err := scheduler.NewScheduler(logger)
	if err != nil {
		return nil, errors.Wrap(err, "error creating scheduler")
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	server := &http.Server{
		Addr:         ":2112",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
		Handler:      mux,
	}

	return &App{
		options:              options,
		monitorConfigs:       monitorConfigs,
		scheduler:            scheduler,
		prometheusHTTPServer: server,
		logger:               logger,
	}, nil
}

func setupJobsFromFile(configFile string, scheduler *scheduler.Scheduler, monitorConfigs MonitorConfigs, logger *log.Logger) error {
	file, err := os.Open(configFile)
	if err != nil {
		return errors.Wrap(err, "error opening config file")
	}
	defer file.Close()

	config, err := yamlconfig.NewYamlConfig(file)
	if err != nil {
		return errors.Wrap(err, "error creating yaml config")
	}

	for monitorType, monitorConfig := range monitorConfigs {
		if err := monitorConfig.Setup(scheduler, config, logger); err != nil {
			return errors.Wrapf(err, "error setting up %s monitor", monitorType)
		}
	}

	return nil
}

func (a *App) Start(ctx context.Context) error {
	errs, _ := errgroup.WithContext(ctx)

	// Serve Prometheus metrics
	errs.Go(func() error {
		if err := a.prometheusHTTPServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return errors.Wrap(err, "error starting prometheus http server")
		}
		return nil
	})

	// Load YAML configuration
	errs.Go(func() error {
		if err := setupJobsFromFile(a.options.ConfigFile, a.scheduler, a.monitorConfigs, a.logger); err != nil {
			return errors.Wrap(err, "error setting up jobs from file")
		}

		return nil
	})

	return errs.Wait()
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
