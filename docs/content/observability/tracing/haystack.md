# Haystack

To enable the Haystack:

```yaml tab="File (YAML)"
tracing:
  haystack: {}
```

```toml tab="File (TOML)"
[tracing]
  [tracing.haystack]
```

```bash tab="CLI"
--tracing.haystack=true
```

#### `localAgentHost`

_Require, Default="127.0.0.1"_

Local Agent Host instructs reporter to send spans to haystack-agent at this address.

```yaml tab="File (YAML)"
tracing:
  haystack:
    localAgentHost: 127.0.0.1
```

```toml tab="File (TOML)"
[tracing]
  [tracing.haystack]
    localAgentHost = "127.0.0.1"
```

```bash tab="CLI"
--tracing.haystack.localAgentHost=127.0.0.1
```

#### `localAgentPort`

_Require, Default=35000_

Local Agent port instructs reporter to send spans to the haystack-agent at this port.

```yaml tab="File (YAML)"
tracing:
  haystack:
    localAgentPort: 35000
```

```toml tab="File (TOML)"
[tracing]
  [tracing.haystack]
    localAgentPort = 35000
```

```bash tab="CLI"
--tracing.haystack.localAgentPort=35000
```

#### `globalTag`

_Optional, Default=empty_

Apply shared tag in a form of Key:Value to all the traces.

```yaml tab="File (YAML)"
tracing:
  haystack:
    globalTag: sample:test
```

```toml tab="File (TOML)"
[tracing]
  [tracing.haystack]
    globalTag = "sample:test"
```

```bash tab="CLI"
--tracing.haystack.globalTag=sample:test
```

#### `traceIDHeaderName`

_Optional, Default=empty_

Specifies the header name that will be used to store the trace ID.

```yaml tab="File (YAML)"
tracing:
  haystack:
    traceIDHeaderName: Trace-ID
```

```toml tab="File (TOML)"
[tracing]
  [tracing.haystack]
    traceIDHeaderName = "Trace-ID"
```

```bash tab="CLI"
--tracing.haystack.traceIDHeaderName=Trace-ID
```

#### `parentIDHeaderName`

_Optional, Default=empty_

Specifies the header name that will be used to store the parent ID.

```yaml tab="File (YAML)"
tracing:
  haystack:
    parentIDHeaderName: Parent-Message-ID
```

```toml tab="File (TOML)"
[tracing]
  [tracing.haystack]
    parentIDHeaderName = "Parent-Message-ID"
```

```bash tab="CLI"
--tracing.haystack.parentIDHeaderName=Parent-Message-ID
```

#### `spanIDHeaderName`

_Optional, Default=empty_

Specifies the header name that will be used to store the span ID.

```yaml tab="File (YAML)"
tracing:
  haystack:
    spanIDHeaderName: Message-ID
```

```toml tab="File (TOML)"
[tracing]
  [tracing.haystack]
    spanIDHeaderName = "Message-ID"
```

```bash tab="CLI"
--tracing.haystack.spanIDHeaderName=Message-ID
```

#### `baggagePrefixHeaderName`

_Optional, Default=empty_

Specifies the header name prefix that will be used to store baggage items in a map.

```yaml tab="File (YAML)"
tracing:
  haystack:
    baggagePrefixHeaderName: "sample"
```

```toml tab="File (TOML)"
[tracing]
  [tracing.haystack]
    baggagePrefixHeaderName = "sample"
```


```bash tab="CLI"
--tracing.haystack.baggagePrefixHeaderName=sample
```
