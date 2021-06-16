# RedirectScheme

Redirecting the Client to a Different Scheme/Port
{: .subtitle }

<!--
TODO: add schema
-->

RedirectScheme redirects requests from a scheme/port to another.

## Configuration Examples

```yaml tab="Docker"
# Redirect to https
labels:
  - "traefik.http.middlewares.test-redirectscheme.redirectscheme.scheme=https"
  - "traefik.http.middlewares.test-redirectscheme.redirectscheme.permanent=true"
```

```yaml tab="Kubernetes"
# Redirect to https
apiVersion: traefik.containo.us/v1alpha1
kind: Middleware
metadata:
  name: test-redirectscheme
spec:
  redirectScheme:
    scheme: https
    permanent: true
```

```yaml tab="Consul Catalog"
# Redirect to https
labels:
  - "traefik.http.middlewares.test-redirectscheme.redirectscheme.scheme=https"
  - "traefik.http.middlewares.test-redirectscheme.redirectscheme.permanent=true"
```

```json tab="Marathon"
"labels": {
  "traefik.http.middlewares.test-redirectscheme.redirectscheme.scheme": "https"
  "traefik.http.middlewares.test-redirectscheme.redirectscheme.permanent": "true"
}
```

```yaml tab="Rancher"
# Redirect to https
labels:
  - "traefik.http.middlewares.test-redirectscheme.redirectscheme.scheme=https"
  - "traefik.http.middlewares.test-redirectscheme.redirectscheme.permanent=true"
```

```toml tab="File (TOML)"
# Redirect to https
[http.middlewares]
  [http.middlewares.test-redirectscheme.redirectScheme]
    scheme = "https"
    permanent = true
```

```yaml tab="File (YAML)"
# Redirect to https
http:
  middlewares:
    test-redirectscheme:
      redirectScheme:
        scheme: https
        permanent: true
```

## Configuration Options

### `permanent`

Set the `permanent` option to `true` to apply a permanent redirection.

```yaml tab="Docker"
# Redirect to https
labels:
  # ...
  - "traefik.http.middlewares.test-redirectscheme.redirectscheme.permanent=true"
```

```yaml tab="Kubernetes"
# Redirect to https
apiVersion: traefik.containo.us/v1alpha1
kind: Middleware
metadata:
  name: test-redirectscheme
spec:
  redirectScheme:
    # ...
    permanent: true
```

```yaml tab="Consul Catalog"
# Redirect to https
labels:
  # ...
  - "traefik.http.middlewares.test-redirectscheme.redirectscheme.permanent=true"
```

```json tab="Marathon"
"labels": {

  "traefik.http.middlewares.test-redirectscheme.redirectscheme.permanent": "true"
}
```

```yaml tab="Rancher"
# Redirect to https
labels:
  # ...
  - "traefik.http.middlewares.test-redirectscheme.redirectscheme.permanent=true"
```

```toml tab="File (TOML)"
# Redirect to https
[http.middlewares]
  [http.middlewares.test-redirectscheme.redirectScheme]
    # ...
    permanent = true
```

```yaml tab="File (YAML)"
# Redirect to https
http:
  middlewares:
    test-redirectscheme:
      redirectScheme:
        # ...
        permanent: true
```

### `scheme`

The `scheme` option defines the scheme of the new URL.

```yaml tab="Docker"
# Redirect to https
labels:
  - "traefik.http.middlewares.test-redirectscheme.redirectscheme.scheme=https"
```

```yaml tab="Kubernetes"
# Redirect to https
apiVersion: traefik.containo.us/v1alpha1
kind: Middleware
metadata:
  name: test-redirectscheme
spec:
  redirectScheme:
    scheme: https
```

```yaml tab="Consul Catalog"
# Redirect to https
labels:
  - "traefik.http.middlewares.test-redirectscheme.redirectscheme.scheme=https"
```

```json tab="Marathon"
"labels": {
  "traefik.http.middlewares.test-redirectscheme.redirectscheme.scheme": "https"
}
```

```yaml tab="Rancher"
# Redirect to https
labels:
  - "traefik.http.middlewares.test-redirectscheme.redirectscheme.scheme=https"
```

```toml tab="File (TOML)"
# Redirect to https
[http.middlewares]
  [http.middlewares.test-redirectscheme.redirectScheme]
    scheme = "https"
```

```yaml tab="File (YAML)"
# Redirect to https
http:
  middlewares:
    test-redirectscheme:
      redirectScheme:
        scheme: https
```

### `port`

The `port` option defines the port of the new URL.

```yaml tab="Docker"
# Redirect to https
labels:
  # ...
  - "traefik.http.middlewares.test-redirectscheme.redirectscheme.port=443"
```

```yaml tab="Kubernetes"
# Redirect to https
apiVersion: traefik.containo.us/v1alpha1
kind: Middleware
metadata:
  name: test-redirectscheme
spec:
  redirectScheme:
    # ...
    port: "443"
```

```yaml tab="Consul Catalog"
# Redirect to https
labels:
  # ...
  - "traefik.http.middlewares.test-redirectscheme.redirectscheme.port=443"
```

```json tab="Marathon"
"labels": {

  "traefik.http.middlewares.test-redirectscheme.redirectscheme.port": "443"
}
```

```yaml tab="Rancher"
# Redirect to https
labels:
  # ...
  - "traefik.http.middlewares.test-redirectscheme.redirectscheme.port=443"
```

```toml tab="File (TOML)"
# Redirect to https
[http.middlewares]
  [http.middlewares.test-redirectscheme.redirectScheme]
    # ...
    port = 443
```

```yaml tab="File (YAML)"
# Redirect to https
http:
  middlewares:
    test-redirectscheme:
      redirectScheme:
        # ...
        port: "443"
```

!!! info "Port in this configuration is a string, not a numeric value."
