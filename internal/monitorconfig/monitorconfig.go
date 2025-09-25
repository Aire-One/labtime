package monitorconfig

import (
	"log"

	"aireone.xyz/labtime/internal/monitors"
	"aireone.xyz/labtime/internal/scheduler"
	"aireone.xyz/labtime/internal/yamlconfig"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

// MonitorSetup defines an interface for setting up monitors.
type MonitorSetup interface {
	Setup(*scheduler.Scheduler, *yamlconfig.YamlConfig, *log.Logger) error
}

// MonitorConfig represents the configuration for a monitor type.
type MonitorConfig[T monitors.Target, C prometheus.Collector] struct {
	Factory  monitors.MonitorFactory[T, C]
	Provider monitors.TargetProvider[T]

	Collector C
}

// NewMonitorConfig creates a new MonitorConfig with the collector already initialized.
func NewMonitorConfig[T monitors.Target, C prometheus.Collector](factory monitors.MonitorFactory[T, C], provider monitors.TargetProvider[T]) *MonitorConfig[T, C] {
	collector := factory.CreateCollector()
	prometheus.MustRegister(collector)
	return &MonitorConfig[T, C]{
		Factory:   factory,
		Provider:  provider,
		Collector: collector,
	}
}

func (mc *MonitorConfig[T, C]) Setup(s *scheduler.Scheduler, config *yamlconfig.YamlConfig, logger *log.Logger) error {
	// Get targets for this monitor type
	targets, err := mc.Provider.GetTargets(config)
	if err != nil {
		return errors.Wrap(err, "error getting targets from configuration")
	}

	// Iterate over the targets
	for _, target := range targets {
		job := mc.Factory.CreateMonitor(target, mc.Collector, logger)
		interval := target.GetInterval()
		if err := s.AddJob(job, interval, scheduler.FileJobTag); err != nil {
			return errors.Wrap(err, "error adding job")
		}
	}

	return nil
}
