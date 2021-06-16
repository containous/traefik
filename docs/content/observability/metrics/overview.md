# Metrics

Traefik supports 4 metrics backends:

- [Datadog](./datadog.md)
- [InfluxDB](./influxdb.md)
- [Prometheus](./prometheus.md)
- [StatsD](./statsd.md)

## Configuration

To enable metrics:

```toml tab="File (TOML)"
[metrics]
```

```yaml tab="File (YAML)"
metrics: {}
```

```bash tab="CLI"
--metrics=true
```

## Server Metrics

| Metric                                                                  | DataDog | InfluxDB | Prometheus | StatsD |
|-------------------------------------------------------------------------|---------|----------|------------|--------|
| [Configuration reloads](#configuration-reloads)                         | ✓       | ✓        | ✓          | ✓      |
| [Configuration reload failures](#configuration-reload-failures)         | ✓       | ✓        | ✓          | ✓      |
| [Last Configuration Reload Success](#last-configuration-reload-success) | ✓       | ✓        | ✓          | ✓      |
| [Last Configuration Reload Failure](#last-configuration-reload-failure) | ✓       | ✓        | ✓          | ✓      |

### Configuration Reloads
The total count of configuration reloads.

```dd tab="Datadog"
config.reload.total
```

```influxdb tab="InfluDB"
traefik.config.reload.total
```

```prom tab="Prometheus"
traefik_config_reloads_total
```

```statsd tab="StatsD"
# Default prefix: "traefik"
{prefix}.config.reload.total
```

### Configuration Reload Failures
The total count of configuration reload failures.

```dd tab="Datadog"
config.reload.total (with tag "failure" to true)
```

```influxdb tab="InfluDB"
traefik.config.reload.total.failure
```

```prom tab="Prometheus"
traefik_config_reloads_failure_total
```

```statsd tab="StatsD"
# Default prefix: "traefik"
{prefix}.config.reload.total.failure
```

### Last Configuration Reload Success
The timestamp of the last configuration reload success.

```dd tab="Datadog"
config.reload.lastSuccessTimestamp
```

```influxdb tab="InfluDB"
traefik.config.reload.lastSuccessTimestamp
```

```prom tab="Prometheus"
traefik_config_last_reload_success
```

```statsd tab="StatsD"
# Default prefix: "traefik"
{prefix}.config.reload.lastSuccessTimestamp
```

### Last Configuration Reload Failure
The timestamp of the last configuration reload failure.

```dd tab="Datadog"
config.reload.lastFailureTimestamp
```

```influxdb tab="InfluDB"
traefik.config.reload.lastFailureTimestamp
```

```prom tab="Prometheus"
traefik_config_last_reload_failure
```

```statsd tab="StatsD"
# Default prefix: "traefik"
{prefix}.config.reload.lastFailureTimestamp
```

## EntryPoint Metrics

| Metric                                                    | DataDog | InfluxDB | Prometheus | StatsD |
|-----------------------------------------------------------|---------|----------|------------|--------|
| [HTTP Requests Count](#http-requests-count)               | ✓       | ✓        | ✓          | ✓      |
| [HTTPS Requests Count](#https-requests-count)             |         |          | ✓          |        |
| [Request Duration Histogram](#request-duration-histogram) | ✓       | ✓        | ✓          | ✓      |
| [Open Connections Count](#open-connections-count)         | ✓       | ✓        | ✓          | ✓      |

### HTTP Requests Count
The total count of HTTP requests processed on an entrypoint.

Available labels: `code`, `method`, `protocol`, `entrypoint`.

```dd tab="Datadog"
entrypoint.request.total
```

```influxdb tab="InfluDB"
traefik.entrypoint.requests.total
```

```prom tab="Prometheus"
traefik_entrypoint_requests_total
```

```statsd tab="StatsD"
# Default prefix: "traefik"
{prefix}.entrypoint.request.total
```

### HTTPS Requests Count
The total count of HTTPS requests processed on an entrypoint.

Available labels: `tls_version`, `tls_cipher`, `entrypoint`.

```prom tab="Prometheus"
traefik_entrypoint_requests_tls_total
```

### Request Duration Histogram
Request process time duration histogram on an entrypoint.

Available labels: `code`, `method`, `protocol`, `entrypoint`.

```dd tab="Datadog"
entrypoint.request.duration
```

```influxdb tab="InfluDB"
traefik.entrypoint.request.duration
```

```prom tab="Prometheus"
traefik_entrypoint_request_duration_seconds
```

```statsd tab="StatsD"
# Default prefix: "traefik"
{prefix}.entrypoint.request.duration
```

### Open Connections Count
The current count of open connections on an entrypoint.

Available labels: `method`, `protocol`, `entrypoint`.

```dd tab="Datadog"
entrypoint.connections.open
```

```influxdb tab="InfluDB"
traefik.entrypoint.connections.open
```

```prom tab="Prometheus"
traefik_entrypoint_open_connections
```

```statsd tab="StatsD"
# Default prefix: "traefik"
{prefix}.entrypoint.connections.open
```

## Service Metrics

| Metric                                                      | DataDog | InfluxDB | Prometheus | StatsD |
|-------------------------------------------------------------|---------|----------|------------|--------|
| [HTTP Requests Count](#http-requests-count_1)               | ✓       | ✓        | ✓          | ✓      |
| [HTTPS Requests Count](#https-requests-count_1)             |         |          | ✓          |        |
| [Request Duration Histogram](#request-duration-histogram_1) | ✓       | ✓        | ✓          | ✓      |
| [Open Connections Count](#open-connections-count_1)         | ✓       | ✓        | ✓          | ✓      |
| [Requests Retries Count](#requests-retries-count)           | ✓       | ✓        | ✓          | ✓      |
| [Service Server UP](#service-server-up)                     | ✓       | ✓        | ✓          | ✓      |

### HTTP Requests Count
The total count of HTTP requests processed on a service.

Available labels: `code`, `method`, `protocol`, `service`.

```dd tab="Datadog"
service.request.total
```

```influxdb tab="InfluDB"
traefik.service.requests.total
```

```prom tab="Prometheus"
traefik_service_requests_total
```

```statsd tab="StatsD"
# Default prefix: "traefik"
{prefix}.service.request.total
```

### HTTPS Requests Count
The total count of HTTPS requests processed on a service.

Available labels: `tls_version`, `tls_cipher`, `service`.

```prom tab="Prometheus"
traefik_service_requests_tls_total
```

### Request Duration Histogram
Request process time duration histogram on a service.

Available labels: `code`, `method`, `protocol`, `service`.

```dd tab="Datadog"
service.request.duration
```

```influxdb tab="InfluDB"
traefik.service.request.duration
```

```prom tab="Prometheus"
traefik_service_request_duration_seconds
```

```statsd tab="StatsD"
# Default prefix: "traefik"
{prefix}.service.request.duration
```

### Open Connections Count
The current count of open connections on a service.

Available labels: `method`, `protocol`, `service`.

```dd tab="Datadog"
service.connections.open
```

```influxdb tab="InfluDB"
traefik.service.connections.open
```

```prom tab="Prometheus"
traefik_service_open_connections
```

```statsd tab="StatsD"
# Default prefix: "traefik"
{prefix}.service.connections.open
```

### Requests Retries Count
The count of requests retries on a service.

Available labels: `service`.

```dd tab="Datadog"
service.retries.total
```

```influxdb tab="InfluDB"
traefik.service.retries.total
```

```prom tab="Prometheus"
traefik_service_retries_total
```

```statsd tab="StatsD"
# Default prefix: "traefik"
{prefix}.service.retries.total
```

### Service Server UP
Current service's server status, described by a gauge with a value of 0 for a down server or a value of 1 for an up server.

Available labels: `service`, `url`.

```dd tab="Datadog"
service.server.up
```

```influxdb tab="InfluDB"
traefik.service.server.up
```

```prom tab="Prometheus"
traefik_service_server_up
```

```statsd tab="StatsD"
# Default prefix: "traefik"
{prefix}.service.server.up
```
