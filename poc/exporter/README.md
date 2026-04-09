# prom-forge-exporter
The exporter generates mock metrics and serves them for prometheus to scrape. This pull mechanism always gets metric metadata because it does not use the remote_write api which does not support metadata in v1. This exporter can also be ran via prometheus in agent mode to remote_write it's metrics to another prometheus instance if desired.

1. Add the scraper in your prometheus.yml.
```yaml
...
scrape_configs:
  - job_name: "prom-forge-exporter"
    static_configs:
      - targets: ["localhost:8080"]
        labels:
          app: "prom-forge-exporter"
...
```

Edit the prom-forge-exporter `config.yaml` to your desire.

3. Run the prom-forge-exporter.
```golang
go run .
```