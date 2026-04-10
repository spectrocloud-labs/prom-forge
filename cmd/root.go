package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spectrocloud-labs/prom-forge/internal/config"
	"github.com/spectrocloud-labs/prom-forge/internal/exporter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	cfg     config.Config
	help    bool
)

var rootCmd = &cobra.Command{
	Use:   "prom-forge",
	Short: "Prometheus tooling CLI",
	Long:  `prom-forge is a command-line tool with YAML-based configuration.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.Unmarshal(&cfg); err != nil {
			return fmt.Errorf("parse config: %w", err)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if help {
			return cmd.Help()
		}

		c := Config()
		fmt.Printf("config: %+v\n", c)

		exporter.Export(c)

		return nil
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: $HOME/.prom-forge/config.yaml)")
	rootCmd.PersistentFlags().BoolVar(&help, "help", false, "help")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "home directory: %v\n", err)
			return
		}
		viper.AddConfigPath(filepath.Join(home, ".prom-forge"))
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
	}

	viper.SetEnvPrefix("PROM_FORGE")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			_, _ = fmt.Fprintf(os.Stderr, "read config: %v\n", err)
		}
	}
}

// Config returns the loaded configuration (after PersistentPreRun).
func Config() config.Config {
	return cfg
}
