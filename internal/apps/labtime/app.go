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

// MonitorSetup defines an interface for setting up monitors.
type MonitorSetup interface {
	Setup(*scheduler.Scheduler, *yamlconfig.YamlConfig, *log.Logger) error
}

// MonitorConfig represents the configuration for a monitor type.
type MonitorConfig[T any, C prometheus.Collector] struct {
	MetricName    string
	MetricHelp    string
	LabelNames    []string
	CreateMonitor func() C
	CreateJob     func(T, C, *log.Logger) monitors.Monitor
	GetTargets    func(*yamlconfig.YamlConfig) []T
	GetInterval   func(T) int
}

// Setup implements MonitorSetup interface.
func (mc *MonitorConfig[T, C]) Setup(scheduler *scheduler.Scheduler, config *yamlconfig.YamlConfig, logger *log.Logger) error {
	// Create Prometheus collector using the CreateMonitor method
	collector := mc.CreateMonitor()
	prometheus.MustRegister(collector)

	// Get targets for this monitor type
	targets := mc.GetTargets(config)

	// Iterate over the targets
	for _, target := range targets {
		job := mc.CreateJob(target, collector, logger)
		interval := mc.GetInterval(target)
		if err := scheduler.AddJob(job, interval); err != nil {
			return errors.Wrap(err, "error adding job")
		}
	}

	return nil
}

// HTTPTarget represents an HTTP monitoring target.
type HTTPTarget struct {
	Name     string `yaml:"name"`
	URL      string `yaml:"url"`
	Interval int    `yaml:"interval,omitempty"`
}

// TLSTarget represents a TLS monitoring target.
type TLSTarget struct {
	Name     string `yaml:"name"`
	Domain   string `yaml:"domain"`
	Interval int    `yaml:"interval,omitempty"`
}

func createScheduler(config *yamlconfig.YamlConfig, logger *log.Logger) (*scheduler.Scheduler, error) {
	scheduler, err := scheduler.NewScheduler(logger)
	if err != nil {
		return nil, errors.Wrap(err, "error creating scheduler")
	}

	// Dictionary of monitor configurations
	monitorConfigs := map[string]MonitorSetup{
		"http": &MonitorConfig[HTTPTarget, *prometheus.GaugeVec]{
			MetricName: "labtime_http_site_status_code",
			MetricHelp: "The status code of the site.",
			LabelNames: []string{"http_monitor_site_name", "http_site_url"},
			CreateMonitor: func() *prometheus.GaugeVec {
				return prometheus.NewGaugeVec(prometheus.GaugeOpts{
					Name: "labtime_http_site_status_code",
					Help: "The status code of the site.",
				}, []string{"http_monitor_site_name", "http_site_url"})
			},
			CreateJob: func(target HTTPTarget, gauge *prometheus.GaugeVec, logger *log.Logger) monitors.Monitor {
				return &monitors.HTTPMonitor{
					Label:                 target.Name,
					URL:                   target.URL,
					Logger:                logger,
					SiteStatusCodeMonitor: gauge,
				}
			},
			GetTargets: func(config *yamlconfig.YamlConfig) []HTTPTarget {
				targets := make([]HTTPTarget, len(config.HTTPStatusCode))
				for i, t := range config.HTTPStatusCode {
					targets[i] = HTTPTarget{
						Name:     t.Name,
						URL:      t.URL,
						Interval: t.Interval,
					}
				}
				return targets
			},
			GetInterval: func(target HTTPTarget) int {
				return target.Interval
			},
		},
		"tls": &MonitorConfig[TLSTarget, *prometheus.GaugeVec]{
			MetricName: "labtime_tls_cert_expire_time",
			MetricHelp: "The duration (in second) until the TLS certificate expires.",
			LabelNames: []string{"tls_monitor_name", "tls_domain_name"},
			CreateMonitor: func() *prometheus.GaugeVec {
				return prometheus.NewGaugeVec(prometheus.GaugeOpts{
					Name: "labtime_tls_cert_expire_time",
					Help: "The duration (in second) until the TLS certificate expires.",
				}, []string{"tls_monitor_name", "tls_domain_name"})
			},
			CreateJob: func(target TLSTarget, gauge *prometheus.GaugeVec, logger *log.Logger) monitors.Monitor {
				return &monitors.TLSMonitor{
					Label:              target.Name,
					Domain:             target.Domain,
					Logger:             logger,
					ExpiresTimeMonitor: gauge,
				}
			},
			GetTargets: func(config *yamlconfig.YamlConfig) []TLSTarget {
				targets := make([]TLSTarget, len(config.TLSMonitors))
				for i, t := range config.TLSMonitors {
					targets[i] = TLSTarget{
						Name:     t.Name,
						Domain:   t.Domain,
						Interval: t.Interval,
					}
				}
				return targets
			},
			GetInterval: func(target TLSTarget) int {
				return target.Interval
			},
		},
	}

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
