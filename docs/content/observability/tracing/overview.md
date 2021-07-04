# Tracing

Visualize the Requests Flow
{: .subtitle }

The tracing system allows developers to visualize call flows in their infrastructure.

Traefik uses OpenTracing, an open standard designed for distributed tracing.

Traefik supports six tracing backends:

- [Jaeger](./jaeger.md)
- [Zipkin](./zipkin.md)
- [Datadog](./datadog.md)
- [Instana](./instana.md)
- [Haystack](./haystack.md)
- [Elastic](./elastic.md)

## Configuration

By default, Traefik uses Jaeger as tracing backend.

To enable the tracing:

```yaml tab="File (YAML)"
tracing: {}
```

```toml tab="File (TOML)"
[tracing]
```

```bash tab="CLI"
--tracing=true
```

### Common Options

#### `serviceName`

_Required, Default="traefik"_

Service name used in selected backend.

```yaml tab="File (YAML)"
tracing:
  serviceName: traefik
```

```toml tab="File (TOML)"
[tracing]
  serviceName = "traefik"
```

```bash tab="CLI"
--tracing.serviceName=traefik
```

#### `spanNameLimit`

_Required, Default=0_

Span name limit allows for name truncation in case of very long names.
This can prevent certain tracing providers to drop traces that exceed their length limits.

`0` means no truncation will occur.

```yaml tab="File (YAML)"
tracing:
  spanNameLimit: 150
```

```toml tab="File (TOML)"
[tracing]
  spanNameLimit = 150
```

```bash tab="CLI"
--tracing.spanNameLimit=150
```
