package exporter

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"log"
	"maps"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spectrocloud-labs/prom-forge/internal/config"
	"github.com/spectrocloud-labs/prom-forge/internal/utilization"
)

type ExportTask struct {
	Name            string
	Labels          map[string]string
	Interval        time.Duration
	UtilizationFunc iter.Seq[float64]
}

type GaugeExportTask struct {
	ExportTask
	Gauge *prometheus.GaugeVec
}

func (task *GaugeExportTask) Export(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	ticker := time.NewTicker(task.Interval)
	defer ticker.Stop() // always stop ticker to free resources

	fmt.Printf("[%s] started\n", task.Name)

	// pull utilization function
	next, stop := iter.Pull(task.UtilizationFunc)
	defer stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Printf("[%s] shutting down\n", task.Name)
			return
		case _ = <-ticker.C:
			fmt.Printf("[%s] tick at %s\n", task.Name, time.Now().Format(time.TimeOnly))

			// iterate next value from utilization function
			if val, ok := next(); ok {
				fmt.Printf("[%s] value: %f\n", task.Name, val)
				task.Gauge.With(task.Labels).Set(val)
			}
		}
	}
}

func getGaugeExportTasks(reg prometheus.Registerer, metrics []config.Metric) map[string]*GaugeExportTask {
	taskMap := map[string]*GaugeExportTask{}
	for _, m := range metrics {
		// checks for valid configuration
		duration, err := time.ParseDuration(m.Interval)
		if err != nil {
			panic(fmt.Sprintf("error parsing interval: %v", err))
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
			panic(fmt.Sprintf("please set exactly 1 utilization patterns for metric %s", m.Name))
		}

		labels := map[string]string{}
		for _, l := range m.Labels {
			for k, v := range l {
				labels[k] = v
			}
		}

		switch m.Type {
		case "gauge":
			gauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: m.Name}, slices.Collect(maps.Keys(labels)))
			taskMap[m.Name] = &GaugeExportTask{
				ExportTask: ExportTask{
					Name:            m.Name,
					Labels:          labels,
					Interval:        duration,
					UtilizationFunc: utilizationFunc,
				},
				Gauge: gauge,
			}
			reg.MustRegister(gauge)
		default:
			panic(fmt.Sprintf("unknown metric type: %s", m.Type))
		}
	}

	return taskMap
}

func Export(config config.Config) {
	// create signal handler for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// create wait group for graceful shutdown
	var wg sync.WaitGroup

	// Create a non-global registry.
	reg := prometheus.NewRegistry()

	// create HTTP server multiplexer for serving metrics
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))

	// create HTTP server
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	// start HTTP server
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	// wait for shutdown signal, then gracefully stop HTTP server
	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("HTTP server shutdown error: %v", err)
		}
	}()

	// create map of gauge export tasks
	gaugeExportTasks := getGaugeExportTasks(reg, config.Metrics)

	// start all gauge export tasks
	for _, task := range gaugeExportTasks {
		wg.Add(1)
		go task.Export(ctx, &wg)
	}

	wg.Wait()
	fmt.Println("exiting")
}
