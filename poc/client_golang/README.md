# client_golang Remote Write v2 POC

Pushes a single gauge metric (`prom_forge_client_golang_gpu`) to a Prometheus instance using the experimental remote_write v2 API.

> **Note:** remote_write v2 metadata (help, unit) is not yet accessible in the Prometheus API or UI.

## Prerequisites

- Go 1.26+
- Docker (for running Prometheus)

## Start Prometheus

Start a Prometheus instance with the remote write receiver enabled:

```bash
docker run -d --name prometheus \
  -p 9090:9090 \
  prom/prometheus \
  --config.file=/etc/prometheus/prometheus.yml \
  --web.enable-remote-write-receiver
```

The `--web.enable-remote-write-receiver` flag enables the `/api/v1/write` endpoint that accepts pushed metrics.

## Run the POC

```bash
cd poc/client_golang
go run main.go
```

On success you should see output like:

```
request: ...
remote_write v2 ok samples=1
```

## Verify

### Prometheus UI

1. Open `http://localhost:9090` in your browser.
2. Type `prom_forge_client_golang_gpu` into the query field.
3. Click **Execute**.
4. The **Table** tab shows the raw value. The **Graph** tab shows it on a timeline (with a single sample it will be one point).

### curl

```bash
curl -s 'http://localhost:9090/api/v1/query?query=prom_forge_client_golang_gpu' | jq
```

You should see a single sample with value `100.0`.

## Cleanup

```bash
docker rm -f prometheus
```
