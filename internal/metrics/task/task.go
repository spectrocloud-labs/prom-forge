package task

import (
	"context"
	"math/rand/v2"
	"time"

	"github.com/charmbracelet/log"
	"github.com/prometheus/client_golang/exp/api/remote"
	"github.com/spectrocloud-labs/prom-forge/internal/metrics"
)

// Task is a structure for remote writing metrics to prometheus.
type Task struct {
	metrics.Metric
	intervalDuration    time.Duration
	jitterDuration      time.Duration
	timeMachineDuration time.Duration
	tick                bool
	client              *remote.API
}

// New creates a new remote metric writer task.
func New(metric metrics.Metric, intervalDuration time.Duration, jitterDuration time.Duration, timeMachineDuration time.Duration, tick bool, client *remote.API) *Task {
	return &Task{Metric: metric, intervalDuration: intervalDuration, jitterDuration: jitterDuration, timeMachineDuration: timeMachineDuration, tick: tick, client: client}
}

// TimeMachineDuration returns the time machine duration of the task.
func (task *Task) TimeMachineDuration() time.Duration {
	return task.timeMachineDuration
}

// Tick returns whether the task should be ticked.
func (task *Task) Tick() bool {
	return task.tick
}

// Start starts the task.
func (task *Task) Start(ctx context.Context) {
	// #nosec G404
	jitterDur := time.Duration(rand.Int64N(int64(task.jitterDuration.Nanoseconds()) + 1))

	ticker := time.NewTicker(task.intervalDuration + jitterDur)
	defer ticker.Stop() // always stop ticker to free resources

	log.Debug("starting ticker", "metric", task.Name())

	for {
		select {
		case <-ctx.Done():
			log.Debug("shutting down ticker", "metric", task.Name())
			return
		case <-ticker.C:
			now := time.Now()
			val := task.Next(now)
			log.Debug("tick", "metric", task.Name(), "timestamp", now.Format(time.DateTime), "value", val)
			if err := task.RemoteWrite(ctx, task.client, val, now); err != nil {
				log.Warn("tick error", "metric", task.Name(), "error", err)
			}

			// add jitter to interval if configured
			// #nosec G404
			jitterDur := time.Duration(rand.Int64N(int64(task.jitterDuration.Nanoseconds()) + 1))
			ticker.Reset(task.intervalDuration + jitterDur)
		}
	}
}

// StartTimeMachine starts the time machine for the task.
func (task *Task) StartTimeMachine(ctx context.Context) {
	log.Debug("starting time machine", "metric", task.Name(), "duration", task.timeMachineDuration)
	pastTime := time.Now().Add(-task.timeMachineDuration)
	presentTime := time.Now()
	currentTime := pastTime
	for currentTime.Before(presentTime) {
		select {
		case <-ctx.Done():
			log.Debug("shutting down time machine", "metric", task.Name())
			return
		default:
		}

		val := task.Next(currentTime)
		log.Debug("time machine tick", "metric", task.Name(), "timestamp", currentTime.Format(time.DateTime), "value", val)
		if err := task.RemoteWrite(ctx, task.client, val, currentTime); err != nil {
			log.Warn("time machine error", "metric", task.Name(), "error", err)
		}

		// #nosec G404
		jitterDur := time.Duration(rand.Int64N(int64(task.jitterDuration.Nanoseconds()) + 1))
		intervalDur := task.intervalDuration + jitterDur
		currentTime = currentTime.Add(intervalDur)
	}

	log.Debug("time machine completed", "metric", task.Name())
}
