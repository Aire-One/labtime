package monitors

import (
	"context"
	"crypto/tls"
	"log"
	"time"

	"aireone.xyz/labtime/internal/yamlconfig"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

type TLSDialFunc func(network, addr string, config *tls.Config) (*tls.Conn, error)

// TLSTarget represents a TLS monitoring target.
type TLSTarget struct {
	Name     string `yaml:"name"`
	Domain   string `yaml:"domain"`
	Interval int    `yaml:"interval,omitempty"`
}

// GetName implements the Target interface.
func (t TLSTarget) GetName() string {
	return t.Name
}

// GetInterval implements the Target interface.
func (t TLSTarget) GetInterval() int {
	return t.Interval
}

// TLSMonitorFactory implements MonitorFactory for TLS monitoring.
type TLSMonitorFactory struct{}

// CreateCollector creates a Prometheus GaugeVec for TLS monitoring.
func (t TLSMonitorFactory) CreateCollector() *prometheus.GaugeVec {
	return prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "labtime_tls_cert_expire_time",
		Help: "The duration (in second) until the TLS certificate expires.",
	}, []string{"tls_monitor_name", "tls_domain_name"})
}

// CreateMonitor creates a TLS monitor instance.
func (t TLSMonitorFactory) CreateMonitor(target TLSTarget, collector *prometheus.GaugeVec, logger *log.Logger) Job {
	return &TLSMonitor{
		Label:              target.Name,
		Domain:             target.Domain,
		Logger:             logger,
		ExpiresTimeMonitor: collector,
		DialFunc:           tls.Dial,
	}
}

// TLSTargetProvider implements TargetProvider for TLS targets.
type TLSTargetProvider struct{}

// GetTargets extracts TLS targets from the configuration.
func (t TLSTargetProvider) GetTargets(config *yamlconfig.YamlConfig) []TLSTarget {
	targets := make([]TLSTarget, len(config.TLSMonitors))
	for i, monitor := range config.TLSMonitors {
		name := monitor.Name
		if name == "" {
			name = monitor.Domain
		}
		interval := monitor.Interval
		if interval == 0 {
			interval = 60
		}
		targets[i] = TLSTarget{
			Name:     name,
			Domain:   monitor.Domain,
			Interval: interval,
		}
	}
	return targets
}

type TLSMonitor struct {
	Label  string
	Domain string

	Logger *log.Logger

	ExpiresTimeMonitor *prometheus.GaugeVec

	DialFunc TLSDialFunc
}

func (t *TLSMonitor) ID() string {
	return t.Label
}

func (t *TLSMonitor) Run(_ context.Context) error {
	d, err := t.tlsHandshake()
	if err != nil {
		return errors.Wrap(err, "error running tls handshake")
	}

	t.pushToPrometheus(d)

	return nil
}

type TLSHealthCheckerData struct {
	Expires time.Time
}

func (t *TLSMonitor) tlsHandshake() (*TLSHealthCheckerData, error) {
	conn, err := t.DialFunc("tcp", t.Domain+":443", nil)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	expires := conn.ConnectionState().PeerCertificates[0].NotAfter
	t.Logger.Printf("TLS certificate expires on %s", expires)

	return &TLSHealthCheckerData{
		Expires: expires,
	}, nil
}

func (t *TLSMonitor) pushToPrometheus(d *TLSHealthCheckerData) {
	remainingTime := time.Until(d.Expires).Seconds()
	t.Logger.Printf("TLS certificate for monitor %s expires in %f seconds", t.Label, remainingTime)

	t.ExpiresTimeMonitor.
		With(prometheus.Labels{
			"tls_monitor_name": t.Label,
			"tls_domain_name":  t.Domain,
		}).
		Set(remainingTime)
}
