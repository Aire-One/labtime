package labtime

import (
	"aireone.xyz/labtime/internal/monitorconfig"
	"aireone.xyz/labtime/internal/monitors"
)

type MonitorConfigs = map[string]monitorconfig.MonitorSetup

// getMonitorConfigs returns a map of monitor configurations.
func getMonitorConfigs() MonitorConfigs {
	return MonitorConfigs{
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
