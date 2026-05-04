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

	"github.com/spectrocloud-labs/prom-forge/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
		if err := viper.Unmarshal(&cfg); err != nil {
			return fmt.Errorf("parse config: %w", err)
		}

		// set default values
		config.Default(&cfg)

		// validate configuration
		if err := config.Validate(cfg); err != nil {
			return fmt.Errorf("validate config: %w", err)
		}
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
			if task.TimeMachineDuration() > 0 {
				wg.Add(1)
				go func() {
					defer wg.Done()
					task.StartTimeMachine(ctx)
				}()
			}
			if task.Tick() {
				wg.Add(1)
				go func() {
					defer wg.Done()
					task.Start(ctx)
				}()
			}
		}

		// wait for signal
		<-ctx.Done()

		// wait for goroutine tasks to cleanup
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
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: $HOME/.prom-forge/config.yaml)")
	rootCmd.PersistentFlags().StringVarP(&logFormat, "format", "f", "text", "format output (text, logfmt, json)")
	rootCmd.PersistentFlags().Int8VarP(&logLevel, "log-level", "l", 0, "log level (-4: debug, 0: info, 4: warn, 8: error)")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Error("home directory", "error", err)
			os.Exit(1)
		}
		viper.AddConfigPath(filepath.Join(home, ".prom-forge"))
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
	}

	viper.SetEnvPrefix("PROM_FORGE")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			log.Error("read config", "error", err)
			os.Exit(1)
		}
	}

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
