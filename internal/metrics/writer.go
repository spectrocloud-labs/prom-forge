package metrics

import (
	"context"
	"crypto/tls"
	"crypto/x509"
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

	"github.com/charmbracelet/log"
	"github.com/prometheus/client_golang/exp/api/remote"
	writev2 "github.com/prometheus/client_golang/exp/api/remote/genproto/v2"
	"github.com/spectrocloud-labs/prom-forge/internal/client"
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
	log.Info("remote_write v2 ok", "metric", task.Name, "value", metricValue, "timestamp", timestamp.Format(time.DateTime))
	return nil
}

// StartTimeMachine starts the time machine for the metric writer task.
func (task *MetricWriterTask) StartTimeMachine(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	next, stop := iter.Pull(task.UtilizationFunc)
	defer stop()

	log.Info("starting time machine", "metric", task.Name, "duration", task.TimeMachineDuration)
	pastTime := time.Now().Add(-task.TimeMachineDuration)
	presentTime := time.Now()
	currentTime := pastTime
	for currentTime.Before(presentTime) {
		select {
		case <-ctx.Done():
			log.Info("shutting down time machine", "metric", task.Name)
			return
		default:
		}

		if val, ok := next(); ok {
			err := task.write(ctx, val, currentTime)
			if err != nil {
				log.Error("time machine error", "metric", task.Name, "error", err)
			}
		}
		// #nosec G404
		jitterDur := time.Duration(rand.Int64N(int64(task.JitterDuration.Nanoseconds()) + 1))
		intervalDur := task.IntervalDuration + jitterDur
		currentTime = currentTime.Add(intervalDur)
	}

	log.Info("time machine completed", "metric", task.Name)
}

// Start starts the metric writer task.
func (task *MetricWriterTask) Start(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	// #nosec G404
	jitterDur := time.Duration(rand.Int64N(int64(task.JitterDuration.Nanoseconds()) + 1))

	ticker := time.NewTicker(task.IntervalDuration + jitterDur)
	defer ticker.Stop() // always stop ticker to free resources

	log.Info("starting ticker", "metric", task.Name)

	// pull utilization function
	next, stop := iter.Pull(task.UtilizationFunc)
	defer stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("shutting down ticker", "metric", task.Name)
			return
		case <-ticker.C:
			now := time.Now()
			log.Info("tick", "metric", task.Name, "timestamp", now.Format(time.DateTime))

			// iterate next value from utilization function
			if val, ok := next(); ok {
				err := task.write(ctx, val, now)
				if err != nil {
					log.Error("tick error", "metric", task.Name, "error", err)
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

		tlsConfig := &tls.Config{
			// #nosec G402
			InsecureSkipVerify: config.Prometheus.InsecureSkipVerify,
			MinVersion:         tls.VersionTLS12,
		}

		// #nosec G402
		if config.Prometheus.InsecureSkipVerify == true {
			log.Warn("insecure skip verify is enabled in your config")
		}

		if config.Prometheus.CaFile != "" {
			caCert, err := os.ReadFile(config.Prometheus.CaFile)
			if err != nil {
				return nil, fmt.Errorf("error reading ca file %s for metric %s", config.Prometheus.CaFile, m.Name)
			}
			caCertPool := x509.NewCertPool()
			if !caCertPool.AppendCertsFromPEM(caCert) {
				return nil, fmt.Errorf("ca file %s for metric %s does not contain a valid PEM certificate", config.Prometheus.CaFile, m.Name)
			}
			tlsConfig.RootCAs = caCertPool
		}

		transport := &http.Transport{TLSClientConfig: tlsConfig}
		httpClient := &http.Client{Transport: transport, Timeout: time.Minute}

		// add basic authentication roundtripper if configured
		if config.Prometheus.BasicAuth != nil {
			basicAuthRoundTripper := &client.BasicAuthRoundTripper{
				Username: config.Prometheus.BasicAuth.Username,
				Password: config.Prometheus.BasicAuth.Password,
				Next:     transport,
			}
			httpClient.Transport = basicAuthRoundTripper
		}

		// create remote API client
		client, err := remote.NewAPI(config.Prometheus.RemoteWriteURL, remote.WithAPIHTTPClient(httpClient))
		if err != nil {
			return nil, fmt.Errorf("error creating prometheus remote API client: %v", err)
		}

		taskList = append(taskList, &MetricWriterTask{
			Name:                m.Name,
			Type:                writev2.Metadata_METRIC_TYPE_GAUGE,
			Labels:              labels,
			Tick:                *m.Tick,
			IntervalDuration:    m.IntervalDuration,
			JitterDuration:      m.JitterDuration,
			TimeMachineDuration: m.TimeMachineDuration,
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
	log.Info("exiting")
	return nil
}
