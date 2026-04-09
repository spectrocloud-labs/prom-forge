package main

import (
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.yaml.in/yaml/v2"
)

// Config holds application settings loaded from YAML (and env overrides).
type Config struct {
	Metrics []Metric `mapstructure:"metrics" yaml:"metrics"`
}

// Metric defines one synthetic metric to emit.
type Metric struct {
	Name               string              `mapstructure:"name" yaml:"name"`
	Type               string              `mapstructure:"type" yaml:"type"`
	UtilizationPattern string              `mapstructure:"utilizationPattern" yaml:"utilizationPattern"`
	Labels             []map[string]string `mapstructure:"labels" yaml:"labels"`
	Interval           string              `mapstructure:"interval" yaml:"interval"`
}

// Read config
func ReadConfig() Config {
	data, err := os.ReadFile("config.yaml")
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	return config
}

// Utilization functions
var utilizationMap = map[string]func() float64{
	"steady": func() float64 {
		min, max := 1, 3
		val := rand.IntN(max-min+1) + min
		return float64(val)
	},
}

func NewMetrics(reg prometheus.Registerer, metrics []Metric) map[string]prometheus.Collector {
	metricsMap := map[string]prometheus.Collector{}
	for _, m := range metrics {
		switch m.Type {
		case "gauge":
			metricsMap[m.Name] = prometheus.NewGauge(prometheus.GaugeOpts{
				Name: m.Name,
			})
		default:
			panic(fmt.Sprintf("unknown metric type: %s", m.Type))
		}
	}
	for _, collector := range metricsMap {
		reg.MustRegister(collector)
	}
	return metricsMap
}

func main() {
	// Read config
	config := ReadConfig()
	fmt.Printf("config: %+v\n", config)

	// Create a non-global registry.
	reg := prometheus.NewRegistry()

	// Create new metrics and register them using the custom registry.
	metrics := NewMetrics(reg, config.Metrics)

	// Serve gauge metrics
	for _, metric := range config.Metrics {
		if _, ok := metrics[metric.Name]; !ok {
			log.Fatalf("skipping, metric %s not found", metric.Name)
			continue
		}

		duration, err := time.ParseDuration(metric.Interval)
		if err != nil {
			log.Fatalf("error: %v", err)
		}
		ticker := time.NewTicker(duration)
		defer ticker.Stop()

		go func() {
			for range ticker.C {
				utilizationFunc := utilizationMap[metric.UtilizationPattern]
				metrics[metric.Name].(prometheus.Gauge).Set(utilizationFunc())
			}
		}()
	}

	// Expose metrics and custom registry via an HTTP server
	// using the HandleFor function. "/metrics" is the usual endpoint for that.
	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
