# Prometheus

To enable the Prometheus:

```yaml tab="File (YAML)"
metrics:
  prometheus: {}
```

```toml tab="File (TOML)"
[metrics]
  [metrics.prometheus]
```

```bash tab="CLI"
--metrics.prometheus=true
```

#### `buckets`

_Optional, Default="0.100000, 0.300000, 1.200000, 5.000000"_

Buckets for latency metrics.

```yaml tab="File (YAML)"
metrics:
  prometheus:
    buckets:
      - 0.1
      - 0.3
      - 1.2
      - 5.0
```

```toml tab="File (TOML)"
[metrics]
  [metrics.prometheus]
    buckets = [0.1,0.3,1.2,5.0]
```

```bash tab="CLI"
--metrics.prometheus.buckets=0.100000, 0.300000, 1.200000, 5.000000
```

#### `addEntryPointsLabels`

_Optional, Default=true_

Enable metrics on entry points.

```yaml tab="File (YAML)"
metrics:
  prometheus:
    addEntryPointsLabels: true
```

```toml tab="File (TOML)"
[metrics]
  [metrics.prometheus]
    addEntryPointsLabels = true
```

```bash tab="CLI"
--metrics.prometheus.addEntryPointsLabels=true
```

#### `AddRoutersLabels`

_Optional, Default=false_

Enable metrics on routers.

```toml tab="File (TOML)"
[metrics]
  [metrics.prometheus]
    addRoutersLabels = true
```

```yaml tab="File (YAML)"
metrics:
  prometheus:
    addRoutersLabels: true
```

```bash tab="CLI"
--metrics.prometheus.addrouterslabels=true
```

#### `addServicesLabels`

_Optional, Default=true_

Enable metrics on services.

```yaml tab="File (YAML)"
metrics:
  prometheus:
    addServicesLabels: true
```

```toml tab="File (TOML)"
[metrics]
  [metrics.prometheus]
    addServicesLabels = true
```

```bash tab="CLI"
--metrics.prometheus.addServicesLabels=true
```

#### `entryPoint`

_Optional, Default=traefik_

Entry point used to expose metrics.

```yaml tab="File (YAML)"
entryPoints:
  metrics:
    address: ":8082"

metrics:
  prometheus:
    entryPoint: metrics
```

```toml tab="File (TOML)"
[entryPoints]
  [entryPoints.metrics]
    address = ":8082"

[metrics]
  [metrics.prometheus]
    entryPoint = "metrics"
```

```bash tab="CLI"
--entryPoints.metrics.address=:8082
--metrics.prometheus.entryPoint=metrics
```

#### `manualRouting`

_Optional, Default=false_

If `manualRouting` is `true`, it disables the default internal router in order to allow one to create a custom router for the `prometheus@internal` service.

```yaml tab="File (YAML)"
metrics:
  prometheus:
    manualRouting: true
```

```toml tab="File (TOML)"
[metrics]
  [metrics.prometheus]
    manualRouting = true
```

```bash tab="CLI"
--metrics.prometheus.manualrouting=true
```
