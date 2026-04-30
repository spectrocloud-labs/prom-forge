package main

import (
	"os"

	"github.com/charmbracelet/log"
	"github.com/spectrocloud-labs/prom-forge/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Error(err)
		os.Exit(1)
	}
}
