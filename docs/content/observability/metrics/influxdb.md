# InfluxDB

To enable the InfluxDB:

```yaml tab="File (YAML)"
metrics:
  influxDB: {}
```

```toml tab="File (TOML)"
[metrics]
  [metrics.influxDB]
```

```bash tab="CLI"
--metrics.influxdb=true
```

#### `address`

_Required, Default="localhost:8089"_

Address instructs exporter to send metrics to influxdb at this address.

```yaml tab="File (YAML)"
metrics:
  influxDB:
    address: localhost:8089
```

```toml tab="File (TOML)"
[metrics]
  [metrics.influxDB]
    address = "localhost:8089"
```

```bash tab="CLI"
--metrics.influxdb.address=localhost:8089
```

#### `protocol`

_Required, Default="udp"_

InfluxDB's address protocol (udp or http).

```yaml tab="File (YAML)"
metrics:
  influxDB:
    protocol: udp
```

```toml tab="File (TOML)"
[metrics]
  [metrics.influxDB]
    protocol = "udp"
```

```bash tab="CLI"
--metrics.influxdb.protocol=udp
```

#### `database`

_Optional, Default=""_

InfluxDB database used when protocol is http.

```yaml tab="File (YAML)"
metrics:
  influxDB:
    database: "db"
```

```toml tab="File (TOML)"
[metrics]
  [metrics.influxDB]
    database = "db"
```

```bash tab="CLI"
--metrics.influxdb.database=db
```

#### `retentionPolicy`

_Optional, Default=""_

InfluxDB retention policy used when protocol is http.

```yaml tab="File (YAML)"
metrics:
  influxDB:
    retentionPolicy: "two_hours"
```

```toml tab="File (TOML)"
[metrics]
  [metrics.influxDB]
    retentionPolicy = "two_hours"
```

```bash tab="CLI"
--metrics.influxdb.retentionPolicy=two_hours
```

#### `username`

_Optional, Default=""_

InfluxDB username (only with http).

```yaml tab="File (YAML)"
metrics:
  influxDB:
    username: "john"
```

```toml tab="File (TOML)"
[metrics]
  [metrics.influxDB]
    username = "john"
```

```bash tab="CLI"
--metrics.influxdb.username=john
```

#### `password`

_Optional, Default=""_

InfluxDB password (only with http).

```yaml tab="File (YAML)"
metrics:
  influxDB:
    password: "secret"
```

```toml tab="File (TOML)"
[metrics]
  [metrics.influxDB]
    password = "secret"
```

```bash tab="CLI"
--metrics.influxdb.password=secret
```

#### `addEntryPointsLabels`

_Optional, Default=true_

Enable metrics on entry points.

```yaml tab="File (YAML)"
metrics:
  influxDB:
    addEntryPointsLabels: true
```

```toml tab="File (TOML)"
[metrics]
  [metrics.influxDB]
    addEntryPointsLabels = true
```

```bash tab="CLI"
--metrics.influxdb.addEntryPointsLabels=true
```

#### `AddRoutersLabels`

_Optional, Default=false_

Enable metrics on routers.

```toml tab="File (TOML)"
[metrics]
  [metrics.influxDB]
    addRoutersLabels = true
```

```yaml tab="File (YAML)"
metrics:
  influxDB:
    addRoutersLabels: true
```

```bash tab="CLI"
--metrics.influxdb.addrouterslabels=true
```

#### `addServicesLabels`

_Optional, Default=true_

Enable metrics on services.

```yaml tab="File (YAML)"
metrics:
  influxDB:
    addServicesLabels: true
```

```toml tab="File (TOML)"
[metrics]
  [metrics.influxDB]
    addServicesLabels = true
```

```bash tab="CLI"
--metrics.influxdb.addServicesLabels=true
```

#### `pushInterval`

_Optional, Default=10s_

The interval used by the exporter to push metrics to influxdb.

```yaml tab="File (YAML)"
metrics:
  influxDB:
    pushInterval: 10s
```

```toml tab="File (TOML)"
[metrics]
  [metrics.influxDB]
    pushInterval = 10s
```

```bash tab="CLI"
--metrics.influxdb.pushInterval=10s
```
