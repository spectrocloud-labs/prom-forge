package counter

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/exp/api/remote"
	writev2 "github.com/prometheus/client_golang/exp/api/remote/genproto/v2"
	"github.com/spectrocloud-labs/prom-forge/internal/metrics"
)

// CounterMetric is the main struct for a counter metric.
type CounterMetric struct {
	name       string
	pattern    Pattern
	labels     map[string]string
	metricType writev2.Metadata_MetricType
}

// New creates a new CounterMetric.
func New(name string, pattern Pattern, labels map[string]string) *CounterMetric {
	return &CounterMetric{name: name, labels: labels, pattern: pattern, metricType: writev2.Metadata_METRIC_TYPE_COUNTER}
}

// Name returns the name of the counter metric.
func (c *CounterMetric) Name() string {
	return c.name
}

// Next returns the next value of the counter metric.
func (c *CounterMetric) Next(timestamp time.Time) float64 {
	return c.pattern.Next(timestamp)
}

// RemoteWrite writes the counter metric to Prometheus.
func (c *CounterMetric) RemoteWrite(ctx context.Context, client *remote.API, metricValue float64, timestamp time.Time) error {
	return metrics.RemoteWrite(ctx, client, c.name, c.labels, c.metricType, metricValue, timestamp)
}
