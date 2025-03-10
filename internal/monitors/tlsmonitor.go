package monitors

import (
	"crypto/tls"
	"log"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

type TLSMonitor struct {
	Label  string
	Domain string

	Logger *log.Logger

	ExpiresTimeMonitor *prometheus.GaugeVec
}

func (t *TLSMonitor) ID() string {
	return t.Label
}

func (t *TLSMonitor) Run() error {
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
	conn, err := tls.Dial("tcp", t.Domain+":443", nil)
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
