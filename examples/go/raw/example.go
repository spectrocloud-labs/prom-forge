package main

import (
	"context"
	"crypto/tls"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/charmbracelet/log"
	"github.com/prometheus/client_golang/exp/api/remote"
	"github.com/spectrocloud-labs/prom-forge/pkg/metrics/gauge"
	"github.com/spectrocloud-labs/prom-forge/pkg/metrics/task"
)

func main() {
	// Set log level if prom-forge to debug
	log.SetLevel(log.DebugLevel)

	// Set up http client
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				// #nosec G402
				InsecureSkipVerify: true,
				MinVersion:         tls.VersionTLS12,
			},
		},
		Timeout: time.Minute,
	}

	apiClient, err := remote.NewAPI("http://localhost:9090/", remote.WithAPIHTTPClient(httpClient))
	if err != nil {
		panic(err)
	}

	pattern, err := gauge.NewRandom(10000.0, 100.0)
	if err != nil {
		panic(err)
	}

	metric := gauge.New("prom_forge_test_metric", pattern, map[string]string{"purpose": "testing"})

	var (
		interval    = time.Second * 1
		jitter      = time.Second * 0
		timeMachine = time.Second * 0
		tick        = true
	)
	task := task.New(metric, interval, jitter, timeMachine, tick, apiClient)

	// set up context and signal handling
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Info("Starting task")
		// start task to write metrics to prometheus
		task.Start(ctx)
	}()

	wg.Wait()

	log.Info("Exited")
}
