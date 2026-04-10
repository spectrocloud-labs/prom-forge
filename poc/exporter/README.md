# prom-forge-exporter

The exporter generates mock metrics and serves them for prometheus to scrape. This pull mechanism always gets metric metadata because it does not use the remote_write api which does not support metadata in v1. This exporter can also be ran via prometheus in agent mode to remote_write it's metrics to another prometheus instance if desired.

## Prerequisites

- Go 1.26+
- Docker (for running Prometheus)

## Step 1: Start Prometheus

A `prometheus.yml` is included in this directory that configures Prometheus to scrape the exporter. Start the container:

```bash
docker run -d --name prometheus \
  -p 9090:9090 \
  -v $(pwd)/prometheus.yml:/etc/prometheus/prometheus.yml \
  prom/prometheus
```

Verify it's running:

```bash
docker ps | grep prometheus
```

## Step 2: Start the Exporter

Edit `config.yaml` if desired, then run from the `poc/exporter` directory:

```bash
go run .
```

You should see the config printed to stdout, and the exporter will start serving metrics on port 8080.

## Step 3: Verify

### Check the raw metrics endpoint

```bash
curl http://localhost:8080/metrics
```

You should see output like:

```
# HELP prom_forge_exporter_gpu
# TYPE prom_forge_exporter_gpu gauge
prom_forge_exporter_gpu 2
```

### Check that Prometheus is scraping

```bash
curl -s 'http://localhost:9090/api/v1/query?query=prom_forge_exporter_gpu' | jq
```

### View in the Prometheus UI

1. Open `http://localhost:9090` in your browser.
2. Type `prom_forge_exporter_gpu` into the query field.
3. Click **Execute**.
4. The **Table** tab shows the current value. The **Graph** tab shows values over time as Prometheus scrapes the exporter at each interval.

## Cleanup

Stop the exporter with `Ctrl+C`, then remove the Prometheus container:

```bash
docker rm -f prometheus
```