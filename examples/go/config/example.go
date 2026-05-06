package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/charmbracelet/log"
	"github.com/spectrocloud-labs/prom-forge/pkg/config"
)

func main() {
	configPath := flag.String("config", "../../configs/example.yaml", "path to YAML config")
	flag.Parse()

	log.SetLevel(log.DebugLevel)

	// Load YAML config: reads file, applies defaults, validates.
	cfg, err := config.Load(*configPath)
	if err != nil {
		panic(err)
	}

	client, err := config.CreateClientFromConfig(cfg)
	if err != nil {
		panic(err)
	}

	tasks, err := config.CreateTasksFromConfig(cfg, client)
	if err != nil {
		panic(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var wg sync.WaitGroup
	for _, task := range tasks {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if task.TimeMachineDuration() > 0 {
				task.StartTimeMachine(ctx)
			}
			if task.Tick() {
				task.Start(ctx)
			}
		}()
	}

	wg.Wait()

	log.Info("Exited")
}
