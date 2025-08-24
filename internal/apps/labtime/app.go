package labtime

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"aireone.xyz/labtime/internal/scheduler"
	"aireone.xyz/labtime/internal/watcher"
	"aireone.xyz/labtime/internal/yamlconfig"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/sync/errgroup"
)

type Options struct {
	ConfigFile      string
	WatchConfigFile bool
}

type App struct {
	options              Options
	monitorConfigs       MonitorConfigs
	scheduler            *scheduler.Scheduler
	prometheusHTTPServer *http.Server
	watcher              *watcher.Watcher

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

	var w *watcher.Watcher
	if options.WatchConfigFile {
		w, err = watcher.NewWatcher(options.ConfigFile)
		if err != nil {
			return nil, errors.Wrap(err, "error creating watcher")
		}
	}

	return &App{
		options:              options,
		monitorConfigs:       monitorConfigs,
		scheduler:            scheduler,
		prometheusHTTPServer: server,
		watcher:              w,
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
	errs, derivedCtx := errgroup.WithContext(ctx)

	// Serve Prometheus metrics
	errs.Go(func() error {
		go func() {
			<-derivedCtx.Done()
			if err := shutdownPrometheusServer(derivedCtx, a.prometheusHTTPServer); err != nil {
				a.logger.Printf("Error shutting down prometheus http server: %v", err)
			}
		}()

		if err := a.prometheusHTTPServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return errors.Wrap(err, "error starting prometheus http server")
		}
		return nil
	})

	// Load YAML configuration
	if err := setupJobsFromFile(a.options.ConfigFile, a.scheduler, a.monitorConfigs, a.logger); err != nil {
		return errors.Wrap(err, "error setting up jobs from file")
	}
	if a.options.WatchConfigFile {
		errs.Go(func() error {
			go func() {
				<-derivedCtx.Done()
				if err := shutdownWatcher(a.watcher); err != nil {
					a.logger.Printf("Error shutting down watcher: %v", err)
				}
			}()

			for {
				select {
				case err := <-a.watcher.Errors:
					return errors.Wrap(err, "error received from watcher")
				case <-a.watcher.Events:
					a.logger.Println("Configuration file changed, reloading jobs...")

					if err := a.scheduler.ClearJobs(); err != nil {
						return errors.Wrap(err, "error clearing jobs")
					}

					if err := setupJobsFromFile(a.options.ConfigFile, a.scheduler, a.monitorConfigs, a.logger); err != nil {
						a.logger.Printf("Error reloading jobs: %v", err)
					}

				}
			}
		})
	}

	return errs.Wait()
}

func shutdownScheduler(scheduler *scheduler.Scheduler) error {
	if err := scheduler.Shutdown(); err != nil {
		return errors.Wrap(err, "error shutting down scheduler")
	}
	return nil
}

func shutdownPrometheusServer(ctx context.Context, server *http.Server) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		return errors.Wrap(err, "error shutting down prometheus http server")
	}

	return nil
}

func shutdownWatcher(watcher *watcher.Watcher) error {
	if err := watcher.Shutdown(); err != nil {
		return errors.Wrap(err, "error shutting down watcher")
	}
	return nil
}

func (a *App) Shutdown(ctx context.Context) error {
	if err := shutdownScheduler(a.scheduler); err != nil {
		return errors.Wrap(err, "error shutting down scheduler")
	}

	if err := shutdownPrometheusServer(ctx, a.prometheusHTTPServer); err != nil {
		return errors.Wrap(err, "error shutting down prometheus http server")
	}

	if err := shutdownWatcher(a.watcher); err != nil {
		return errors.Wrap(err, "error shutting down watcher")
	}

	return nil
}
