package config

import (
	"fmt"
	"time"
)

// Config holds application settings loaded from YAML (and env overrides).
type Config struct {
	Prometheus prometheusConfig `mapstructure:"prometheus" yaml:"prometheus"`
	Metrics    []metric         `mapstructure:"metrics" yaml:"metrics"`
}

// PrometheusConfig holds Prometheus configuration.
type prometheusConfig struct {
	RemoteWriteURL     string `mapstructure:"remote_write_url" yaml:"remote_write_url"`
	InsecureSkipVerify bool   `mapstructure:"insecure_skip_verify" yaml:"insecure_skip_verify"`
}

// Metric defines one synthetic metric to emit.
type metric struct {
	Name                string             `mapstructure:"name" yaml:"name"`
	Type                string             `mapstructure:"type" yaml:"type"`
	UtilizationPattern  utilizationPattern `mapstructure:"utilization_pattern" yaml:"utilization_pattern"`
	Labels              map[string]string  `mapstructure:"labels" yaml:"labels"`
	Tick                *bool              `mapstructure:"tick" yaml:"tick"`
	IntervalDuration    string             `mapstructure:"interval_duration" yaml:"interval_duration"`
	JitterDuration      string             `mapstructure:"jitter_duration" yaml:"jitter_duration"`
	TimeMachineDuration string             `mapstructure:"time_machine_duration" yaml:"time_machine_duration"`
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

// OscillatingUtilizationPattern defines the oscillating utilization pattern.
type OscillatingUtilizationPattern struct {
	Y1            float64 `mapstructure:"y1" yaml:"y1"`
	Y1Count       int     `mapstructure:"y1_count" yaml:"y1_count"`
	Y2            float64 `mapstructure:"y2" yaml:"y2"`
	Y2Count       int     `mapstructure:"y2_count" yaml:"y2_count"`
	Y1Y2StepCount int     `mapstructure:"y1y2_step_count" yaml:"y1y2_step_count"`
	Y2Y1StepCount int     `mapstructure:"y2y1_step_count" yaml:"y2y1_step_count"`
}

// Validate validates the configuration.
func Validate(config Config) error {
	if config.Prometheus.RemoteWriteURL == "" {
		return fmt.Errorf("required field 'prometheus.remote_write_url' is not set")
	}
	if len(config.Metrics) == 0 {
		return fmt.Errorf("required field 'metrics' is not set")
	}

	for _, cfgMetric := range config.Metrics {
		m := &cfgMetric
		_, err := time.ParseDuration(m.IntervalDuration)
		if err != nil {
			return fmt.Errorf("error parsing required field 'interval_duration': %v", err)
		}

		_, err = time.ParseDuration(m.JitterDuration)
		if err != nil {
			return fmt.Errorf("error parsing optional field 'jitter_duration': %v", err)
		}

		_, err = time.ParseDuration(m.TimeMachineDuration)
		if err != nil {
			return fmt.Errorf("error parsing optional field 'time_machine_duration': %v", err)
		}

		utilPatternsSet := 0
		switch {
		case m.UtilizationPattern.Steady != nil:
			utilPatternsSet++
			fallthrough
		case m.UtilizationPattern.Oscillating != nil:
			utilPatternsSet++
			fallthrough
		case m.UtilizationPattern.Random != nil:
			if m.UtilizationPattern.Random.Max > m.UtilizationPattern.Random.Min {
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
		if cfgMetric.JitterDuration == "" {
			cfgMetric.JitterDuration = time.Duration(0).String()
		}
		if cfgMetric.TimeMachineDuration == "" {
			cfgMetric.TimeMachineDuration = time.Duration(0).String()
		}
		var tick bool = true
		if cfgMetric.Tick == nil {
			cfgMetric.Tick = &tick
		}
	}
}
