package metrics

import (
	"context"
	"crypto/tls"
	"fmt"
	"iter"
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
	IntervalDuration    time.Duration
	JitterDuration      time.Duration
	TimeMachineDuration time.Duration
	UtilizationFunc     iter.Seq[float64]
	client              *remote.API
}

// write writes a metric to Prometheus.
func (task *MetricWriterTask) write(_ context.Context, metricValue float64, timestamp int64) error {
	sym := writev2.NewSymbolTable()
	labelsRefs := []string{
		"__name__", task.Name,
	}
	for k, v := range task.Labels {
		labelsRefs = append(labelsRefs, k, v)
	}

	tsMs := (time.Duration(timestamp) * time.Second).Milliseconds()
	ts := &writev2.TimeSeries{
		LabelsRefs: sym.SymbolizeLabels(labelsRefs, nil),
		Samples: []*writev2.Sample{
			{Value: metricValue, Timestamp: tsMs},
		},
		Metadata: &writev2.Metadata{
			Type: writev2.Metadata_METRIC_TYPE_GAUGE,
		},
	}

	req := &writev2.Request{
		Symbols:    sym.Symbols(),
		Timeseries: []*writev2.TimeSeries{ts},
	}
	req.Symbols = sym.Symbols()

	_, err := task.client.Write(context.Background(), remote.WriteV2MessageType, req)
	if err != nil {
		return fmt.Errorf("remote_write v2 failed: %v", err)
	}
	fmt.Printf("[%s] remote_write v2 ok (value=%f, timestamp=%s)\n", task.Name, metricValue, time.UnixMilli(tsMs).Format(time.TimeOnly))
	return nil
}

// StartTimeMachine starts the time machine for the metric writer task.
func (task *MetricWriterTask) StartTimeMachine(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	next, stop := iter.Pull(task.UtilizationFunc)
	defer stop()

	fmt.Printf("[%s] starting time machine for the last %s\n", task.Name, task.TimeMachineDuration)
	pastTime := time.Now().Add(-task.TimeMachineDuration).Unix()
	currentTime := pastTime
	for currentTime < time.Now().Unix() {
		if val, ok := next(); ok {
			err := task.write(ctx, val, currentTime)
			if err != nil {
				fmt.Printf("[%s] time machine error: %v\n", task.Name, err)
			}
		}
		jitterDur := time.Duration(rand.IntN(int(task.JitterDuration.Seconds())+1)) * time.Second
		currentTime += int64(task.IntervalDuration.Seconds() + jitterDur.Seconds())
	}

	fmt.Printf("[%s] time machine completed\n", task.Name)
}

// Start starts the metric writer task.
func (task *MetricWriterTask) Start(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	jitterDur := time.Duration(rand.IntN(int(task.JitterDuration.Seconds())+1)) * time.Second

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
			fmt.Printf("[%s] tick at %s\n", task.Name, time.Now().Format(time.TimeOnly))

			// iterate next value from utilization function
			if val, ok := next(); ok {
				err := task.write(ctx, val, time.Now().Unix())
				if err != nil {
					fmt.Printf("[%s] tick error: %v\n", task.Name, err)
				}
			}

			// add jitter to interval if configured
			jitterDur := time.Duration(rand.IntN(int(task.JitterDuration.Seconds())+1)) * time.Second
			ticker.Reset(task.IntervalDuration + jitterDur)
		}
	}
}

// getTasksFromConfig gets the metric writer tasks from the config.
func getTasksFromConfig(config config.Config) (map[string]*MetricWriterTask, error) {
	if config.Prometheus.RemoteWriteURL == "" {
		return nil, fmt.Errorf("required field 'prometheus.remote_write_url' is not set")
	}
	if len(config.Metrics) == 0 {
		return nil, fmt.Errorf("required field 'metrics' is not set")
	}

	taskMap := map[string]*MetricWriterTask{}
	for _, m := range config.Metrics {
		// checks for valid configuration
		interval, err := time.ParseDuration(m.IntervalDuration)
		if err != nil {
			return nil, fmt.Errorf("error parsing required field 'interval_duration': %v", err)
		}

		jitter, err := time.ParseDuration(m.JitterDuration)
		if err != nil && m.JitterDuration != "" {
			return nil, fmt.Errorf("error parsing optional field 'jitter_duration': %v", err)
		} else if m.JitterDuration == "" {
			jitter = time.Duration(0)
		}

		timeMachineDuration, err := time.ParseDuration(m.TimeMachineDuration)
		if err != nil && m.TimeMachineDuration != "" {
			return nil, fmt.Errorf("error parsing optional field 'time_machine_duration': %v", err)
		} else if m.TimeMachineDuration == "" {
			timeMachineDuration = time.Duration(0)
		}

		var utilizationFunc iter.Seq[float64]
		utilPatternsSet := 0
		if m.UtilizationPattern.Steady != nil {
			utilizationFunc = utilization.SteadyUtilization(*m.UtilizationPattern.Steady)
			utilPatternsSet++
		}
		if m.UtilizationPattern.Oscillating != nil {
			utilizationFunc = utilization.OscillatingUtilization(*m.UtilizationPattern.Oscillating)
			utilPatternsSet++
		}
		if m.UtilizationPattern.Random != nil {
			utilizationFunc = utilization.RandomUtilization(*m.UtilizationPattern.Random)
			utilPatternsSet++
		}
		if utilPatternsSet > 1 || utilPatternsSet == 0 {
			return nil, fmt.Errorf("please set exactly 1 utilization pattern for metric %s", m.Name)
		}

		labels := map[string]string{}
		for _, l := range m.Labels {
			for k, v := range l {
				labels[k] = v
			}
		}

		client, err := remote.NewAPI(config.Prometheus.RemoteWriteURL, remote.WithAPIHTTPClient(&http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: config.Prometheus.InsecureSkipVerify},
			},
		}))
		if err != nil {
			return nil, fmt.Errorf("error creating prometheus remote API client: %v", err)
		}

		switch m.Type {
		case "gauge":
			taskMap[m.Name] = &MetricWriterTask{
				Name:                m.Name,
				Type:                writev2.Metadata_METRIC_TYPE_GAUGE,
				Labels:              labels,
				IntervalDuration:    interval,
				UtilizationFunc:     utilizationFunc,
				JitterDuration:      jitter,
				TimeMachineDuration: timeMachineDuration,
				client:              client,
			}
		default:
			return nil, fmt.Errorf("unknown metric type: %s", m.Type)
		}
	}

	return taskMap, nil
}

// StartWriter writes the metrics to Prometheus.
func StartWriter(config config.Config) {
	// create signal handler for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// create wait group for graceful shutdown
	var wg sync.WaitGroup

	// create map of metric writer tasks
	metricWriterTasks, err := getTasksFromConfig(config)
	if err != nil {
		panic(fmt.Sprintf("error getting metric writer tasks: %v", err))
	}

	// run time machine to generate metrics in the past
	for _, task := range metricWriterTasks {
		wg.Add(1)
		go task.StartTimeMachine(ctx, &wg)
	}

	// wait for time machine generation to complete
	wg.Wait()

	// generate metrics in the present
	for _, task := range metricWriterTasks {
		wg.Add(1)
		go task.Start(ctx, &wg)
	}

	// wait for all metric writer tasks to complete
	wg.Wait()

	// exit
	fmt.Println("exiting")
}
