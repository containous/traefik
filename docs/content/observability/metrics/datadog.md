# Datadog

To enable the Datadog:

```yaml tab="File (YAML)"
metrics:
  datadog: {}
```

```toml tab="File (TOML)"
[metrics]
  [metrics.datadog]
```

```bash tab="CLI"
--metrics.datadog=true
```

#### `address`

_Required, Default="127.0.0.1:8125"_

Address instructs exporter to send metrics to datadog-agent at this address.

```yaml tab="File (YAML)"
metrics:
  datadog:
    address: 127.0.0.1:8125
```

```toml tab="File (TOML)"
[metrics]
  [metrics.datadog]
    address = "127.0.0.1:8125"
```

```bash tab="CLI"
--metrics.datadog.address=127.0.0.1:8125
```

#### `addEntryPointsLabels`

_Optional, Default=true_

Enable metrics on entry points.

```yaml tab="File (YAML)"
metrics:
  datadog:
    addEntryPointsLabels: true
```

```toml tab="File (TOML)"
[metrics]
  [metrics.datadog]
    addEntryPointsLabels = true
```

```bash tab="CLI"
--metrics.datadog.addEntryPointsLabels=true
```
#### `AddRoutersLabels`

_Optional, Default=false_

Enable metrics on routers.

```toml tab="File (TOML)"
[metrics]
  [metrics.datadog]
    addRoutersLabels = true
```

```yaml tab="File (YAML)"
metrics:
  datadog:
    addRoutersLabels: true
```

```bash tab="CLI"
--metrics.datadog.addrouterslabels=true
```

#### `addServicesLabels`

_Optional, Default=true_

Enable metrics on services.

```yaml tab="File (YAML)"
metrics:
  datadog:
    addServicesLabels: true
```

```toml tab="File (TOML)"
[metrics]
  [metrics.datadog]
    addServicesLabels = true
```

```bash tab="CLI"
--metrics.datadog.addServicesLabels=true
```

#### `pushInterval`

_Optional, Default=10s_

The interval used by the exporter to push metrics to datadog-agent.

```yaml tab="File (YAML)"
metrics:
  datadog:
    pushInterval: 10s
```

```toml tab="File (TOML)"
[metrics]
  [metrics.datadog]
    pushInterval = 10s
```

```bash tab="CLI"
--metrics.datadog.pushInterval=10s
```

