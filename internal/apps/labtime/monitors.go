package labtime

import (
	"aireone.xyz/labtime/internal/monitorconfig"
	"aireone.xyz/labtime/internal/monitors"
)

// getMonitorConfigs returns a map of monitor configurations.
func getMonitorConfigs() map[string]monitorconfig.MonitorSetup {
	return map[string]monitorconfig.MonitorSetup{
		"http": monitorconfig.NewMonitorConfig(
			monitors.HTTPMonitorFactory{},
			monitors.HTTPTargetProvider{},
		),
		"tls": monitorconfig.NewMonitorConfig(
			monitors.TLSMonitorFactory{},
			monitors.TLSTargetProvider{},
		),
		"docker": monitorconfig.NewMonitorConfig(
			monitors.DockerMonitorFactory{},
			monitors.DockerTargetProvider{},
		),
	}
}
