package config

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"maps"
	"net/http"
	"os"
	"time"

	"github.com/charmbracelet/log"
	"github.com/prometheus/client_golang/exp/api/remote"
	"github.com/spectrocloud-labs/prom-forge/pkg/metrics"
	"github.com/spectrocloud-labs/prom-forge/pkg/metrics/counter"
	"github.com/spectrocloud-labs/prom-forge/pkg/metrics/gauge"
	"github.com/spectrocloud-labs/prom-forge/pkg/metrics/task"
)

// CreateTasksFromConfig creates metric writer tasks from the config.
func CreateTasksFromConfig(config Config, client *remote.API) ([]*task.Task, error) {
	taskList := []*task.Task{}
	for _, m := range config.Metrics {
		labels := map[string]string{}
		maps.Copy(labels, m.Labels)

		tick := true
		if m.Tick != nil {
			tick = *m.Tick
		}

		var metric metrics.Metric
		switch m.Type {
		case "gauge":
			var (
				pattern gauge.Pattern
				err     error
			)
			switch {
			case m.UtilizationPattern.Steady != nil:
				pattern, err = gauge.NewSteady(m.UtilizationPattern.Steady.Slope, m.UtilizationPattern.Steady.Offset)
				if err != nil {
					return nil, fmt.Errorf("error creating steady pattern for metric %s: %w", m.Name, err)
				}
			case m.UtilizationPattern.Oscillating != nil:
				pattern, err = gauge.NewOscillating(m.UtilizationPattern.Oscillating.PhaseA.Value, m.UtilizationPattern.Oscillating.PhaseB.Value, uint(m.UtilizationPattern.Oscillating.PhaseA.HoldCount), uint(m.UtilizationPattern.Oscillating.PhaseA.RampSteps), uint(m.UtilizationPattern.Oscillating.PhaseB.HoldCount), uint(m.UtilizationPattern.Oscillating.PhaseB.RampSteps))
				if err != nil {
					return nil, fmt.Errorf("error creating oscillating pattern for metric %s: %w", m.Name, err)
				}
			case m.UtilizationPattern.Random != nil:
				pattern, err = gauge.NewRandom(m.UtilizationPattern.Random.Min, m.UtilizationPattern.Random.Max)
				if err != nil {
					return nil, fmt.Errorf("error creating random pattern for metric %s: %w", m.Name, err)
				}
			}

			metric = gauge.New(m.Name, pattern, labels)
			t := task.New(metric, m.IntervalDuration, m.JitterDuration, m.TimeMachineDuration, tick, client)
			taskList = append(taskList, t)
		case "counter":
			var (
				pattern counter.Pattern
				err     error
			)
			switch {
			case m.UtilizationPattern.Steady != nil:
				pattern, err = counter.NewSteady(m.UtilizationPattern.Steady.Slope, m.UtilizationPattern.Steady.Offset)
				if err != nil {
					return nil, fmt.Errorf("error creating steady pattern for metric %s: %w", m.Name, err)
				}
			case m.UtilizationPattern.Oscillating != nil:
				pattern, err = counter.NewOscillating(m.UtilizationPattern.Oscillating.PhaseA.Value, m.UtilizationPattern.Oscillating.PhaseB.Value, uint(m.UtilizationPattern.Oscillating.PhaseA.HoldCount), uint(m.UtilizationPattern.Oscillating.PhaseA.RampSteps), uint(m.UtilizationPattern.Oscillating.PhaseB.HoldCount), uint(m.UtilizationPattern.Oscillating.PhaseB.RampSteps))
				if err != nil {
					return nil, fmt.Errorf("error creating oscillating pattern for metric %s: %w", m.Name, err)
				}
			case m.UtilizationPattern.Random != nil:
				pattern, err = counter.NewRandom(m.UtilizationPattern.Random.Min, m.UtilizationPattern.Random.Max)
				if err != nil {
					return nil, fmt.Errorf("error creating random pattern for metric %s: %w", m.Name, err)
				}
			}

			metric = counter.New(m.Name, pattern, labels)
			t := task.New(metric, m.IntervalDuration, m.JitterDuration, m.TimeMachineDuration, tick, client)
			taskList = append(taskList, t)
		}
	}

	return taskList, nil
}

// CreateClientFromConfig creates a remote API client from the config.
func CreateClientFromConfig(config Config) (*remote.API, error) {
	tlsConfig := &tls.Config{
		// #nosec G402
		InsecureSkipVerify: config.Prometheus.InsecureSkipVerify,
		MinVersion:         tls.VersionTLS12,
	}

	// #nosec G402
	if config.Prometheus.InsecureSkipVerify {
		log.Warn("insecure skip verify is enabled in your config")
	}

	if config.Prometheus.CaFile != "" {
		caCert, err := os.ReadFile(config.Prometheus.CaFile)
		if err != nil {
			return nil, fmt.Errorf("error reading ca file %s: %w", config.Prometheus.CaFile, err)
		}
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("ca file %s does not contain a valid PEM certificate", config.Prometheus.CaFile)
		}
		tlsConfig.RootCAs = caCertPool
	}

	transport := &http.Transport{TLSClientConfig: tlsConfig}
	httpClient := &http.Client{Transport: transport, Timeout: time.Minute}

	// add basic authentication roundtripper if configured
	if config.Prometheus.BasicAuth != nil {
		basicAuthRoundTripper := basicAuthRoundTripper{
			Username: config.Prometheus.BasicAuth.Username,
			Password: config.Prometheus.BasicAuth.Password,
			Next:     transport,
		}
		httpClient.Transport = &basicAuthRoundTripper
	}

	// create remote API client
	apiClient, err := remote.NewAPI(config.Prometheus.RemoteWriteURL, remote.WithAPIHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("error creating prometheus remote API client: %w", err)
	}

	return apiClient, nil
}
