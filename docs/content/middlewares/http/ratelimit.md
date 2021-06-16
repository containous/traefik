# RateLimit

To Control the Number of Requests Going to a Service
{: .subtitle }

The RateLimit middleware ensures that services will receive a _fair_ amount of requests, and allows one to define what fair is.

## Configuration Example

```yaml tab="Docker"
# Here, an average of 100 requests per second is allowed.
# In addition, a burst of 50 requests is allowed.
labels:
  - "traefik.http.middlewares.test-ratelimit.ratelimit.average=100"
  - "traefik.http.middlewares.test-ratelimit.ratelimit.burst=50"
```

```yaml tab="Kubernetes"
# Here, an average of 100 requests per second is allowed.
# In addition, a burst of 50 requests is allowed.
apiVersion: traefik.containo.us/v1alpha1
kind: Middleware
metadata:
  name: test-ratelimit
spec:
  rateLimit:
    average: 100
    burst: 50
```

```yaml tab="Consul Catalog"
# Here, an average of 100 requests per second is allowed.
# In addition, a burst of 50 requests is allowed.
- "traefik.http.middlewares.test-ratelimit.ratelimit.average=100"
- "traefik.http.middlewares.test-ratelimit.ratelimit.burst=50"
```

```json tab="Marathon"
"labels": {
  "traefik.http.middlewares.test-ratelimit.ratelimit.average": "100",
  "traefik.http.middlewares.test-ratelimit.ratelimit.burst": "50"
}
```

```yaml tab="Rancher"
# Here, an average of 100 requests per second is allowed.
# In addition, a burst of 50 requests is allowed.
labels:
  - "traefik.http.middlewares.test-ratelimit.ratelimit.average=100"
  - "traefik.http.middlewares.test-ratelimit.ratelimit.burst=50"
```

```toml tab="File (TOML)"
# Here, an average of 100 requests per second is allowed.
# In addition, a burst of 50 requests is allowed.
[http.middlewares]
  [http.middlewares.test-ratelimit.rateLimit]
    average = 100
    burst = 50
```

```yaml tab="File (YAML)"
# Here, an average of 100 requests per second is allowed.
# In addition, a burst of 50 requests is allowed.
http:
  middlewares:
    test-ratelimit:
      rateLimit:
        average: 100
        burst: 50
```

## Configuration Options

### `average`

`average` is the maximum rate, by default in requests per second, allowed from a given source.

It defaults to `0`, which means no rate limiting.

The rate is actually defined by dividing `average` by `period`.
So for a rate below 1 req/s, one needs to define a `period` larger than a second.

```yaml tab="Docker"
# 100 reqs/s
labels:
  - "traefik.http.middlewares.test-ratelimit.ratelimit.average=100"
```

```yaml tab="Kubernetes"
# 100 reqs/s
apiVersion: traefik.containo.us/v1alpha1
kind: Middleware
metadata:
  name: test-ratelimit
spec:
  rateLimit:
    average: 100
```

```yaml tab="Consul Catalog"
# 100 reqs/s
- "traefik.http.middlewares.test-ratelimit.ratelimit.average=100"
```

```json tab="Marathon"
"labels": {
  "traefik.http.middlewares.test-ratelimit.ratelimit.average": "100",
}
```

```yaml tab="Rancher"
labels:
  - "traefik.http.middlewares.test-ratelimit.ratelimit.average=100"
```

```toml tab="File (TOML)"
# 100 reqs/s
[http.middlewares]
  [http.middlewares.test-ratelimit.rateLimit]
    average = 100
```

```yaml tab="File (YAML)"
# 100 reqs/s
http:
  middlewares:
    test-ratelimit:
      rateLimit:
        average: 100
```

### `period`

`period`, in combination with `average`, defines the actual maximum rate, such as:

```go
r = average / period
```

It defaults to `1` second.

```yaml tab="Docker"
# 6 reqs/minute
labels:
  - "traefik.http.middlewares.test-ratelimit.ratelimit.average=6"
  - "traefik.http.middlewares.test-ratelimit.ratelimit.period=1m"
```

```yaml tab="Kubernetes"
# 6 reqs/minute
apiVersion: traefik.containo.us/v1alpha1
kind: Middleware
metadata:
  name: test-ratelimit
spec:
  rateLimit:
    period: 1m
    average: 6
```

```yaml tab="Consul Catalog"
# 6 reqs/minute
- "traefik.http.middlewares.test-ratelimit.ratelimit.average=6"
- "traefik.http.middlewares.test-ratelimit.ratelimit.period=1m"
```

```json tab="Marathon"
"labels": {
  "traefik.http.middlewares.test-ratelimit.ratelimit.average": "6",
  "traefik.http.middlewares.test-ratelimit.ratelimit.period": "1m",
}
```

```yaml tab="Rancher"
# 6 reqs/minute
labels:
  - "traefik.http.middlewares.test-ratelimit.ratelimit.average=6"
  - "traefik.http.middlewares.test-ratelimit.ratelimit.period=1m"
```

