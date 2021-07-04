# Datadog

To enable the Datadog:

```yaml tab="File (YAML)"
tracing:
  datadog: {}
```

```toml tab="File (TOML)"
[tracing]
  [tracing.datadog]
```

```bash tab="CLI"
--tracing.datadog=true
```

#### `localAgentHostPort`

_Required, Default="127.0.0.1:8126"_

Local Agent Host Port instructs reporter to send spans to datadog-tracing-agent at this address.

```yaml tab="File (YAML)"
tracing:
  datadog:
    localAgentHostPort: 127.0.0.1:8126
```

```toml tab="File (TOML)"
[tracing]
  [tracing.datadog]
    localAgentHostPort = "127.0.0.1:8126"
```

```bash tab="CLI"
--tracing.datadog.localAgentHostPort=127.0.0.1:8126
```

#### `debug`

_Optional, Default=false_

Enable Datadog debug.

```yaml tab="File (YAML)"
tracing:
  datadog:
    debug: true
```

```toml tab="File (TOML)"
[tracing]
  [tracing.datadog]
    debug = true
```

```bash tab="CLI"
--tracing.datadog.debug=true
```

#### `globalTag`

_Optional, Default=empty_

Apply shared tag in a form of Key:Value to all the traces.

```yaml tab="File (YAML)"
tracing:
  datadog:
    globalTag: sample
```

```toml tab="File (TOML)"
[tracing]
  [tracing.datadog]
    globalTag = "sample"
```

```bash tab="CLI"
--tracing.datadog.globalTag=sample
```

#### `prioritySampling`

_Optional, Default=false_

Enable priority sampling. When using distributed tracing,
this option must be enabled in order to get all the parts of a distributed trace sampled.

```yaml tab="File (YAML)"
tracing:
  datadog:
    prioritySampling: true
```

```toml tab="File (TOML)"
[tracing]
  [tracing.datadog]
    prioritySampling = true
```

```bash tab="CLI"
--tracing.datadog.prioritySampling=true
```
