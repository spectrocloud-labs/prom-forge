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

### Generating Data in the Past
You can generate data in the past. Below is a configuration to generate a steady gpu utilization metric for the past 15m at an interval of 5s plus a random jitter between 0-2s.

```yaml
prometheus:
  remote_write_url: "http://localhost:9090/"
  insecure_skip_verify: true
metrics:
- name: "gpu_utilization"
  type: "gauge"
  utilizationPattern:
    steady:
      value: 7.0
  labels:
  - node: edge-ffa238429efe572a777ef4a17e4fd9b7
  tick: false
  interval_duration: 5s
  jitter_duration: 2s
  time_machine_duration: 15m
```

### Generating Data in the Present
You can generate data in the present. Below is a configuration to generate a steady gpu utilization metric at an interval of 5s plus a random jitter between 0-2s.

```yaml
prometheus:
  remote_write_url: "http://localhost:9090/"
  insecure_skip_verify: true
metrics:
- name: "gpu_utilization"
  type: "gauge"
  utilizationPattern:
    steady:
      value: 7.0
  labels:
  - node: edge-ffa238429efe572a777ef4a17e4fd9b7
  interval_duration: 5s
  jitter_duration: 2s
```

### Generating Data in the Past... and the Present
You can generate data in the past and the present.

Below is a configuration to generate a steady gpu utilization metric for the past 15m and the present at an interval of 5s plus a random jitter between 0-2s.

```yaml
prometheus:
  remote_write_url: "http://localhost:9090/"
  insecure_skip_verify: true
metrics:
- name: "gpu_utilization"
  type: "gauge"
  utilizationPattern:
    steady:
      value: 7.0
  labels:
  - node: edge-ffa238429efe572a777ef4a17e4fd9b7
  interval_duration: 5s
  jitter_duration: 2s
  time_machine_duration: 15m
```

Note, you can also generate data in the past that has a different utilization pattern than data in the present. This can be helpful for mocking CPU, GPU, etc. signal behaviors. Below is a configuration to generate a steady gpu utilization metric for the past 15m then generate a random gpu utilization metric for the present at an interval of 5s plus a random jitter between 0-2s.

```yaml
prometheus:
  remote_write_url: "http://localhost:9090/"
  insecure_skip_verify: true
metrics:
- name: "gpu_utilization"
  type: "gauge"
  utilizationPattern:
    steady:
      value: 7.0
  labels:
  - node: edge-ffa238429efe572a777ef4a17e4fd9b7
  tick: false
  interval_duration: 5s
  jitter_duration: 2s
  time_machine_duration: 15m
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