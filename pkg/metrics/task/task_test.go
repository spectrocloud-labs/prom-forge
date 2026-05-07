package task

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/prometheus/client_golang/exp/api/remote"
	"github.com/spectrocloud-labs/prom-forge/pkg/metrics"
	"github.com/spectrocloud-labs/prom-forge/pkg/metrics/counter"
	"github.com/spectrocloud-labs/prom-forge/pkg/metrics/gauge"
)

// recordingMetric delegates Next/Name to an inner metric and records RemoteWrite
// calls without contacting Prometheus.
type recordingMetric struct {
	inner  metrics.Metric
	writes []writeRecord
}

type writeRecord struct {
	value     float64
	timestamp time.Time
}

func (r *recordingMetric) Name() string { return r.inner.Name() }

func (r *recordingMetric) Next(t time.Time) float64 { return r.inner.Next(t) }

func (r *recordingMetric) RemoteWrite(ctx context.Context, client *remote.API, metricValue float64, timestamp time.Time) error {
	r.writes = append(r.writes, writeRecord{value: metricValue, timestamp: timestamp})
	return nil
}

func approxEq(a, b float64) bool {
	return math.Abs(a-b) <= 1e-6
}

func assertPastWritesMatchPattern(t *testing.T, writes []writeRecord, twinPattern func() metrics.Metric, tmDuration, interval time.Duration) {
	t.Helper()

	if len(writes) < 1 {
		t.Fatalf("expected at least one write, got %d", len(writes))
	}

	after := time.Now()

	for i, w := range writes {
		if !w.timestamp.Before(after) {
			t.Fatalf("write %d: timestamp %v must be before wall-clock end %v (historical samples)", i, w.timestamp, after)
		}
		if after.Sub(w.timestamp) > tmDuration+10*interval {
			t.Fatalf("write %d: timestamp %v is older than backfill window (got age %v, max ~%v)", i, w.timestamp, after.Sub(w.timestamp), tmDuration+10*interval)
		}
		if i > 0 && !writes[i-1].timestamp.Before(w.timestamp) {
			t.Fatalf("write %d: timestamps must increase strictly (got %v then %v)", i, writes[i-1].timestamp, w.timestamp)
		}
	}

	twin := twinPattern()
	for i, w := range writes {
		want := twin.Next(w.timestamp)
		if !approxEq(w.value, want) {
			t.Fatalf("write %d: value %g want %g (timestamp %v)", i, w.value, want, w.timestamp)
		}
	}
}

func TestStartTimeMachine_gaugeSteady_writesPastData(t *testing.T) {
	const slope, offset = 2.5, 100

	pat, err := gauge.NewSteady(slope, offset)
	if err != nil {
		t.Fatal(err)
	}
	inner := gauge.New("gauge_steady_tm", pat, nil)

	rec := &recordingMetric{inner: inner}

	tmDuration := 30 * time.Millisecond
	interval := 10 * time.Millisecond

	New(rec, interval, 0, tmDuration, false, nil).StartTimeMachine(context.Background())

	twin := func() metrics.Metric {
		p, err := gauge.NewSteady(slope, offset)
		if err != nil {
			t.Fatal(err)
		}
		return gauge.New("gauge_steady_twin", p, nil)
	}

	assertPastWritesMatchPattern(t, rec.writes, twin, tmDuration, interval)
}

func TestStartTimeMachine_counterSteady_writesPastData(t *testing.T) {
	const slope, offset = 0.75, 10

	pat, err := counter.NewSteady(slope, offset)
	if err != nil {
		t.Fatal(err)
	}
	inner := counter.New("counter_steady_tm", pat, nil)

	rec := &recordingMetric{inner: inner}

	tmDuration := 30 * time.Millisecond
	interval := 10 * time.Millisecond

	New(rec, interval, 0, tmDuration, false, nil).StartTimeMachine(context.Background())

	twin := func() metrics.Metric {
		p, err := counter.NewSteady(slope, offset)
		if err != nil {
			t.Fatal(err)
		}
		return counter.New("counter_steady_twin", p, nil)
	}

	assertPastWritesMatchPattern(t, rec.writes, twin, tmDuration, interval)
}
