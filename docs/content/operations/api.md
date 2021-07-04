# API

Traefik exposes a number of information through an API handler, such as the configuration of all routers, services, middlewares, etc.

As with all features of Traefik, this handler can be enabled with the [static configuration](../getting-started/configuration-overview.md#the-static-configuration).

## Security

Enabling the API in production is not recommended, because it will expose all configuration elements,
including sensitive data.

In production, it should be at least secured by authentication and authorizations.

A good sane default (non exhaustive) set of recommendations
would be to apply the following protection mechanisms:

* At the transport level:
  NOT publicly exposing the API's port,
  keeping it restricted to internal networks
  (as in the [principle of least privilege](https://en.wikipedia.org/wiki/Principle_of_least_privilege), applied to networks).

## Configuration

If you enable the API, a new special `service` named `api@internal` is created and can then be referenced in a router.

To enable the API handler, use the following option on the
[static configuration](../getting-started/configuration-overview.md#the-static-configuration):

```yaml tab="File (YAML)"
# Static Configuration
api: {}
```

```toml tab="File (TOML)"
# Static Configuration
[api]
```

```bash tab="CLI"
--api=true
```

And then define a routing configuration on Traefik itself with the
[dynamic configuration](../getting-started/configuration-overview.md#the-dynamic-configuration):

--8<-- "content/operations/include-api-examples.md"

??? warning "The router's [rule](../../routing/routers#rule) must catch requests for the URI path `/api`"
    Using an "Host" rule is recommended, by catching all the incoming traffic on this host domain to the API.
    However, you can also use "path prefix" rule or any combination or rules.

    ```bash tab="Host Rule"
    # Matches http://traefik.example.com, http://traefik.example.com/api
    # or http://traefik.example.com/hello
    rule = "Host(`traefik.example.com`)"
    ```

    ```bash tab="Path Prefix Rule"
    # Matches http://api.traefik.example.com/api or http://example.com/api
    # but does not match http://api.traefik.example.com/hello
    rule = "PathPrefix(`/api`)"
    ```

    ```bash tab="Combination of Rules"
    # Matches http://traefik.example.com/api or http://traefik.example.com/dashboard
    # but does not match http://traefik.example.com/hello
    rule = "Host(`traefik.example.com`) && (PathPrefix(`/api`) || PathPrefix(`/dashboard`))"
    ```

### `insecure`

Enable the API in `insecure` mode, which means that the API will be available directly on the entryPoint named `traefik`.

!!! info
    If the entryPoint named `traefik` is not configured, it will be automatically created on port 8080.

```yaml tab="File (YAML)"
api:
  insecure: true
```

```toml tab="File (TOML)"
[api]
  insecure = true
```

```bash tab="CLI"
--api.insecure=true
```

### `dashboard`

_Optional, Default=true_

Enable the dashboard. More about the dashboard features [here](./dashboard.md).

```yaml tab="File (YAML)"
api:
  dashboard: true
```

```toml tab="File (TOML)"
[api]
  dashboard = true
```

```bash tab="CLI"
--api.dashboard=true
```

!!! warning "With Dashboard enabled, the router [rule](../../routing/routers#rule) must catch requests for both `/api` and `/dashboard`"
    Please check the [Dashboard documentation](./dashboard.md#dashboard-router-rule) to learn more about this and to get examples.

### `debug`

_Optional, Default=false_

Enable additional [endpoints](./api.md#endpoints) for debugging and profiling, served under `/debug/`.

```yaml tab="File (YAML)"
api:
  debug: true
```

```toml tab="File (TOML)"
[api]
  debug = true
```

```bash tab="CLI"
--api.debug=true
```

## Endpoints

All the following endpoints must be accessed with a `GET` HTTP request.

| Path                           | Description                                                                                 |
|--------------------------------|---------------------------------------------------------------------------------------------|
| `/api/http/routers`            | Lists all the HTTP routers information.                                                     |
| `/api/http/routers/{name}`     | Returns the information of the HTTP router specified by `name`.                             |
| `/api/http/services`           | Lists all the HTTP services information.                                                    |
| `/api/http/services/{name}`    | Returns the information of the HTTP service specified by `name`.                            |
| `/api/http/middlewares`        | Lists all the HTTP middlewares information.                                                 |
| `/api/http/middlewares/{name}` | Returns the information of the HTTP middleware specified by `name`.                         |
| `/api/tcp/routers`             | Lists all the TCP routers information.                                                      |
| `/api/tcp/routers/{name}`      | Returns the information of the TCP router specified by `name`.                              |
| `/api/tcp/services`            | Lists all the TCP services information.                                                     |
| `/api/tcp/services/{name}`     | Returns the information of the TCP service specified by `name`.                             |
| `/api/entrypoints`             | Lists all the entry points information.                                                     |
| `/api/entrypoints/{name}`      | Returns the information of the entry point specified by `name`.                             |
| `/api/overview`                | Returns statistic information about http and tcp as well as enabled features and providers. |
| `/api/version`                 | Returns information about Traefik version.                                                  |
| `/debug/vars`                  | See the [expvar](https://golang.org/pkg/expvar/) Go documentation.                          |
| `/debug/pprof/`                | See the [pprof Index](https://golang.org/pkg/net/http/pprof/#Index) Go documentation.       |
| `/debug/pprof/cmdline`         | See the [pprof Cmdline](https://golang.org/pkg/net/http/pprof/#Cmdline) Go documentation.   |
| `/debug/pprof/profile`         | See the [pprof Profile](https://golang.org/pkg/net/http/pprof/#Profile) Go documentation.   |
| `/debug/pprof/symbol`          | See the [pprof Symbol](https://golang.org/pkg/net/http/pprof/#Symbol) Go documentation.     |
| `/debug/pprof/trace`           | See the [pprof Trace](https://golang.org/pkg/net/http/pprof/#Trace) Go documentation.       |
