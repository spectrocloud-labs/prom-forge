package metrics

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/exp/api/remote"
	writev2 "github.com/prometheus/client_golang/exp/api/remote/genproto/v2"
)

// Metric is the interface for a metric.
type Metric interface {
	// Name returns the name of the metric.
	Name() string

	// Next returns the next value of the metric.
	Next(timestamp time.Time) float64

	// RemoteWrite writes the metric to Prometheus.
	RemoteWrite(ctx context.Context, client *remote.API, metricValue float64, timestamp time.Time) error
}

// RemoteWrite writes a metric to Prometheus.
func RemoteWrite(ctx context.Context, client *remote.API, metricName string, metricLabels map[string]string, metricType writev2.Metadata_MetricType, metricValue float64, timestamp time.Time) error {
	sym := writev2.NewSymbolTable()
	labelsRefs := []string{
		"__name__", metricName,
	}
	for k, v := range metricLabels {
		labelsRefs = append(labelsRefs, k, v)
	}

	tsMs := timestamp.UnixMilli()
	ts := &writev2.TimeSeries{
		LabelsRefs: sym.SymbolizeLabels(labelsRefs, nil),
		Samples: []*writev2.Sample{
			{Value: metricValue, Timestamp: tsMs},
		},
		Metadata: &writev2.Metadata{
			Type: metricType,
		},
	}

	req := &writev2.Request{
		Symbols:    sym.Symbols(),
		Timeseries: []*writev2.TimeSeries{ts},
	}

	_, err := client.Write(ctx, remote.WriteV2MessageType, req)
	if err != nil {
		return fmt.Errorf("remote_write v2 failed: %v", err)
	}

	return nil
}
