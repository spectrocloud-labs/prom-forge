# prom-forge
Config-driven synthetic metrics generator for Prometheus using remote_write

## Quick Start
1. Run prometheus with the remote_write receiver flag.
```sh
./prometheus --config.file prometheus.yml --web.enable-remote-write-receiver
```

2. Edit the example config `config.example.yaml` to your liking.

2. Run cli.
```sh
go run . --config config.example.yaml
```

## The Past... and the Present
You can generate data in the past that has a different utilization pattern than data in the present. This can be helpful for mocking CPU, GPU, etc. signal behaviors. Below is an example config of this.

```yaml
prometheus:
  remote_write_url: "http://localhost:9090/"
  insecure_skip_verify: true
metrics:
- name: "gpu_utilization"
  type: "gauge"
  utilizationPattern:
    oscillating:
      y1: 30.0
      y1Count: 3
      y2: 10.0
      y2Count: 3
      y1y2StepCount: 2
      y2y1StepCount: 2
  labels:
  - node: edge-ffa238429efe572a777ef4a17e4fd9b7
  tick: false
  interval_duration: 5s
  jitter_duration: 2s
  time_machine_duration: 5m
- name: "gpu_utilization"
  type: "gauge"
  utilizationPattern:
    random:
      max: 100.0
      min: 50.0
  labels:
  - node: edge-ffa238429efe572a777ef4a17e4fd9b7
  interval_duration: 2s
  jitter_duration: 5s
```