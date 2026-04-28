package main

import (
	"os"

	"github.com/spectrocloud-labs/prom-forge/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