```toml tab="File (TOML)"
# 6 reqs/minute
[http.middlewares]
  [http.middlewares.test-ratelimit.rateLimit]
    average = 6
    period = "1m"
```

```yaml tab="File (YAML)"
# 6 reqs/minute
http:
  middlewares:
    test-ratelimit:
      rateLimit:
        average: 6
        period: 1m
```

### `burst`

`burst` is the maximum number of requests allowed to go through in the same arbitrarily small period of time.

It defaults to `1`.

```yaml tab="Docker"
labels:
  - "traefik.http.middlewares.test-ratelimit.ratelimit.burst=100"
```

```yaml tab="Kubernetes"
apiVersion: traefik.containo.us/v1alpha1
kind: Middleware
metadata:
  name: test-ratelimit
spec:
  rateLimit:
    burst: 100
```

```yaml tab="Consul Catalog"
- "traefik.http.middlewares.test-ratelimit.ratelimit.burst=100"
```

```json tab="Marathon"
"labels": {
  "traefik.http.middlewares.test-ratelimit.ratelimit.burst": "100",
}
```

```yaml tab="Rancher"
labels:
  - "traefik.http.middlewares.test-ratelimit.ratelimit.burst=100"
```

```toml tab="File (TOML)"
[http.middlewares]
  [http.middlewares.test-ratelimit.rateLimit]
    burst = 100
```

```yaml tab="File (YAML)"
http:
  middlewares:
    test-ratelimit:
      rateLimit:
        burst: 100
```

### `sourceCriterion`

The `sourceCriterion` option defines what criterion is used to group requests as originating from a common source.
The precedence order is `ipStrategy`, then `requestHeaderName`, then `requestHost`.
If none are set, the default is to use the request's remote address field (as an `ipStrategy`).

#### `sourceCriterion.ipStrategy`

The `ipStrategy` option defines two parameters that configures how Traefik determines the client IP: `depth`, and `excludedIPs`.

##### `ipStrategy.depth`

The `depth` option tells Traefik to use the `X-Forwarded-For` header and select the IP located at the `depth` position (starting from the right).

- If `depth` is greater than the total number of IPs in `X-Forwarded-For`, then the client IP is empty.
- `depth` is ignored if its value is less than or equal to 0.

!!! example "Example of Depth & X-Forwarded-For"

    If `depth` is set to 2, and the request `X-Forwarded-For` header is `"10.0.0.1,11.0.0.1,12.0.0.1,13.0.0.1"` then the "real" client IP is `"10.0.0.1"` (at depth 4) but the IP used as the criterion is `"12.0.0.1"` (`depth=2`).

    | `X-Forwarded-For`                       | `depth` | clientIP     |
    |-----------------------------------------|---------|--------------|
    | `"10.0.0.1,11.0.0.1,12.0.0.1,13.0.0.1"` | `1`     | `"13.0.0.1"` |
    | `"10.0.0.1,11.0.0.1,12.0.0.1,13.0.0.1"` | `3`     | `"11.0.0.1"` |
    | `"10.0.0.1,11.0.0.1,12.0.0.1,13.0.0.1"` | `5`     | `""`         |

```yaml tab="Docker"
labels:
  - "traefik.http.middlewares.test-ratelimit.ratelimit.sourcecriterion.ipstrategy.depth=2"
```

```yaml tab="Kubernetes"
apiVersion: traefik.containo.us/v1alpha1
kind: Middleware
metadata:
  name: test-ratelimit
spec:
  rateLimit:
    sourceCriterion:
      ipStrategy:
        depth: 2
```

```yaml tab="Consul Catalog"
- "traefik.http.middlewares.test-ratelimit.ratelimit.sourcecriterion.ipstrategy.depth=2"
```

```json tab="Marathon"
"labels": {
  "traefik.http.middlewares.test-ratelimit.ratelimit.sourcecriterion.ipstrategy.depth": "2"
}
```

```yaml tab="Rancher"
labels:
  - "traefik.http.middlewares.test-ratelimit.ratelimit.sourcecriterion.ipstrategy.depth=2"
```

```toml tab="File (TOML)"
[http.middlewares]
  [http.middlewares.test-ratelimit.rateLimit]
    [http.middlewares.test-ratelimit.rateLimit.sourceCriterion.ipStrategy]
      depth = 2
```

```yaml tab="File (YAML)"
http:
  middlewares:
    test-ratelimit:
      rateLimit:
        sourceCriterion:
          ipStrategy:
            depth: 2
```

##### `ipStrategy.excludedIPs`

`excludedIPs` configures Traefik to scan the `X-Forwarded-For` header and select the first IP not in the list.

!!! important "If `depth` is specified, `excludedIPs` is ignored."

