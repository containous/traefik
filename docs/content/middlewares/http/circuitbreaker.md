# CircuitBreaker

Don't Waste Time Calling Unhealthy Services
{: .subtitle }

![CircuitBreaker](../../assets/img/middleware/circuitbreaker.png)

The circuit breaker protects your system from stacking requests to unhealthy services, resulting in cascading failures.

When your system is healthy, the circuit is closed (normal operations).
When your system becomes unhealthy, the circuit opens, and the requests are no longer forwarded, but instead are handled by a fallback mechanism.

To assess if your system is healthy, the circuit breaker constantly monitors the services.

!!! note ""

    The CircuitBreaker only analyzes what happens _after_ its position within the middleware chain. What happens _before_ has no impact on its state.

!!! important

    Each router gets its own instance of a given circuit breaker.
    One circuit breaker instance can be open while the other remains closed: their state is not shared.

    This is the expected behavior, we want you to be able to define what makes a service healthy without having to declare a circuit breaker for each route.

## Configuration Examples

```yaml tab="Docker"
# Latency Check
labels:
  - "traefik.http.middlewares.latency-check.circuitbreaker.expression=LatencyAtQuantileMS(50.0) > 100"
```

```yaml tab="Kubernetes"
# Latency Check
apiVersion: traefik.containo.us/v1alpha1
kind: Middleware
metadata:
  name: latency-check
spec:
  circuitBreaker:
    expression: LatencyAtQuantileMS(50.0) > 100
```

```yaml tab="Consul Catalog"
# Latency Check
- "traefik.http.middlewares.latency-check.circuitbreaker.expression=LatencyAtQuantileMS(50.0) > 100"
```

```json tab="Marathon"
"labels": {
  "traefik.http.middlewares.latency-check.circuitbreaker.expression": "LatencyAtQuantileMS(50.0) > 100"
}
```

```yaml tab="Rancher"
# Latency Check
labels:
  - "traefik.http.middlewares.latency-check.circuitbreaker.expression=LatencyAtQuantileMS(50.0) > 100"
```

```yaml tab="File (YAML)"
# Latency Check
http:
  middlewares:
    latency-check:
      circuitBreaker:
        expression: "LatencyAtQuantileMS(50.0) > 100"
```

```toml tab="File (TOML)"
# Latency Check
[http.middlewares]
  [http.middlewares.latency-check.circuitBreaker]
    expression = "LatencyAtQuantileMS(50.0) > 100"
```

## Possible States

There are three possible states for your circuit breaker:

- Closed (your service operates normally)
- Open (the fallback mechanism takes over your service)
- Recovering (the circuit breaker tries to resume normal operations by progressively sending requests to your service)

### Closed

While the circuit is closed, the circuit breaker only collects metrics to analyze the behavior of the requests.

At specified intervals (`checkPeriod`), the circuit breaker evaluates `expression` to decide if its state must change.

### Open

While open, the fallback mechanism takes over the normal service calls for a duration of `FallbackDuration`.
After this duration, it enters the recovering state.

### Recovering

While recovering, the circuit breaker sends linearly increasing amounts of requests to your service (for `RecoveryDuration`).
If your service fails during recovery, the circuit breaker opens again.
If the service operates normally during the entire recovery duration, then the circuit breaker closes.

## Configuration Options

### Configuring the Trigger

You can specify an `expression` that, once matched, opens the circuit breaker and applies the fallback mechanism instead of calling your services.

The `expression` option can check three different metrics:

- The network error ratio (`NetworkErrorRatio`)
- The status code ratio (`ResponseCodeRatio`)
- The latency at a quantile in milliseconds (`LatencyAtQuantileMS`)

#### `NetworkErrorRatio`

If you want the circuit breaker to open at a 30% ratio of network errors, the `expression` is `NetworkErrorRatio() > 0.30`

#### `ResponseCodeRatio`

You can configure the circuit breaker to open based on the ratio of a given range of status codes.

The `ResponseCodeRatio` accepts four parameters, `from`, `to`, `dividedByFrom`, `dividedByTo`.

The operation that will be computed is sum(`to` -> `from`) / sum (`dividedByFrom` -> `dividedByTo`).

!!! note ""

    If sum (`dividedByFrom` -> `dividedByTo`) equals 0, then `ResponseCodeRatio` returns 0.

    `from`is inclusive, `to` is exclusive.

For example, the expression `ResponseCodeRatio(500, 600, 0, 600) > 0.25` will trigger the circuit breaker if 25% of the requests returned a 5XX status (amongst the request that returned a status code from 0 to 5XX).

#### `LatencyAtQuantileMS`

You can configure the circuit breaker to open when a given proportion of your requests become too slow.

For example, the expression `LatencyAtQuantileMS(50.0) > 100` opens the circuit breaker when the median latency (quantile 50) reaches 100ms.

!!! note ""

    You must provide a floating point number (with the trailing .0) for the quantile value

#### Using Multiple Metrics

You can combine multiple metrics using operators in your `expression`.

Supported operators are:

- AND (`&&`)
- OR (`||`)

For example, `ResponseCodeRatio(500, 600, 0, 600) > 0.30 || NetworkErrorRatio() > 0.10` triggers the circuit breaker when 30% of the requests return a 5XX status code, or when the ratio of network errors reaches 10%.

#### Operators

Here is the list of supported operators:

- Greater than (`>`)
- Greater or equal than (`>=`)
- Lesser than (`<`)
- Lesser or equal than (`<=`)
- Equal (`==`)
- Not Equal (`!=`)

### Fallback mechanism

The fallback mechanism returns a `HTTP 503 Service Unavailable` to the client instead of calling the target service.
This behavior cannot be configured.

### `CheckPeriod`

The interval used to evaluate `expression` and decide if the state of the circuit breaker must change.
By default, `CheckPeriod` is 100ms. This value cannot be configured.

### `FallbackDuration`

By default, `FallbackDuration` is 10 seconds. This value cannot be configured.

### `RecoveringDuration`

The duration of the recovering mode (recovering state).

By default, `RecoveringDuration` is 10 seconds. This value cannot be configured.
