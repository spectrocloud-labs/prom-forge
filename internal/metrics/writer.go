package metrics

import (
	"context"
	"crypto/tls"
	"fmt"
	"iter"
	"maps"
	"math/rand/v2"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/exp/api/remote"
	writev2 "github.com/prometheus/client_golang/exp/api/remote/genproto/v2"
	"github.com/spectrocloud-labs/prom-forge/internal/config"
	"github.com/spectrocloud-labs/prom-forge/internal/utilization"
)

// MetricWriterTask writes a metric to Prometheus.
type MetricWriterTask struct {
	Name                string
	Type                writev2.Metadata_MetricType
	Labels              map[string]string
	Tick                bool
	IntervalDuration    time.Duration
	JitterDuration      time.Duration
	TimeMachineDuration time.Duration
	UtilizationFunc     iter.Seq[float64]
	client              *remote.API
}

// write writes a metric to Prometheus.
func (task *MetricWriterTask) write(ctx context.Context, metricValue float64, timestamp time.Time) error {
	sym := writev2.NewSymbolTable()
	labelsRefs := []string{
		"__name__", task.Name,
	}
	for k, v := range task.Labels {
		labelsRefs = append(labelsRefs, k, v)
	}

	tsMs := timestamp.UnixMilli()
	ts := &writev2.TimeSeries{
		LabelsRefs: sym.SymbolizeLabels(labelsRefs, nil),
		Samples: []*writev2.Sample{
			{Value: metricValue, Timestamp: tsMs},
		},
		Metadata: &writev2.Metadata{
			Type: task.Type,
		},
	}

	req := &writev2.Request{
		Symbols:    sym.Symbols(),
		Timeseries: []*writev2.TimeSeries{ts},
	}

	_, err := task.client.Write(ctx, remote.WriteV2MessageType, req)
	if err != nil {
		return fmt.Errorf("remote_write v2 failed: %v", err)
	}
	fmt.Printf("[%s] remote_write v2 ok (value=%f, timestamp=%s)\n", task.Name, metricValue, timestamp.Format(time.DateTime))
	return nil
}

// StartTimeMachine starts the time machine for the metric writer task.
func (task *MetricWriterTask) StartTimeMachine(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	next, stop := iter.Pull(task.UtilizationFunc)
	defer stop()

	fmt.Printf("[%s] starting time machine for the last %s\n", task.Name, task.TimeMachineDuration)
	pastTime := time.Now().Add(-task.TimeMachineDuration)
	presentTime := time.Now()
	currentTime := pastTime
	for currentTime.Before(presentTime) {
		if val, ok := next(); ok {
			err := task.write(ctx, val, currentTime)
			if err != nil {
				fmt.Printf("[%s] time machine error: %v\n", task.Name, err)
			}
		}
		// #nosec G404
		jitterDur := time.Duration(rand.Int64N(int64(task.JitterDuration.Nanoseconds()) + 1))
		intervalDur := task.IntervalDuration + jitterDur
		currentTime = currentTime.Add(intervalDur)
	}

	fmt.Printf("[%s] time machine completed\n", task.Name)
}

// Start starts the metric writer task.
func (task *MetricWriterTask) Start(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	// #nosec G404
	jitterDur := time.Duration(rand.Int64N(int64(task.JitterDuration.Nanoseconds()) + 1))

	ticker := time.NewTicker(task.IntervalDuration + jitterDur)
	defer ticker.Stop() // always stop ticker to free resources

	fmt.Printf("[%s] starting ticker\n", task.Name)

	// pull utilization function
	next, stop := iter.Pull(task.UtilizationFunc)
	defer stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Printf("[%s] shutting down ticker\n", task.Name)
			return
		case <-ticker.C:
			now := time.Now()
			fmt.Printf("[%s] tick at %s\n", task.Name, now.Format(time.DateTime))

			// iterate next value from utilization function
			if val, ok := next(); ok {
				err := task.write(ctx, val, now)
				if err != nil {
					fmt.Printf("[%s] tick error: %v\n", task.Name, err)
				}
			}

			// add jitter to interval if configured
			// #nosec G404
			jitterDur := time.Duration(rand.Int64N(int64(task.JitterDuration.Nanoseconds()) + 1))
			ticker.Reset(task.IntervalDuration + jitterDur)
		}
	}
}

// getTasksFromConfig gets the metric writer tasks from the config.
func getTasksFromConfig(config config.Config) ([]*MetricWriterTask, error) {
	taskList := []*MetricWriterTask{}
	for _, m := range config.Metrics {
		intervalDuration, _ := time.ParseDuration(m.IntervalDuration)
		jitterDuration, _ := time.ParseDuration(m.JitterDuration)
		timeMachineDuration, _ := time.ParseDuration(m.TimeMachineDuration)

		var utilizationFunc iter.Seq[float64]
		switch {
		case m.UtilizationPattern.Steady != nil:
			utilizationFunc = utilization.SteadyUtilization(*m.UtilizationPattern.Steady)
		case m.UtilizationPattern.Oscillating != nil:
			utilizationFunc = utilization.OscillatingUtilization(*m.UtilizationPattern.Oscillating)
		case m.UtilizationPattern.Random != nil:
			utilizationFunc = utilization.RandomUtilization(*m.UtilizationPattern.Random)
		}

		labels := map[string]string{}
		maps.Copy(labels, m.Labels)

		client, err := remote.NewAPI(config.Prometheus.RemoteWriteURL, remote.WithAPIHTTPClient(&http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					// #nosec G402
					InsecureSkipVerify: config.Prometheus.InsecureSkipVerify,
				},
			},
		}))
		if err != nil {
			return nil, fmt.Errorf("error creating prometheus remote API client: %v", err)
		}

		taskList = append(taskList, &MetricWriterTask{
			Name:                m.Name,
			Type:                writev2.Metadata_METRIC_TYPE_GAUGE,
			Labels:              labels,
			Tick:                *m.Tick,
			IntervalDuration:    intervalDuration,
			JitterDuration:      jitterDuration,
			TimeMachineDuration: timeMachineDuration,
			UtilizationFunc:     utilizationFunc,
			client:              client,
		})
	}

	return taskList, nil
}

// StartWriter writes the metrics to Prometheus.
func StartWriter(config config.Config) error {
	// create signal handler for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// create wait group for graceful shutdown
	var wg sync.WaitGroup

	// create map of metric writer tasks
	metricWriterTasks, err := getTasksFromConfig(config)
	if err != nil {
		return fmt.Errorf("error getting metric writer tasks: %v", err)
	}

	// run time machine to generate metrics in the past
	for _, task := range metricWriterTasks {
		wg.Add(1)
		go task.StartTimeMachine(ctx, &wg)
	}

	// wait for time machine generation to complete
	// wg.Wait()

	// generate metrics in the present
	for _, task := range metricWriterTasks {
		if !task.Tick {
			continue
		}
		wg.Add(1)
		go task.Start(ctx, &wg)
	}

	// wait for all metric writer tasks to complete
	wg.Wait()

	// exit
	fmt.Println("exiting")
	return nil
}
