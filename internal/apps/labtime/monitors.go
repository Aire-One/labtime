package labtime

import (
	"aireone.xyz/labtime/internal/monitorconfig"
	"aireone.xyz/labtime/internal/monitors"
	"github.com/prometheus/client_golang/prometheus"
)

// getMonitorConfigs returns a map of monitor configurations.
func getMonitorConfigs() map[string]monitorconfig.MonitorSetup {
	return map[string]monitorconfig.MonitorSetup{
		"http": &monitorconfig.MonitorConfig[monitors.HTTPTarget, *prometheus.GaugeVec]{
			Factory:  monitors.HTTPMonitorFactory{},
			Provider: monitors.HTTPTargetProvider{},
		},
		"tls": &monitorconfig.MonitorConfig[monitors.TLSTarget, *prometheus.GaugeVec]{
			Factory:  monitors.TLSMonitorFactory{},
			Provider: monitors.TLSTargetProvider{},
		},
	}
}
