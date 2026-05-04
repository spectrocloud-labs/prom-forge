package gauge

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/exp/api/remote"
	writev2 "github.com/prometheus/client_golang/exp/api/remote/genproto/v2"
	"github.com/spectrocloud-labs/prom-forge/internal/metrics"
)

// GaugeMetric is the main struct for a gauge metric.
type GaugeMetric struct {
	name    string
	pattern Pattern
	labels  map[string]string
}

// New creates a new GaugeMetric.
func New(name string, pattern Pattern, labels map[string]string) *GaugeMetric {
	return &GaugeMetric{name: name, labels: labels, pattern: pattern}
}

// Name returns the name of the gauge metric.
func (g *GaugeMetric) Name() string {
	return g.name
}

// Type returns the type of the gauge metric.
func (g *GaugeMetric) Type() writev2.Metadata_MetricType {
	return writev2.Metadata_METRIC_TYPE_GAUGE
}

// Next returns the next value of the gauge metric.
func (g *GaugeMetric) Next(timestamp time.Time) float64 {
	return g.pattern.Next(timestamp)
}

// RemoteWrite writes the gauge metric to Prometheus.
func (g *GaugeMetric) RemoteWrite(ctx context.Context, client *remote.API, metricValue float64, timestamp time.Time) error {
	return metrics.RemoteWrite(ctx, client, g.name, g.labels, g.Type(), metricValue, timestamp)
}
