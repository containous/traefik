# InfluxDB 2

To enable the InfluxDB 2:

```yaml tab="File (YAML)"
metrics:
  influxDB2:
    token: "<token>"
    bucket: "my-bucket"
    org: "my-org"
```

```toml tab="File (TOML)"
[metrics]
  [metrics.influxDB2]
    token = "<token>"
    bucket = "my-bucket"
    org = "my-org"
```

```bash tab="CLI"
--metrics.influxdb2=true --metrics.influxdb2.token="<token>" --metrics.influxdb2.bucket="my-bucket" --metrics.influxdb2.org="my-org"
```

#### `address`

_Required, Default="http://localhost:8086"_

Address of InfluxDB 2 instance.

```yaml tab="File (YAML)"
metrics:
  influxDB2:
    address: http://localhost:8086
```

```toml tab="File (TOML)"
[metrics]
  [metrics.influxDB2]
    address = "http://localhost:8086"
```

```bash tab="CLI"
--metrics.influxdb2.address=http://localhost:8086
```

#### `token`

_Required, Default=""_

InfluxDB token (can be obtained from InfluxDB instance web UI). Mandatory for connection.

```yaml tab="File (YAML)"
metrics:
  influxDB2:
    token: "secret"
```

```toml tab="File (TOML)"
[metrics]
  [metrics.influxDB2]
    token = "secret"
```

```bash tab="CLI"
--metrics.influxdb2.token="secret"
```

#### `org`

_Required, Default=""_

InfluxDB org where Traefik metrics will be stored. Must not be left blank, necessary for connection.

```yaml tab="File (YAML)"
metrics:
  influxDB2:
    org: "my-org"
```

```toml tab="File (TOML)"
[metrics]
  [metrics.influxDB2]
    org = "my-org"
```

```bash tab="CLI"
--metrics.influxdb2.org="my-org"
```

#### `bucket`

_Required, Default=""_

InfluxDB bucket where Traefik metrics will be stored. Must not be left blank, necessary for connection.

```yaml tab="File (YAML)"
metrics:
  influxDB2:
    bucket: "my-bucket"
```

```toml tab="File (TOML)"
[metrics]
  [metrics.influxDB2]
    bucket = "my-bucket"
```

```bash tab="CLI"
--metrics.influxdb2.bucket="my-bucket"
```

#### `addEntryPointsLabels`

_Optional, Default=true_

Enable metrics on entry points.

```yaml tab="File (YAML)"
metrics:
  influxDB2:
    addEntryPointsLabels: true
```

```toml tab="File (TOML)"
[metrics]
  [metrics.influxDB2]
    addEntryPointsLabels = true
```

```bash tab="CLI"
--metrics.influxdb2.addEntryPointsLabels=true
```

#### `addRoutersLabels`

_Optional, Default=false_

Enable metrics on routers.

```toml tab="File (TOML)"
[metrics]
  [metrics.influxDB2]
    addRoutersLabels = true
```

```yaml tab="File (YAML)"
metrics:
  influxDB2:
    addRoutersLabels: true
```

```bash tab="CLI"
--metrics.influxdb2.addrouterslabels=true
```

#### `addServicesLabels`

_Optional, Default=true_

Enable metrics on services.

```yaml tab="File (YAML)"
metrics:
  influxDB2:
    addServicesLabels: true
```

```toml tab="File (TOML)"
[metrics]
  [metrics.influxDB2]
    addServicesLabels = true
```

```bash tab="CLI"
--metrics.influxdb2.addServicesLabels=true
```

#### `batchSize`

_Optional, Default=10_

Number of metrics reports collected before pushing to InfluxDB.

```yaml tab="File (YAML)"
metrics:
  influxDB2:
    batchSize: 10
```

```toml tab="File (TOML)"
[metrics]
  [metrics.influxDB2]
    batchSize = 10
```

```bash tab="CLI"
--metrics.influxdb2.batchSize=10
```

#### `pushInterval`

_Optional, Default=30s_

Interval at which Traefik will save metrics to InfluxDB. Even if less than `BatchSize` metrics were collected, they will be sent to InfluxDB.

```yaml tab="File (YAML)"
metrics:
  influxDB2:
    pushInterval: 30s
```

```toml tab="File (TOML)"
[metrics]
  [metrics.influxDB2]
    pushInterval = 30s
```

```bash tab="CLI"
--metrics.influxdb2.pushInterval=30s
```
