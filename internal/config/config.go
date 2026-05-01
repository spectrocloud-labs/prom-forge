package config

import (
	"fmt"
	"os"
	"time"
)

// Config holds application settings loaded from YAML (and env overrides).
type Config struct {
	Prometheus prometheusConfig `mapstructure:"prometheus" yaml:"prometheus"`
	Metrics    []metric         `mapstructure:"metrics" yaml:"metrics"`
}

// PrometheusConfig holds Prometheus configuration.
type prometheusConfig struct {
	RemoteWriteURL     string     `mapstructure:"remote_write_url" yaml:"remote_write_url"`
	CaFile             string     `mapstructure:"ca_file" yaml:"ca_file"`
	InsecureSkipVerify bool       `mapstructure:"insecure_skip_verify" yaml:"insecure_skip_verify"`
	BasicAuth          *BasicAuth `mapstructure:"basic_auth" yaml:"basic_auth"`
}

// BasicAuth holds basic authentication configuration.
type BasicAuth struct {
	Username string `mapstructure:"username" yaml:"username"`
	Password string `mapstructure:"password" yaml:"password"`
}

// Metric defines one synthetic metric to emit.
type metric struct {
	Name                string             `mapstructure:"name" yaml:"name"`
	Type                string             `mapstructure:"type" yaml:"type"`
	UtilizationPattern  utilizationPattern `mapstructure:"utilization_pattern" yaml:"utilization_pattern"`
	Labels              map[string]string  `mapstructure:"labels" yaml:"labels"`
	Tick                *bool              `mapstructure:"tick" yaml:"tick"`
	IntervalDuration    time.Duration      `mapstructure:"interval_duration" yaml:"interval_duration"`
	JitterDuration      time.Duration      `mapstructure:"jitter_duration" yaml:"jitter_duration"`
	TimeMachineDuration time.Duration      `mapstructure:"time_machine_duration" yaml:"time_machine_duration"`
}

// UtilizationPattern defines the utilization pattern for a metric.
type utilizationPattern struct {
	Steady      *SteadyUtilizationPattern      `mapstructure:"steady" yaml:"steady,omitempty"`
	Oscillating *OscillatingUtilizationPattern `mapstructure:"oscillating" yaml:"oscillating,omitempty"`
	Random      *RandomUtilizationPattern      `mapstructure:"random" yaml:"random,omitempty"`
}

// SteadyUtilizationPattern defines the steady utilization pattern.
type SteadyUtilizationPattern struct {
	Value float64 `mapstructure:"value" yaml:"value"`
}

// RandomUtilizationPattern defines the random utilization pattern.
type RandomUtilizationPattern struct {
	Max float64 `mapstructure:"max" yaml:"max"`
	Min float64 `mapstructure:"min" yaml:"min"`
}

// oscillationPhase defines one half of oscillation cycle.
type oscillationPhase struct {
	Value     float64 `mapstructure:"value" yaml:"value"`
	HoldCount int     `mapstructure:"hold_count" yaml:"hold_count"`
	RampSteps int     `mapstructure:"ramp_steps" yaml:"ramp_steps"`
}

// OscillatingUtilizationPattern oscillates between two phases.
type OscillatingUtilizationPattern struct {
	PhaseA oscillationPhase `mapstructure:"phase_a" yaml:"phase_a"`
	PhaseB oscillationPhase `mapstructure:"phase_b" yaml:"phase_b"`
}

// Validate validates the configuration.
func Validate(config Config) error {
	if config.Prometheus.RemoteWriteURL == "" {
		return fmt.Errorf("required field 'prometheus.remote_write_url' is not set")
	}
	if len(config.Metrics) == 0 {
		return fmt.Errorf("required field 'metrics' is not set")
	}

	if config.Prometheus.CaFile != "" {
		if _, err := os.ReadFile(config.Prometheus.CaFile); err != nil {
			return fmt.Errorf("field 'prometheus.ca_file' is set but file was not readable at path %s", config.Prometheus.CaFile)
		}
	}

	for _, cfgMetric := range config.Metrics {
		m := &cfgMetric

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
			p := m.UtilizationPattern.Oscillating
			if p.PhaseA.HoldCount < 0 || p.PhaseA.RampSteps < 0 || p.PhaseB.HoldCount < 0 || p.PhaseB.RampSteps < 0 {
				return fmt.Errorf("oscillating hold_count and ramp_steps must be >= 0 for metric %s", m.Name)
			}
			if p.PhaseA.HoldCount+p.PhaseA.RampSteps+p.PhaseB.HoldCount+p.PhaseB.RampSteps == 0 {
				return fmt.Errorf("oscillating pattern must emit at least one sample for metric %s", m.Name)
			}
		}
		if m.UtilizationPattern.Random != nil {
			if m.UtilizationPattern.Random.Max <= m.UtilizationPattern.Random.Min {
				return fmt.Errorf("utilization_pattern.random required field 'max' must be greater than required field 'min' for metric %s", m.Name)
			}
			utilPatternsSet++
		}

		if utilPatternsSet != 1 {
			return fmt.Errorf("please set exactly 1 utilization pattern for metric %s", m.Name)
		}

		switch m.Type {
		case "gauge":
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
		var tick bool = true
		if cfgMetric.Tick == nil {
			cfgMetric.Tick = &tick
		}
	}
}
