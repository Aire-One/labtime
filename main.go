package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"net/http/httptrace"
	"os"
	"os/signal"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gopkg.in/yaml.v3"
)

type Config struct {
	// List of targets to ping
	Targets []struct {
		// Name of the target. Used to identify the target from Prometheus.
		Name string `yaml:"name"`
		// URL of the target. The target should be accessible from the machine running the exporter. The URL should contain the protocol (http:// or https://) and the port if it's not the default one.
		Url string `yaml:"url"`
		// Interval to ping the target. Default is 5 seconds
		Interval int `yaml:"interval,omitempty"`
	} `yaml:"targets"`
}

func NewConfig(configPath string) (*Config, error) {
	// Create config structure
	config := &Config{}

	// Open config file
	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Init new YAML decode
	d := yaml.NewDecoder(file)

	// Start YAML decoding from file
	if err := d.Decode(&config); err != nil {
		return nil, err
	}

	return config, nil
}

func Ping(url string) (int, time.Duration, error) {
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return 0, 0, err
	}
	var start, connect, dns, tlsHandshake time.Time
	trace := &httptrace.ClientTrace{
		DNSStart: func(dsi httptrace.DNSStartInfo) { dns = time.Now() },
		DNSDone: func(ddi httptrace.DNSDoneInfo) {
			fmt.Printf("DNS Done: %v\n", time.Since(dns))
		},

		TLSHandshakeStart: func() { tlsHandshake = time.Now() },
		TLSHandshakeDone: func(cs tls.ConnectionState, err error) {
			fmt.Printf("TLS Handshake: %v\n", time.Since(tlsHandshake))
		},

		ConnectStart: func(network, addr string) { connect = time.Now() },
		ConnectDone: func(network, addr string, err error) {
			fmt.Printf("Connect time: %v\n", time.Since(connect))
		},

		GotFirstResponseByte: func() {
			fmt.Printf("Time from start to first byte: %v\n", time.Since(start))
		},
	}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	start = time.Now()
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return 0, 0, err
	}
	resp.Body.Close()
	return resp.StatusCode, time.Since(start), nil
}

func main() {

	// Load config
	config, err := NewConfig("config.yaml")
	if err != nil {
		log.Fatalf("Error loading config: %s", err)
	}
	// Print config
	fmt.Printf("%+v\n", config)

	// create a scheduler
	s, err := gocron.NewScheduler()
	if err != nil {
		// handle error
	}

	// Prometheus metrics
	responseTimeMonitor := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "labtime_response_time_duration",
		Help: "The ping time.",
	}, []string{"target_name"})
	prometheus.MustRegister(responseTimeMonitor)

	// Intercept the signal to stop the program
	go func() {
		sigchan := make(chan os.Signal)
		signal.Notify(sigchan, os.Interrupt)
		<-sigchan
		log.Println("Program killed !")

		// do last actions and wait for all write operations to end

		// when you're done, shut it down
		err = s.Shutdown()
		if err != nil {
			// handle error
		}

		os.Exit(0)
	}()

	// add a job to the scheduler
	for _, target := range config.Targets {
		j, err := s.NewJob(
			gocron.DurationJob(
				func(interval int) time.Duration {
					if interval == 0 {
						return time.Duration(5) * time.Second
					}
					return time.Duration(interval) * time.Second
				}(target.Interval),
			),
			gocron.NewTask(
				func(url string) {
					status, elapsedTime, err := Ping(url)
					if err != nil {
						fmt.Println(err)
						return
					}
					fmt.Printf("%s - Status: %d in %v\n", target.Name, status, elapsedTime.Seconds())
					// push to Prometheus
					responseTimeMonitor.With(prometheus.Labels{"target_name": target.Name}).Set(elapsedTime.Seconds())
				},
				target.Url,
			),
		)
		if err != nil {
			// handle error
		}
		// each job has a unique id
		fmt.Printf("Job %s started with ID: %s\n", target.Name, j.ID().String())
	}

	// start the scheduler
	s.Start()

	// // block until you are ready to shut down
	// select {
	// // case <-time.After(time.Minute):
	// }

	// Serve Prometheus metrics
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":2112", nil)
}
