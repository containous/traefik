# Traefik & Etcd

A Story of KV store & Containers
{: .subtitle }

Store your configuration in etcd and let Traefik do the rest!

## Routing Configuration

See the dedicated section in [routing](../routing/providers/kv.md).

## Provider Configuration

### `endpoints`

_Required, Default="127.0.0.1:2379"_

Defines how to access etcd.

```yaml tab="File (YAML)"
providers:
  etcd:
    endpoints:
      - "127.0.0.1:2379"
```

```toml tab="File (TOML)"
[providers.etcd]
  endpoints = ["127.0.0.1:2379"]
```

```bash tab="CLI"
--providers.etcd.endpoints=127.0.0.1:2379
```

### `rootKey`

_Required, Default="traefik"_

Defines the root key of the configuration.

```yaml tab="File (YAML)"
providers:
  etcd:
    rootKey: "traefik"
```

```toml tab="File (TOML)"
[providers.etcd]
  rootKey = "traefik"
```

```bash tab="CLI"
--providers.etcd.rootkey=traefik
```

### `username`

_Optional, Default=""_

Defines a username with which to connect to etcd.

```yaml tab="File (YAML)"
providers:
  etcd:
    # ...
    usename: "foo"
```

```toml tab="File (TOML)"
[providers.etcd]
  # ...
  username = "foo"
```

```bash tab="CLI"
--providers.etcd.username=foo
```

### `password`

_Optional, Default=""_

Defines a password with which to connect to etcd.

```yaml tab="File (YAML)"
providers:
  etcd:
    # ...
    password: "bar"
```

```toml tab="File (TOML)"
[providers.etcd]
  # ...
  password = "bar"
```

```bash tab="CLI"
--providers.etcd.password=foo
```

### `tls`

_Optional_

#### `tls.ca`

Certificate Authority used for the secure connection to etcd.

```yaml tab="File (YAML)"
providers:
  etcd:
    tls:
      ca: path/to/ca.crt
```

```toml tab="File (TOML)"
[providers.etcd.tls]
  ca = "path/to/ca.crt"
```

```bash tab="CLI"
--providers.etcd.tls.ca=path/to/ca.crt
```

#### `tls.caOptional`

The value of `tls.caOptional` defines which policy should be used for the secure connection with TLS Client Authentication to etcd.

!!! warning ""

    If `tls.ca` is undefined, this option will be ignored, and no client certificate will be requested during the handshake. Any provided certificate will thus never be verified.

When this option is set to `true`, a client certificate is requested during the handshake but is not required. If a certificate is sent, it is required to be valid.

When this option is set to `false`, a client certificate is requested during the handshake, and at least one valid certificate should be sent by the client.

```yaml tab="File (YAML)"
providers:
  etcd:
    tls:
      caOptional: true
```

```toml tab="File (TOML)"
[providers.etcd.tls]
  caOptional = true
```

```bash tab="CLI"
--providers.etcd.tls.caOptional=true
```

#### `tls.cert`

Public certificate used for the secure connection to etcd.

```yaml tab="File (YAML)"
providers:
  etcd:
    tls:
      cert: path/to/foo.cert
      key: path/to/foo.key
```

```toml tab="File (TOML)"
[providers.etcd.tls]
  cert = "path/to/foo.cert"
  key = "path/to/foo.key"
```

```bash tab="CLI"
--providers.etcd.tls.cert=path/to/foo.cert
--providers.etcd.tls.key=path/to/foo.key
```

#### `tls.key`

Private certificate used for the secure connection to etcd.

```yaml tab="File (YAML)"
providers:
  etcd:
    tls:
      cert: path/to/foo.cert
      key: path/to/foo.key
```

```toml tab="File (TOML)"
[providers.etcd.tls]
  cert = "path/to/foo.cert"
  key = "path/to/foo.key"
```

```bash tab="CLI"
--providers.etcd.tls.cert=path/to/foo.cert
--providers.etcd.tls.key=path/to/foo.key
```

#### `tls.insecureSkipVerify`

If `insecureSkipVerify` is `true`, the TLS connection to etcd accepts any certificate presented by the server regardless of the hostnames it covers.

```yaml tab="File (YAML)"
providers:
  etcd:
    tls:
      insecureSkipVerify: true
```

```toml tab="File (TOML)"
[providers.etcd.tls]
  insecureSkipVerify = true
```

```bash tab="CLI"
--providers.etcd.tls.insecureSkipVerify=true
```
