package gauge

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/exp/api/remote"
	writev2 "github.com/prometheus/client_golang/exp/api/remote/genproto/v2"
)

// GaugeMetric is the main struct for a gauge metric.
type GaugeMetric struct {
	name       string
	pattern    Pattern
	labels     map[string]string
	metricType writev2.Metadata_MetricType
}

// New creates a new GaugeMetric.
func New(name string, pattern Pattern, labels map[string]string) *GaugeMetric {
	return &GaugeMetric{name: name, labels: labels, pattern: pattern, metricType: writev2.Metadata_METRIC_TYPE_GAUGE}
}

// Name returns the name of the gauge metric.
func (g *GaugeMetric) Name() string {
	return g.name
}

// Next returns the next value of the gauge metric.
func (g *GaugeMetric) Next(timestamp time.Time) float64 {
	return g.pattern.Next(timestamp)
}

// RemoteWrite writes the gauge metric to Prometheus.
func (g *GaugeMetric) RemoteWrite(ctx context.Context, client *remote.API, metricValue float64, timestamp time.Time) error {
	return metrics.remoteWrite(ctx, client, g.name, g.labels, g.metricType, metricValue, timestamp)
}
