package config

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
