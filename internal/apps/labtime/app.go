package labtime

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"aireone.xyz/labtime/internal/dynamicdockermonitoring"
	"aireone.xyz/labtime/internal/monitorconfig"
	"aireone.xyz/labtime/internal/monitors"
	"aireone.xyz/labtime/internal/scheduler"
	"aireone.xyz/labtime/internal/watcher"
	"aireone.xyz/labtime/internal/yamlconfig"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/sync/errgroup"
)

type Options struct {
	ConfigFile              string
	WatchConfigFile         bool
	DynamicDockerMonitoring bool
}

type App struct {
	options              Options
	monitorConfigs       MonitorConfigs
	scheduler            *scheduler.Scheduler
	prometheusHTTPServer *http.Server
	watcher              *watcher.Watcher
	dockerWatcher        *dynamicdockermonitoring.DynamicDockerMonitor

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

	var dockerWatcher *dynamicdockermonitoring.DynamicDockerMonitor
	if options.DynamicDockerMonitoring {
		dockerWatcher, err = dynamicdockermonitoring.NewDynamicDockerMonitor(context.Background())
		if err != nil {
			return nil, errors.Wrap(err, "error creating dynamic docker monitor")
		}
	}

	return &App{
		options:              options,
		monitorConfigs:       monitorConfigs,
		scheduler:            scheduler,
		prometheusHTTPServer: server,
		watcher:              w,
		dockerWatcher:        dockerWatcher,
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

type Container struct {
	ID           string
	Name         string
	Interval     int
	LabtimeLabel bool
}

func setupDynamicDockerMonitoring(container Container, s *scheduler.Scheduler, mc *monitorconfig.MonitorConfig[monitors.DockerTarget, *prometheus.GaugeVec], logger *log.Logger) error {
	if !container.LabtimeLabel {
		return nil
	}

	logger.Printf("New container created: %s, setting up monitoring jobs...", container.ID)

	target := monitors.DockerTarget{
		Name:          container.Name,
		ContainerName: container.Name,
		Interval:      container.Interval,
	}

	job := mc.Factory.CreateMonitor(target, mc.Collector, logger)
	interval := target.GetInterval()
	if err := s.AddJob(job, interval, scheduler.DynamicDockerJobTag); err != nil {
		return errors.Wrap(err, "error adding job for new docker container")
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
			for {
				select {
				case <-derivedCtx.Done():
					if err := shutdownWatcher(a.watcher); err != nil {
						return errors.Wrap(err, "error shutting down watcher")
					}
					return nil

				case err := <-a.watcher.Errors:
					return errors.Wrap(err, "error received from watcher")
				case <-a.watcher.Events:
					a.logger.Println("Configuration file changed, reloading jobs...")

					a.scheduler.RemoveByTag(scheduler.FileJobTag)

					if err := setupJobsFromFile(a.options.ConfigFile, a.scheduler, a.monitorConfigs, a.logger); err != nil {
						a.logger.Printf("Error reloading jobs: %v", err)
					}

				}
			}
		})
	}

	// Enable dynamic Docker monitoring
	if a.options.DynamicDockerMonitoring {
		containers, err := dynamicdockermonitoring.GetRunningContainers(derivedCtx)
		if err != nil {
			return errors.Wrap(err, "error listing running containers for dynamic docker monitoring")
		}

		mc, ok := a.monitorConfigs["docker"].(*monitorconfig.MonitorConfig[monitors.DockerTarget, *prometheus.GaugeVec])
		if !ok {
			panic("docker monitor config not found or wrong type")
		}

		for _, container := range containers {
			interval, err := strconv.Atoi(container.Labels["labtime_interval"])
			if err != nil {
				interval = 60 // default interval
			}
			if err := setupDynamicDockerMonitoring(Container{
				ID:           container.ID,
				Name:         strings.Trim(container.Names[0], "/"),
				Interval:     interval,
				LabtimeLabel: container.Labels["labtime"] == "true",
			}, a.scheduler, mc, a.logger); err != nil {
				a.logger.Printf("Error setting up monitoring for existing docker container: %v", err)
			}
		}

		errs.Go(func() error {
			for {
				select {
				case <-derivedCtx.Done():
					if err := shutdownDynamicDockerMonitor(a.dockerWatcher); err != nil {
						return errors.Wrap(err, "error shutting down dynamic docker monitor")
					}
					return nil

				case err := <-a.dockerWatcher.Errors:
					return errors.Wrap(err, "error received from dynamic docker monitor")
				case event := <-a.dockerWatcher.Events:
					a.logger.Println("Docker event received")

					if event.Action == "create" && event.Type == "container" && event.Actor.Attributes["labtime"] == "true" {
						mc, ok := a.monitorConfigs["docker"].(*monitorconfig.MonitorConfig[monitors.DockerTarget, *prometheus.GaugeVec])
						if !ok {
							panic("docker monitor config not found or wrong type")
						}

						interval, err := strconv.Atoi(event.Actor.Attributes["labtime_interval"])
						if err != nil {
							interval = 60 // default interval
						}

						if err := setupDynamicDockerMonitoring(Container{
							ID:           event.Actor.ID,
							Name:         event.Actor.Attributes["name"],
							Interval:     interval,
							LabtimeLabel: event.Actor.Attributes["labtime"] == "true",
						}, a.scheduler, mc, a.logger); err != nil {
							a.logger.Printf("Error setting up monitoring for new docker container: %v", err)
						}
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

func shutdownDynamicDockerMonitor(d *dynamicdockermonitoring.DynamicDockerMonitor) error {
	if err := d.Shutdown(); err != nil {
		return errors.Wrap(err, "error shutting down dynamic docker monitor")
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

	if err := shutdownDynamicDockerMonitor(a.dockerWatcher); err != nil {
		return errors.Wrap(err, "error shutting down dynamic docker monitor")
	}

	return nil
}
