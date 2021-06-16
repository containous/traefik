# Datadog

To enable the Datadog:

```toml tab="File (TOML)"
[metrics]
  [metrics.datadog]
```

```yaml tab="File (YAML)"
metrics:
  datadog: {}
```

```bash tab="CLI"
--metrics.datadog=true
```

#### `address`

_Required, Default="127.0.0.1:8125"_

Address instructs exporter to send metrics to datadog-agent at this address.

```toml tab="File (TOML)"
[metrics]
  [metrics.datadog]
    address = "127.0.0.1:8125"
```

```yaml tab="File (YAML)"
metrics:
  datadog:
    address: 127.0.0.1:8125
```

```bash tab="CLI"
--metrics.datadog.address=127.0.0.1:8125
```

#### `addEntryPointsLabels`

_Optional, Default=true_

Enable metrics on entry points.

```toml tab="File (TOML)"
[metrics]
  [metrics.datadog]
    addEntryPointsLabels = true
```

```yaml tab="File (YAML)"
metrics:
  datadog:
    addEntryPointsLabels: true
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

```toml tab="File (TOML)"
[metrics]
  [metrics.datadog]
    addServicesLabels = true
```

```yaml tab="File (YAML)"
metrics:
  datadog:
    addServicesLabels: true
```

```bash tab="CLI"
--metrics.datadog.addServicesLabels=true
```

#### `pushInterval`

_Optional, Default=10s_

The interval used by the exporter to push metrics to datadog-agent.

```toml tab="File (TOML)"
[metrics]
  [metrics.datadog]
    pushInterval = 10s
```

```yaml tab="File (YAML)"
metrics:
  datadog:
    pushInterval: 10s
```

```bash tab="CLI"
--metrics.datadog.pushInterval=10s
```

