# Ping

Checking the Health of Your Traefik Instances
{: .subtitle }

## Configuration Examples

To enable the API handler:

```yaml tab="File (YAML)"
ping: {}
```

```toml tab="File (TOML)"
[ping]
```

```bash tab="CLI"
--ping=true
```

## Configuration Options

The `/ping` health-check URL is enabled with the command-line `--ping` or config file option `[ping]`.

The `entryPoint` where the `/ping` is active can be customized with the `entryPoint` option,
whose default value is `traefik` (port `8080`).

| Path    | Method        | Description                                                                                         |
|---------|---------------|-----------------------------------------------------------------------------------------------------|
| `/ping` | `GET`, `HEAD` | A simple endpoint to check for Traefik process liveness. Return a code `200` with the content: `OK` |

!!! note
    The `cli` comes with a [`healthcheck`](./cli.md#healthcheck) command which can be used for calling this endpoint.

### `entryPoint`

_Optional, Default="traefik"_

Enabling /ping on a dedicated EntryPoint.

```yaml tab="File (YAML)"
entryPoints:
  ping:
    address: ":8082"

ping:
  entryPoint: "ping"
```

```toml tab="File (TOML)"
[entryPoints]
  [entryPoints.ping]
    address = ":8082"

[ping]
  entryPoint = "ping"
```

```bash tab="CLI"
--entryPoints.ping.address=:8082
--ping.entryPoint=ping
```

### `manualRouting`

_Optional, Default=false_

If `manualRouting` is `true`, it disables the default internal router in order to allow one to create a custom router for the `ping@internal` service.

```yaml tab="File (YAML)"
ping:
  manualRouting: true
```

```toml tab="File (TOML)"
[ping]
  manualRouting = true
```

```bash tab="CLI"
--ping.manualrouting=true
```

### `terminatingStatusCode`

_Optional, Default=503_

During the period in which Traefik is gracefully shutting down, the ping handler
returns a 503 status code by default. If Traefik is behind e.g. a load-balancer
doing health checks (such as the Kubernetes LivenessProbe), another code might
be expected as the signal for graceful termination. In which case, the
terminatingStatusCode can be used to set the code returned by the ping
handler during termination.

```yaml tab="File (YAML)"
ping:
  terminatingStatusCode: 204
```

```toml tab="File (TOML)"
[ping]
  terminatingStatusCode = 204
```

```bash tab="CLI"
--ping.terminatingStatusCode=204
```
