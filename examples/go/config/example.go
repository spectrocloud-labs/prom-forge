package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/charmbracelet/log"
	"github.com/spectrocloud-labs/prom-forge/pkg/config"
)

func main() {
	// Set log level to debug
	log.SetLevel(log.DebugLevel)

	// Set up config
	cfg := config.Config{
		Prometheus: config.PrometheusConfig{RemoteWriteURL: "http://localhost:9090/", InsecureSkipVerify: true},
		Metrics: []config.Metric{
			{
				Name: "prom_forge_test_metric",
				Type: "gauge",
				UtilizationPattern: config.UtilizationPattern{
					Random: &config.RandomUtilizationPattern{
						Min: 1.0,
						Max: 100.0,
					},
				},
				Labels: map[string]string{
					"purpose": "testing",
				},
				Tick:             new(true),
				IntervalDuration: time.Duration(time.Second * 1),
			},
		},
	}

	// Set default values on config
	config.Default(&cfg)

	// Validate config
	err := config.Validate(cfg)
	if err != nil {
		panic(err)
	}

	// create client from config
	client, err := config.CreateClientFromConfig(cfg)
	if err != nil {
		panic(err)
	}

	// create tasks from config
	tasks, err := config.CreateTasksFromConfig(cfg, client)
	if err != nil {
		panic(err)
	}

	// set up context and signal handling
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Info("Starting task")
		// start task to write metrics to prometheus
		tasks[0].Start(ctx)
	}()

	wg.Wait()

	log.Info("Exited")
}
