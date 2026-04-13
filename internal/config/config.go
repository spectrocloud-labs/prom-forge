package config

// Config holds application settings loaded from YAML (and env overrides).
type Config struct {
	Metrics []Metric `mapstructure:"metrics" yaml:"metrics"`
}

// Metric defines one synthetic metric to emit.
type Metric struct {
	Name               string              `mapstructure:"name" yaml:"name"`
	Type               string              `mapstructure:"type" yaml:"type"`
	UtilizationPattern UtilizationPattern  `mapstructure:"utilizationPattern" yaml:"utilizationPattern"`
	Labels             []map[string]string `mapstructure:"labels" yaml:"labels"`
	Interval           string              `mapstructure:"interval" yaml:"interval"`
	Jitter             string              `mapstructure:"jitter" yaml:"jitter"`
}

// UtilizationPattern defines the utilization pattern for a metric.
type UtilizationPattern struct {
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
	SustainValue float64 `mapstructure:"sustainValue" yaml:"sustainValue"`
	MinValue     float64 `mapstructure:"minValue" yaml:"minValue"`
	MinCount     int     `mapstructure:"minCount" yaml:"minCount"`
	SustainCount int     `mapstructure:"sustainCount" yaml:"sustainCount"`
	DropSteps    int     `mapstructure:"dropSteps" yaml:"dropSteps"`
	RiseSteps    int     `mapstructure:"riseSteps" yaml:"riseSteps"`
}
