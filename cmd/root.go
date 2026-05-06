package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/charmbracelet/log"

	"github.com/spectrocloud-labs/prom-forge/pkg/config"
	"github.com/spf13/cobra"
)

var (
	cfgFile   string
	cfg       config.Config
	logFormat string
	logLevel  int8
)

var rootCmd = &cobra.Command{
	Use:           "prom-forge",
	Short:         "Prometheus tooling CLI",
	Long:          `prom-forge is a command-line tool with YAML-based configuration.`,
	SilenceErrors: true,
	SilenceUsage:  true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		path := cfgFile
		if path == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("home directory: %w", err)
			}
			path = filepath.Join(home, ".prom-forge", "config.yaml")
		}
		loaded, err := config.Load(path)
		if err != nil {
			return err
		}
		cfg = loaded
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {

		client, err := config.CreateClientFromConfig(cfg)
		if err != nil {
			return fmt.Errorf("create client: %w", err)
		}

		tasks, err := config.CreateTasksFromConfig(cfg, client)
		if err != nil {
			return fmt.Errorf("create tasks: %w", err)
		}

		// create signal handler for graceful shutdown
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		// create and run time machine and tick tasks for each metric
		var wg sync.WaitGroup
		for _, task := range tasks {
			wg.Go(func() {
				if task.TimeMachineDuration() > 0 {
					task.StartTimeMachine(ctx)
				}
				if task.Tick() {
					task.Start(ctx)
				}
			})
		}

		// wait for goroutine tasks to cleanup and return
		wg.Wait()

		// exit
		log.Info("exiting")

		return nil
	},
}

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initLog)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: $HOME/.prom-forge/config.yaml)")
	rootCmd.PersistentFlags().StringVarP(&logFormat, "format", "f", "text", "format output (text, logfmt, json)")
	rootCmd.PersistentFlags().Int8VarP(&logLevel, "log-level", "l", 0, "log level (-4: debug, 0: info, 4: warn, 8: error)")
}

func initLog() {
	switch logFormat {
	case "text":
		log.SetFormatter(log.TextFormatter)
	case "json":
		log.SetFormatter(log.JSONFormatter)
	case "logfmt":
		log.SetFormatter(log.LogfmtFormatter)
	default:
		log.Error("invalid log format", "format", logFormat)
		os.Exit(1)
	}

	log.SetLevel(log.Level(logLevel))
}
