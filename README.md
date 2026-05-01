# prom-forge
Config-driven synthetic metrics generator for Prometheus using remote_write

## Quick Start
1. Create a prometheus config.

```yaml
global:
  scrape_interval: 1s

scrape_configs:
  - job_name: "prometheus"
    static_configs:
      - targets: ["localhost:9090"]
        labels:
          app: "prometheus"

# Note: out_of_order_time_window is required if using `time machine` to allow for concurrent metric writes in the past and present.
storage:
  tsdb:
    out_of_order_time_window: 1d
```

1. Run prometheus with the remote_write receiver flag.
```sh
./prometheus --config.file prometheus.yml --web.enable-remote-write-receiver
```

1. Edit the example config `config.example.yaml` to your liking.

1. Run cli.
```sh
go run . --config config.example.yaml
```

## Generating Data

### Generating Data in the Past
You can generate data in the past. Below is a configuration to generate a steady gpu utilization metric for the past 15m at an interval of 5s plus a random jitter between 0-2s.

```yaml
prometheus:
  remote_write_url: "http://localhost:9090/"
  insecure_skip_verify: true
metrics:
- name: "gpu_utilization"
  type: "gauge"
  utilization_pattern:
    steady:
      value: 7.0
  labels:
    node: edge-ffa238429efe572a777ef4a17e4fd9b7
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
  utilization_pattern:
    steady:
      value: 7.0
  labels:
    node: edge-ffa238429efe572a777ef4a17e4fd9b7
  interval_duration: 5s
  jitter_duration: 2s
```

### Generating Data in the Past and the Present
You can generate data in the past and the present.

Below is a configuration to generate a steady gpu utilization metric for the past 15m and the present at an interval of 5s plus a random jitter between 0-2s.

```yaml
prometheus:
  remote_write_url: "http://localhost:9090/"
  insecure_skip_verify: true
metrics:
- name: "gpu_utilization"
  type: "gauge"
  utilization_pattern:
    steady:
      value: 7.0
  labels:
    node: edge-ffa238429efe572a777ef4a17e4fd9b7
  interval_duration: 5s
  jitter_duration: 2s
  time_machine_duration: 15m
```

Note, you can also generate data in the past that has a different utilization pattern than data in the present. This can be helpful for mocking CPU, GPU, etc. signal behaviors. Below is a configuration to generate a steady gpu utilization metric for the past 15m at an interval of 5s plus a random jitter between 0-2s, then generate a random gpu utilization metric for the present at an interval of 5s plus a random jitter between 0-2s.

```yaml
prometheus:
  remote_write_url: "http://localhost:9090/"
  insecure_skip_verify: true
metrics:
- name: "gpu_utilization"
  type: "gauge"
  utilization_pattern:
    steady:
      value: 7.0
  labels:
    node: edge-ffa238429efe572a777ef4a17e4fd9b7
  tick: false
  interval_duration: 5s
  jitter_duration: 2s
  time_machine_duration: 15m
- name: "gpu_utilization"
  type: "gauge"
  utilization_pattern:
    random:
      max: 100.0
      min: 50.0
  labels:
    node: edge-ffa238429efe572a777ef4a17e4fd9b7
  interval_duration: 2s
  jitter_duration: 5s
```

## Config

```yaml
# Note: be sure to add the following flag when running the prometheus server: --web.enable-remote-write-receiver
prometheus:
  # the remote write url to write the metrics to (required)
  remote_write_url: string
  # certificate check switch (optional)
  insecure_skip_verify: bool
  # certificate file to use for the remote write url (optional)
  ca_file: string
  # basic authentication configuration (optional)
  basic_auth:
    username: string
    password: string
metrics:
  # metric name (required)
- name: string
  # the metric type (required)
  type: "gauge"
  # utilization pattern for metric (required)
  utilization_pattern:
    # write a steady value
    steady:
      value: float64
    # write a random value
    random:
      max: float64
      min: float64
    # oscillate between 2 values
    oscillating:
      phase_a:
        value: float64
        hold_count: int
        ramp_steps: int
      phase_b:
        value: float64
        hold_count: int
        ramp_steps: int
  # labels for metric (optional)
  labels:
    string: string
  # generate metrics in the present (optional, default: true)
  tick: bool
  # writes metric at the given interval duration (required)
  interval_duration: time.Duration
  # add jitter between 0 and the given value to the interval duration (default: 0s)
  jitter_duration: time.Duration
  # generates metrics in the past for the time machine duration (default: 0s)
  time_machine_duration: time.Duration
```