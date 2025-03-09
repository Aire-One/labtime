package labtime

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"aireone.xyz/labtime/internal/monitors"
	"aireone.xyz/labtime/internal/scheduler"
	"aireone.xyz/labtime/internal/yamlconfig"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
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

	// HTTP monitor
	httpMonitor := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "labtime_response_time_duration",
		Help: "The ping time.",
	}, []string{"target_name"})
	prometheus.MustRegister(httpMonitor)

	for _, t := range config.Targets {
		if err := scheduler.AddJob(&monitors.HTTPMonitor{
			Label:               t.Name,
			URL:                 t.URL,
			Logger:              logger,
			ResponseTimeMonitor: httpMonitor,
		}, t.Interval); err != nil {
			return nil, errors.Wrap(err, "error adding job")
		}
	}

	// TLS monitor
	tlsMonitor := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "labtime_tls_cert_expire_time",
		Help: "The duration (in second) until the TLS certificate expires.",
	}, []string{"tls_monitor_name", "tls_domain_name"})
	prometheus.MustRegister(tlsMonitor)

	for _, t := range config.TLSMonitors {
		if err := scheduler.AddJob(&monitors.TLSMonitor{
			Label:              t.Name,
			Domain:             t.Domain,
			Logger:             logger,
			ExpiresTimeMonitor: tlsMonitor,
		}, t.Interval); err != nil {
			return nil, errors.Wrap(err, "error adding job")
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