!!! example "Example of ExcludedIPs & X-Forwarded-For"

    | `X-Forwarded-For`                       | `excludedIPs`         | clientIP     |
    |-----------------------------------------|-----------------------|--------------|
    | `"10.0.0.1,11.0.0.1,12.0.0.1,13.0.0.1"` | `"12.0.0.1,13.0.0.1"` | `"11.0.0.1"` |
    | `"10.0.0.1,11.0.0.1,12.0.0.1,13.0.0.1"` | `"15.0.0.1,13.0.0.1"` | `"12.0.0.1"` |
    | `"10.0.0.1,11.0.0.1,12.0.0.1,13.0.0.1"` | `"10.0.0.1,13.0.0.1"` | `"12.0.0.1"` |
    | `"10.0.0.1,11.0.0.1,12.0.0.1,13.0.0.1"` | `"15.0.0.1,16.0.0.1"` | `"13.0.0.1"` |
    | `"10.0.0.1,11.0.0.1"`                   | `"10.0.0.1,11.0.0.1"` | `""`         |

```yaml tab="Docker"
labels:
  - "traefik.http.middlewares.test-ratelimit.ratelimit.sourcecriterion.ipstrategy.excludedips=127.0.0.1/32, 192.168.1.7"
```

```yaml tab="Kubernetes"
apiVersion: traefik.containo.us/v1alpha1
kind: Middleware
metadata:
  name: test-ratelimit
spec:
  rateLimit:
    sourceCriterion:
      ipStrategy:
        excludedIPs:
        - 127.0.0.1/32
        - 192.168.1.7
```

```yaml tab="Consul Catalog"
- "traefik.http.middlewares.test-ratelimit.ratelimit.sourcecriterion.ipstrategy.excludedips=127.0.0.1/32, 192.168.1.7"
```

```json tab="Marathon"
"labels": {
  "traefik.http.middlewares.test-ratelimit.ratelimit.sourcecriterion.ipstrategy.excludedips": "127.0.0.1/32, 192.168.1.7"
}
```

```yaml tab="Rancher"
labels:
  - "traefik.http.middlewares.test-ratelimit.ratelimit.sourcecriterion.ipstrategy.excludedips=127.0.0.1/32, 192.168.1.7"
```

```toml tab="File (TOML)"
[http.middlewares]
  [http.middlewares.test-ratelimit.rateLimit]
    [http.middlewares.test-ratelimit.rateLimit.sourceCriterion.ipStrategy]
      excludedIPs = ["127.0.0.1/32", "192.168.1.7"]
```

```yaml tab="File (YAML)"
http:
  middlewares:
    test-ratelimit:
      rateLimit:
        sourceCriterion:
          ipStrategy:
            excludedIPs:
              - "127.0.0.1/32"
              - "192.168.1.7"
```

#### `sourceCriterion.requestHeaderName`

Name of the header used to group incoming requests.

```yaml tab="Docker"
labels:
  - "traefik.http.middlewares.test-ratelimit.ratelimit.sourcecriterion.requestheadername=username"
```

```yaml tab="Kubernetes"
apiVersion: traefik.containo.us/v1alpha1
kind: Middleware
metadata:
  name: test-ratelimit
spec:
  rateLimit:
	sourceCriterion:
      requestHeaderName: username
```

```yaml tab="Consul Catalog"
- "traefik.http.middlewares.test-ratelimit.ratelimit.sourcecriterion.requestheadername=username"
```

```json tab="Marathon"
"labels": {
  "traefik.http.middlewares.test-ratelimit.ratelimit.sourcecriterion.requestheadername": "username"
}
```

```yaml tab="Rancher"
labels:
  - "traefik.http.middlewares.test-ratelimit.ratelimit.sourcecriterion.requestheadername=username"
```

```toml tab="File (TOML)"
[http.middlewares]
  [http.middlewares.test-ratelimit.rateLimit]
    [http.middlewares.test-ratelimit.rateLimit.sourceCriterion]
      requestHeaderName = "username"
```

```yaml tab="File (YAML)"
http:
  middlewares:
    test-ratelimit:
      rateLimit:
        sourceCriterion:
          requestHeaderName: username
```

#### `sourceCriterion.requestHost`

Whether to consider the request host as the source.

```yaml tab="Docker"
labels:
  - "traefik.http.middlewares.test-ratelimit.ratelimit.sourcecriterion.requesthost=true"
```

```yaml tab="Kubernetes"
apiVersion: traefik.containo.us/v1alpha1
kind: Middleware
metadata:
  name: test-ratelimit
spec:
  rateLimit:
    sourceCriterion:
      requestHost: true
```

```yaml tab="Consul Catalog"
- "traefik.http.middlewares.test-ratelimit.ratelimit.sourcecriterion.requesthost=true"
```

```json tab="Marathon"
"labels": {
  "traefik.http.middlewares.test-ratelimit.ratelimit.sourcecriterion.requesthost": "true"
}
```

```yaml tab="Rancher"
labels:
  - "traefik.http.middlewares.test-ratelimit.ratelimit.sourcecriterion.requesthost=true"
```

```toml tab="File (TOML)"
[http.middlewares]
  [http.middlewares.test-ratelimit.rateLimit]
    [http.middlewares.test-ratelimit.rateLimit.sourceCriterion]
      requestHost = true
```

```yaml tab="File (YAML)"
http:
  middlewares:
    test-ratelimit:
      rateLimit:
        sourceCriterion:
          requestHost: true
```
