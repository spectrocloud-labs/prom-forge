# prom-forge
Config-driven synthetic metrics generator for Prometheus using remote_write

## Quick Start
1. Edit example config `config.example.yaml` ensuring you choose a single `utilizationPattern`.

2. Run cli.
```sh
go run . --config config.example.yaml
```