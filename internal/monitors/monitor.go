package monitors

import (
	"context"
	"log"

	"aireone.xyz/labtime/internal/yamlconfig"
	"github.com/prometheus/client_golang/prometheus"
)

// Job defines the interface for all monitoring implementations.
type Job interface {
	ID() string
	Run(context.Context) error
}

// Target defines the interface for monitoring targets.
type Target interface {
	GetName() string
	GetInterval() int
}

// MonitorFactory defines the interface for creating monitoring jobs.
type MonitorFactory[T Target, C prometheus.Collector] interface {
	CreateCollector() C
	CreateMonitor(target T, collector C, logger *log.Logger) Job
}

// TargetProvider defines the interface for extracting targets from configuration.
type TargetProvider[T Target] interface {
	GetTargets(config *yamlconfig.YamlConfig) ([]T, error)
}
