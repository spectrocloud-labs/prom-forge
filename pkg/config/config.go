package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"
)

// Config holds application settings loaded from YAML (and env overrides).
type Config struct {
	Prometheus PrometheusConfig `mapstructure:"prometheus" yaml:"prometheus"`
	Metrics    []Metric         `mapstructure:"metrics" yaml:"metrics"`
}

// PrometheusConfig holds Prometheus configuration.
type PrometheusConfig struct {
	RemoteWriteURL     string     `mapstructure:"remote_write_url" yaml:"remote_write_url"`
	CaFile             string     `mapstructure:"ca_file" yaml:"ca_file"`
	InsecureSkipVerify bool       `mapstructure:"insecure_skip_verify" yaml:"insecure_skip_verify"`
	BasicAuth          *BasicAuth `mapstructure:"basic_auth" yaml:"basic_auth"`
}

// BasicAuth holds basic authentication configuration.
type BasicAuth struct {
	Username string       `mapstructure:"username" yaml:"username"`
	Password OpaqueString `mapstructure:"password" yaml:"password"`
}

// Metric defines one synthetic metric to emit.
type Metric struct {
	Name                string             `mapstructure:"name" yaml:"name"`
	Type                string             `mapstructure:"type" yaml:"type"`
	UtilizationPattern  UtilizationPattern `mapstructure:"utilization_pattern" yaml:"utilization_pattern"`
	Labels              map[string]string  `mapstructure:"labels" yaml:"labels"`
	Tick                *bool              `mapstructure:"tick" yaml:"tick"`
	IntervalDuration    time.Duration      `mapstructure:"interval_duration" yaml:"interval_duration"`
	JitterDuration      time.Duration      `mapstructure:"jitter_duration" yaml:"jitter_duration"`
	TimeMachineDuration time.Duration      `mapstructure:"time_machine_duration" yaml:"time_machine_duration"`
}

// UtilizationPattern defines the utilization pattern for a metric.
type UtilizationPattern struct {
	Steady      *SteadyUtilizationPattern      `mapstructure:"steady" yaml:"steady,omitempty"`
	Oscillating *OscillatingUtilizationPattern `mapstructure:"oscillating" yaml:"oscillating,omitempty"`
	Random      *RandomUtilizationPattern      `mapstructure:"random" yaml:"random,omitempty"`
}

// SteadyUtilizationPattern defines the steady utilization pattern.
type SteadyUtilizationPattern struct {
	Slope  float64 `mapstructure:"slope" yaml:"slope"`
	Offset float64 `mapstructure:"offset" yaml:"offset"`
}

// RandomUtilizationPattern defines the random utilization pattern.
type RandomUtilizationPattern struct {
	Max float64 `mapstructure:"max" yaml:"max"`
	Min float64 `mapstructure:"min" yaml:"min"`
}

// OscillationPhase defines one half of oscillation cycle.
type OscillationPhase struct {
	Value     float64 `mapstructure:"value" yaml:"value"`
	HoldCount uint    `mapstructure:"hold_count" yaml:"hold_count"`
	RampSteps uint    `mapstructure:"ramp_steps" yaml:"ramp_steps"`
}

// OscillatingUtilizationPattern oscillates between two phases.
type OscillatingUtilizationPattern struct {
	PhaseA OscillationPhase `mapstructure:"phase_a" yaml:"phase_a"`
	PhaseB OscillationPhase `mapstructure:"phase_b" yaml:"phase_b"`
}

// Validate validates the configuration.
func Validate(config Config) error {
	if config.Prometheus.RemoteWriteURL == "" {
		return fmt.Errorf("required field 'prometheus.remote_write_url' is not set")
	}
	parsedURL, err := url.Parse(config.Prometheus.RemoteWriteURL)
	if err != nil {
		return fmt.Errorf("field 'prometheus.remote_write_url' is not a valid URL: %w", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("field 'prometheus.remote_write_url' must use http or https scheme, got %q", parsedURL.Scheme)
	}
	if parsedURL.Host == "" {
		return fmt.Errorf("field 'prometheus.remote_write_url' is missing a host")
	}
	if len(config.Metrics) == 0 {
		return fmt.Errorf("required field 'metrics' is not set")
	}

	if config.Prometheus.CaFile != "" {
		if _, err := os.Stat(config.Prometheus.CaFile); err != nil {
			return fmt.Errorf("field 'prometheus.ca_file' is set but file was not accessible at path %s: %w", config.Prometheus.CaFile, err)
		}
	}

	for i, cfgMetric := range config.Metrics {
		m := &cfgMetric

		if m.Name == "" {
			return fmt.Errorf("required field 'metrics[%d].name' is not set", i)
		}

		if m.IntervalDuration <= 0 {
			return fmt.Errorf("interval_duration must be greater than 0 for metric %s", m.Name)
		}
		if m.JitterDuration < 0 {
			return fmt.Errorf("jitter_duration must be >= 0 for metric %s", m.Name)
		}
		if m.TimeMachineDuration < 0 {
			return fmt.Errorf("time_machine_duration must be >= 0 for metric %s", m.Name)
		}

		utilPatternsSet := 0
		if m.UtilizationPattern.Steady != nil {
			utilPatternsSet++
		}
		if m.UtilizationPattern.Oscillating != nil {
			utilPatternsSet++
		}
		if m.UtilizationPattern.Random != nil {
			utilPatternsSet++
		}

		if utilPatternsSet != 1 {
			return fmt.Errorf("please set exactly 1 utilization pattern for metric %s", m.Name)
		}

		switch strings.ToLower(m.Type) {
		case "gauge":
			continue
		case "counter":
			continue
		default:
			return fmt.Errorf("unknown metric type: %s", m.Type)
		}
	}

	return nil
}

// Default sets the default values for the configuration.
func Default(config *Config) {
	for i := range config.Metrics {
		cfgMetric := &config.Metrics[i]
		cfgMetric.Type = strings.ToLower(cfgMetric.Type)
		var tick bool = true
		if cfgMetric.Tick == nil {
			cfgMetric.Tick = &tick
		}
	}
}
