# StripPrefix

Removing Prefixes From the Path Before Forwarding the Request
{: .subtitle }

<!--
TODO: add schema
-->

Remove the specified prefixes from the URL path.

## Configuration Examples

```yaml tab="Docker"
# Strip prefix /foobar and /fiibar
labels:
  - "traefik.http.middlewares.test-stripprefix.stripprefix.prefixes=/foobar,/fiibar"
```

```yaml tab="Kubernetes"
# Strip prefix /foobar and /fiibar
apiVersion: traefik.containo.us/v1alpha1
kind: Middleware
metadata:
  name: test-stripprefix
spec:
  stripPrefix:
    prefixes:
      - /foobar
      - /fiibar
```

```yaml tab="Consul Catalog"
# Strip prefix /foobar and /fiibar
- "traefik.http.middlewares.test-stripprefix.stripprefix.prefixes=/foobar,/fiibar"
```

```json tab="Marathon"
"labels": {
  "traefik.http.middlewares.test-stripprefix.stripprefix.prefixes": "/foobar,/fiibar"
}
```

```yaml tab="Rancher"
# Strip prefix /foobar and /fiibar
labels:
  - "traefik.http.middlewares.test-stripprefix.stripprefix.prefixes=/foobar,/fiibar"
```

```toml tab="File (TOML)"
# Strip prefix /foobar and /fiibar
[http.middlewares]
  [http.middlewares.test-stripprefix.stripPrefix]
    prefixes = ["/foobar", "/fiibar"]
```

```yaml tab="File (YAML)"
# Strip prefix /foobar and /fiibar
http:
  middlewares:
    test-stripprefix:
      stripPrefix:
        prefixes:
          - "/foobar"
          - "/fiibar"
```

## Configuration Options

### General

The StripPrefix middleware strips the matching path prefix and stores it in a `X-Forwarded-Prefix` header.

!!! tip

    Use a `StripPrefix` middleware if your backend listens on the root path (`/`) but should be exposed on a specific prefix.

### `prefixes`

The `prefixes` option defines the prefixes to strip from the request URL.

For instance, `/products` also matches `/products/shoes` and `/products/shirts`.

If your backend is serving assets (e.g., images or JavaScript files), it can use the `X-Forwarded-Prefix` header to properly construct relative URLs.
Using the previous example, the backend should return `/products/shoes/image.png` (and not `/images.png`, which Traefik would likely not be able to associate with the same backend).

### `forceSlash`

_Optional, Default=true_

The `forceSlash` option ensures the resulting stripped path is not the empty string, by replacing it with `/` when necessary.

This option was added to keep the initial (non-intuitive) behavior of this middleware, in order to avoid introducing a breaking change.

It is recommended to explicitly set `forceSlash` to `false`.

??? info "Behavior examples"

    - `forceSlash=true`

    | Path       | Prefix to strip | Result |
    |------------|-----------------|--------|
    | `/`        | `/`             | `/`    |
    | `/foo`     | `/foo`          | `/`    |
    | `/foo/`    | `/foo`          | `/`    |
    | `/foo/`    | `/foo/`         | `/`    |
    | `/bar`     | `/foo`          | `/bar` |
    | `/foo/bar` | `/foo`          | `/bar` |

    - `forceSlash=false`

    | Path       | Prefix to strip | Result |
    |------------|-----------------|--------|
    | `/`        | `/`             | empty  |
    | `/foo`     | `/foo`          | empty  |
    | `/foo/`    | `/foo`          | `/`    |
    | `/foo/`    | `/foo/`         | empty  |
    | `/bar`     | `/foo`          | `/bar` |
    | `/foo/bar` | `/foo`          | `/bar` |

```yaml tab="Docker"
labels:
  - "traefik.http.middlewares.example.stripprefix.prefixes=/foobar"
  - "traefik.http.middlewares.example.stripprefix.forceSlash=false"
```

```yaml tab="Kubernetes"
apiVersion: traefik.containo.us/v1alpha1
kind: Middleware
metadata:
  name: example
spec:
  stripPrefix:
    prefixes:
      - "/foobar"
    forceSlash: false
```

```json tab="Marathon"
"labels": {
  "traefik.http.middlewares.example.stripprefix.prefixes": "/foobar",
  "traefik.http.middlewares.example.stripprefix.forceSlash": "false"
}
```

```yaml tab="Rancher"
labels:
  - "traefik.http.middlewares.example.stripprefix.prefixes=/foobar"
  - "traefik.http.middlewares.example.stripprefix.forceSlash=false"
```

```toml tab="File (TOML)"
[http.middlewares]
  [http.middlewares.example.stripPrefix]
    prefixes = ["/foobar"]
    forceSlash = false
```

```yaml tab="File (YAML)"
http:
  middlewares:
    example:
      stripPrefix:
        prefixes:
          - "/foobar"
        forceSlash: false
```
