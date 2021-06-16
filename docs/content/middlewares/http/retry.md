# Retry

Retrying until it Succeeds
{: .subtitle }

<!--
TODO: add schema
-->

The Retry middleware reissues requests a given number of times to a backend server if that server does not reply.
As soon as the server answers, the middleware stops retrying, regardless of the response status.
The Retry middleware has an optional configuration to enable an exponential backoff.

## Configuration Examples

```yaml tab="Docker"
# Retry 4 times with exponential backoff
labels:
  - "traefik.http.middlewares.test-retry.retry.attempts=4"
  - "traefik.http.middlewares.test-retry.retry.initialinterval=100ms"
```

```yaml tab="Kubernetes"
# Retry 4 times with exponential backoff
apiVersion: traefik.containo.us/v1alpha1
kind: Middleware
metadata:
  name: test-retry
spec:
  retry:
    attempts: 4
    initialInterval: 100ms
```

```yaml tab="Consul Catalog"
# Retry 4 times with exponential backoff
- "traefik.http.middlewares.test-retry.retry.attempts=4"
- "traefik.http.middlewares.test-retry.retry.initialinterval=100ms"
```

```json tab="Marathon"
"labels": {
  "traefik.http.middlewares.test-retry.retry.attempts": "4",
  "traefik.http.middlewares.test-retry.retry.initialinterval": "100ms",
}
```

```yaml tab="Rancher"
# Retry 4 times with exponential backoff
labels:
  - "traefik.http.middlewares.test-retry.retry.attempts=4"
  - "traefik.http.middlewares.test-retry.retry.initialinterval=100ms"
```

```toml tab="File (TOML)"
# Retry 4 times with exponential backoff
[http.middlewares]
  [http.middlewares.test-retry.retry]
    attempts = 4
    initialInterval = "100ms"
```

```yaml tab="File (YAML)"
# Retry 4 times with exponential backoff
http:
  middlewares:
    test-retry:
      retry:
        attempts: 4
        initialInterval: 100ms
```

## Configuration Options

### `attempts`

_mandatory_

The `attempts` option defines how many times the request should be retried.

### `initialInterval`

The `initialInterval` option defines the first wait time in the exponential backoff series. The maximum interval is
calculated as twice the `initialInterval`. If unspecified, requests will be retried immediately.

The value of initialInterval should be provided in seconds or as a valid duration format, see [time.ParseDuration](https://golang.org/pkg/time/#ParseDuration).
