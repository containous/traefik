# Traefik & Redis

A Story of KV store & Containers
{: .subtitle }

Store your configuration in Redis and let Traefik do the rest!

## Routing Configuration

See the dedicated section in [routing](../routing/providers/kv.md).

## Provider Configuration

### `endpoints`

_Required, Default="127.0.0.1:6379"_

Defines how to access to Redis.

```yaml tab="File (YAML)"
providers:
  redis:
    endpoints:
      - "127.0.0.1:6379"
```

```toml tab="File (TOML)"
[providers.redis]
  endpoints = ["127.0.0.1:6379"]
```

```bash tab="CLI"
--providers.redis.endpoints=127.0.0.1:6379
```

### `rootKey`

_Required, Default="traefik"_

Defines the root key of the configuration.

```yaml tab="File (YAML)"
providers:
  redis:
    rootKey: "traefik"
```

```toml tab="File (TOML)"
[providers.redis]
  rootKey = "traefik"
```

```bash tab="CLI"
--providers.redis.rootkey=traefik
```

### `username`

_Optional, Default=""_

Defines a username to connect with Redis.

```yaml tab="File (YAML)"
providers:
  redis:
    # ...
    usename: "foo"
```

```toml tab="File (TOML)"
[providers.redis]
  # ...
  username = "foo"
```

```bash tab="CLI"
--providers.redis.username=foo
```

### `password`

_Optional, Default=""_

Defines a password to connect with Redis.

```yaml tab="File (YAML)"
providers:
  redis:
    # ...
    password: "bar"
```

```toml tab="File (TOML)"
[providers.redis]
  # ...
  password = "bar"
```

```bash tab="CLI"
--providers.redis.password=foo
```

### `tls`

_Optional_

#### `tls.ca`

Certificate Authority used for the secure connection to Redis.

```yaml tab="File (YAML)"
providers:
  redis:
    tls:
      ca: path/to/ca.crt
```

```toml tab="File (TOML)"
[providers.redis.tls]
  ca = "path/to/ca.crt"
```

```bash tab="CLI"
--providers.redis.tls.ca=path/to/ca.crt
```

#### `tls.caOptional`

The value of `tls.caOptional` defines which policy should be used for the secure connection with TLS Client Authentication to Redis.

!!! warning ""

    If `tls.ca` is undefined, this option will be ignored, and no client certificate will be requested during the handshake. Any provided certificate will thus never be verified.

When this option is set to `true`, a client certificate is requested during the handshake but is not required. If a certificate is sent, it is required to be valid.

When this option is set to `false`, a client certificate is requested during the handshake, and at least one valid certificate should be sent by the client.

```yaml tab="File (YAML)"
providers:
  redis:
    tls:
      caOptional: true
```

```toml tab="File (TOML)"
[providers.redis.tls]
  caOptional = true
```

```bash tab="CLI"
--providers.redis.tls.caOptional=true
```

#### `tls.cert`

Public certificate used for the secure connection to Redis.

```yaml tab="File (YAML)"
providers:
  redis:
    tls:
      cert: path/to/foo.cert
      key: path/to/foo.key
```

```toml tab="File (TOML)"
[providers.redis.tls]
  cert = "path/to/foo.cert"
  key = "path/to/foo.key"
```

```bash tab="CLI"
--providers.redis.tls.cert=path/to/foo.cert
--providers.redis.tls.key=path/to/foo.key
```

#### `tls.key`

Private certificate used for the secure connection to Redis.

```yaml tab="File (YAML)"
providers:
  redis:
    tls:
      cert: path/to/foo.cert
      key: path/to/foo.key
```

```toml tab="File (TOML)"
[providers.redis.tls]
  cert = "path/to/foo.cert"
  key = "path/to/foo.key"
```

```bash tab="CLI"
--providers.redis.tls.cert=path/to/foo.cert
--providers.redis.tls.key=path/to/foo.key
```

#### `tls.insecureSkipVerify`

If `insecureSkipVerify` is `true`, the TLS connection to Redis accepts any certificate presented by the server regardless of the hostnames it covers.

```yaml tab="File (YAML)"
providers:
  redis:
    tls:
      insecureSkipVerify: true
```

```toml tab="File (TOML)"
[providers.redis.tls]
  insecureSkipVerify = true
```

```bash tab="CLI"
--providers.redis.tls.insecureSkipVerify=true
```
