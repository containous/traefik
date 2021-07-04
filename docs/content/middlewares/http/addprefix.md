# Add Prefix

Prefixing the Path
{: .subtitle }

![AddPrefix](../../assets/img/middleware/addprefix.png)

The AddPrefix middleware updates the path of a request before forwarding it.

## Configuration Examples

```yaml tab="Docker"
# Prefixing with /foo
labels:
  - "traefik.http.middlewares.add-foo.addprefix.prefix=/foo"
```

```yaml tab="Kubernetes"
# Prefixing with /foo
apiVersion: traefik.containo.us/v1alpha1
kind: Middleware
metadata:
  name: add-foo
spec:
  addPrefix:
    prefix: /foo
```

```yaml tab="Consul Catalog"
# Prefixing with /foo
- "traefik.http.middlewares.add-foo.addprefix.prefix=/foo"
```

```json tab="Marathon"
"labels": {
  "traefik.http.middlewares.add-foo.addprefix.prefix": "/foo"
}
```

```yaml tab="Rancher"
# Prefixing with /foo
labels:
  - "traefik.http.middlewares.add-foo.addprefix.prefix=/foo"
```

```yaml tab="File (YAML)"
# Prefixing with /foo
http:
  middlewares:
    add-foo:
      addPrefix:
        prefix: "/foo"
```

```toml tab="File (TOML)"
# Prefixing with /foo
[http.middlewares]
  [http.middlewares.add-foo.addPrefix]
    prefix = "/foo"
```

## Configuration Options

### `prefix`

`prefix` is the string to add before the current path in the requested URL.
It should include a leading slash (`/`).
