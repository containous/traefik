# StatsD

To enable the Statsd:

```toml tab="File (TOML)"
[metrics]
  [metrics.statsD]
```

```yaml tab="File (YAML)"
metrics:
  statsD: {}
```

```bash tab="CLI"
--metrics.statsd=true
```

#### `address`

_Required, Default="localhost:8125"_

Address instructs exporter to send metrics to statsd at this address.

```toml tab="File (TOML)"
[metrics]
  [metrics.statsD]
    address = "localhost:8125"
```

```yaml tab="File (YAML)"
metrics:
  statsD:
    address: localhost:8125
```

```bash tab="CLI"
--metrics.statsd.address=localhost:8125
```

#### `addEntryPointsLabels`

_Optional, Default=true_

Enable metrics on entry points.

```toml tab="File (TOML)"
[metrics]
  [metrics.statsD]
    addEntryPointsLabels = true
```

```yaml tab="File (YAML)"
metrics:
  statsD:
    addEntryPointsLabels: true
```

```bash tab="CLI"
--metrics.statsd.addEntryPointsLabels=true
```

#### `AddRoutersLabels`

_Optional, Default=false_

Enable metrics on entry points.

```toml tab="File (TOML)"
[metrics]
  [metrics.statsD]
    addRoutersLabels = true
```

```yaml tab="File (YAML)"
metrics:
  statsD:
    addRoutersLabels: true
```

```bash tab="CLI"
--metrics.statsd.addrouterslabels=true
```

#### `addServicesLabels`

_Optional, Default=true_

Enable metrics on services.

```toml tab="File (TOML)"
[metrics]
  [metrics.statsD]
    addServicesLabels = true
```

```yaml tab="File (YAML)"
metrics:
  statsD:
    addServicesLabels: true
```

```bash tab="CLI"
--metrics.statsd.addServicesLabels=true
```

#### `pushInterval`

_Optional, Default=10s_

The interval used by the exporter to push metrics to statsD.

```toml tab="File (TOML)"
[metrics]
  [metrics.statsD]
    pushInterval = 10s
```

```yaml tab="File (YAML)"
metrics:
  statsD:
    pushInterval: 10s
```

```bash tab="CLI"
--metrics.statsd.pushInterval=10s
```

#### `prefix`

_Optional, Default="traefik"_

The prefix to use for metrics collection.

```toml tab="File (TOML)"
[metrics]
  [metrics.statsD]
    prefix = "traefik"
```

```yaml tab="File (YAML)"
metrics:
  statsD:
    prefix: traefik
```

```bash tab="CLI"
--metrics.statsd.prefix="traefik"
```
