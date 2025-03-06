# prom-exporter

This project is an application for exporting metrics to Prometheus.

## About the Project (WIP)
The application starts an HTTP server for Prometheus and data contributors.
Each contributor should have its own handler, for example, `./handlers/images_handler` for the image parser.
In it, the metrics that need to be forwarded to Prometheus are implemented.

### Metric Naming:
- The metrics themselves are named following the principle `metricName` = `<application>_<metricName>`
- The Redis key name is the metric name with the prefix `prometheus:<metricName>`
- Example: for the metric `successful_uploads_total` of the `parser_images` application, the full metric name for debugging is `parser_images_successful_uploads_total`, and the Redis key is `prometheus:parser_images_successful_uploads_total`

For adding a new metric: ...<TBD>

For adding a new contributor: ...<TBD>

The main metrics of the application are available at http://localhost:8200/metrics

## Environment Variables

- `REDIS_SYNC_INTERVAL`: Interval in seconds for saving metrics to Redis. Example value: `10`
- `PORT`: Port on which the server will run. Example value: `8200`
- `REDIS_DSN`: Connection string for Redis. Example value: `redis://localhost:6379`
- `GIN_MODE`: Gin mode (a web framework for Go). Example value: `release`. Possible values: `debug | release | test`.

## Installing Dependencies

To install the project dependencies, run:

```bash
go mod download
```

## Local launch

The project can be launched using the following command:

```bash
go run main.go
```

Alternatively, you can run it in Docker with the following commands:
```bash
docker build -t prom-exporter . 
docker run -p 8200:8200 prom-exporter
```

## Build and Production Run

To build, run:
```bash
go build
```

The application is executed by invoking the compiled file:
```bash
./prom-exporter
```

## Integration with Prometheus

For integration with Prometheus, ensure that you have added your metrics in the code and registered them using prometheus.MustRegister().
Then, for Prometheus to collect metrics, add the following configuration in your YAML manifest. Example:
```yaml
scrape_configs:
  - job_name: 'prom-exporter'
    static_configs:
      - targets: ['localhost:8200']
```

## Deployment

When deploying this application in a DevOps environment, it is recommended to configure the following items:
*	Set up a Redis server and configure REDIS_DSN according to its address and port.
*	Set the value of the REDIS_SYNC_INTERVAL variable based on the desired frequency of metric updates.