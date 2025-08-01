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
}

// Setup implements MonitorSetup interface.
func (mc *MonitorConfig[T, C]) Setup(scheduler *scheduler.Scheduler, config *yamlconfig.YamlConfig, logger *log.Logger) error {
	// Create Prometheus collector using the factory
	collector := mc.Factory.CreateCollector()
	prometheus.MustRegister(collector)

	// Get targets for this monitor type
	targets, err := mc.Provider.GetTargets(config)
	if err != nil {
		return errors.Wrap(err, "error getting targets from configuration")
	}

	// Iterate over the targets
	for _, target := range targets {
		job := mc.Factory.CreateMonitor(target, collector, logger)
		interval := target.GetInterval()
		if err := scheduler.AddJob(job, interval); err != nil {
			return errors.Wrap(err, "error adding job")
		}
	}

	return nil
}
