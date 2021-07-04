# Zipkin

To enable the Zipkin:

```yaml tab="File (YAML)"
tracing:
  zipkin: {}
```

```toml tab="File (TOML)"
[tracing]
  [tracing.zipkin]
```

```bash tab="CLI"
--tracing.zipkin=true
```

#### `httpEndpoint`

_Required, Default="http://localhost:9411/api/v2/spans"_

Zipkin HTTP endpoint used to send data.

```yaml tab="File (YAML)"
tracing:
  zipkin:
    httpEndpoint: http://localhost:9411/api/v2/spans
```

```toml tab="File (TOML)"
[tracing]
  [tracing.zipkin]
    httpEndpoint = "http://localhost:9411/api/v2/spans"
```

```bash tab="CLI"
--tracing.zipkin.httpEndpoint=http://localhost:9411/api/v2/spans
```

#### `sameSpan`

_Optional, Default=false_

Use Zipkin SameSpan RPC style traces.

```yaml tab="File (YAML)"
tracing:
  zipkin:
    sameSpan: true
```

```toml tab="File (TOML)"
[tracing]
  [tracing.zipkin]
    sameSpan = true
```

```bash tab="CLI"
--tracing.zipkin.sameSpan=true
```

#### `id128Bit`

_Optional, Default=true_

Use Zipkin 128 bit trace IDs.

```yaml tab="File (YAML)"
tracing:
  zipkin:
    id128Bit: false
```

```toml tab="File (TOML)"
[tracing]
  [tracing.zipkin]
    id128Bit = false
```

```bash tab="CLI"
--tracing.zipkin.id128Bit=false
```

#### `sampleRate`

_Required, Default=1.0_

The rate between 0.0 and 1.0 of requests to trace.

```yaml tab="File (YAML)"
tracing:
  zipkin:
    sampleRate: 0.2
```

```toml tab="File (TOML)"
[tracing]
  [tracing.zipkin]
    sampleRate = 0.2
```

```bash tab="CLI"
--tracing.zipkin.sampleRate=0.2
```
