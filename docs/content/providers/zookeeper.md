# Traefik & ZooKeeper

A Story of KV Store & Containers
{: .subtitle }

Store your configuration in ZooKeeper and let Traefik do the rest!

## Routing Configuration

See the dedicated section in [routing](../routing/providers/kv.md).

## Provider Configuration

### `endpoints`

_Required, Default="127.0.0.1:2181"_

Defines how to access to ZooKeeper.

```yaml tab="File (YAML)"
providers:
  zooKeeper:
    endpoints:
      - "127.0.0.1:2181"
```

```toml tab="File (TOML)"
[providers.zooKeeper]
  endpoints = ["127.0.0.1:2181"]
```

```bash tab="CLI"
--providers.zookeeper.endpoints=127.0.0.1:2181
```

### `rootKey`

_Required, Default="traefik"_

Defines the root key of the configuration.

```yaml tab="File (YAML)"
providers:
  zooKeeper:
    rootKey: "traefik"
```

```toml tab="File (TOML)"
[providers.zooKeeper]
  rootKey = "traefik"
```

```bash tab="CLI"
--providers.zookeeper.rootkey=traefik
```

### `username`

_Optional, Default=""_

Defines a username to connect with ZooKeeper.

```yaml tab="File (YAML)"
providers:
  zooKeeper:
    # ...
    usename: "foo"
```

```toml tab="File (TOML)"
[providers.zooKeeper]
  # ...
  username = "foo"
```

```bash tab="CLI"
--providers.zookeeper.username=foo
```

### `password`

_Optional, Default=""_

Defines a password to connect with ZooKeeper.

```yaml tab="File (YAML)"
providers:
  zooKeeper:
    # ...
    password: "bar"
```

```toml tab="File (TOML)"
[providers.zooKeeper]
  # ...
  password = "bar"
```

```bash tab="CLI"
--providers.zookeeper.password=foo
```

### `tls`

_Optional_

#### `tls.ca`

Certificate Authority used for the secure connection to ZooKeeper.

```yaml tab="File (YAML)"
providers:
  zooKeeper:
    tls:
      ca: path/to/ca.crt
```

```toml tab="File (TOML)"
[providers.zooKeeper.tls]
  ca = "path/to/ca.crt"
```

```bash tab="CLI"
--providers.zookeeper.tls.ca=path/to/ca.crt
```

#### `tls.caOptional`

The value of `tls.caOptional` defines which policy should be used for the secure connection with TLS Client Authentication to Zookeeper.

!!! warning ""

    If `tls.ca` is undefined, this option will be ignored, and no client certificate will be requested during the handshake. Any provided certificate will thus never be verified.

When this option is set to `true`, a client certificate is requested during the handshake but is not required. If a certificate is sent, it is required to be valid.

When this option is set to `false`, a client certificate is requested during the handshake, and at least one valid certificate should be sent by the client.

```yaml tab="File (YAML)"
providers:
  zooKeeper:
    tls:
      caOptional: true
```

```toml tab="File (TOML)"
[providers.zooKeeper.tls]
  caOptional = true
```

```bash tab="CLI"
--providers.zookeeper.tls.caOptional=true
```

#### `tls.cert`

Public certificate used for the secure connection to ZooKeeper.

```yaml tab="File (YAML)"
providers:
  zooKeeper:
    tls:
      cert: path/to/foo.cert
      key: path/to/foo.key
```

```toml tab="File (TOML)"
[providers.zooKeeper.tls]
  cert = "path/to/foo.cert"
  key = "path/to/foo.key"
```

```bash tab="CLI"
--providers.zookeeper.tls.cert=path/to/foo.cert
--providers.zookeeper.tls.key=path/to/foo.key
```

#### `tls.key`

Private certificate used for the secure connection to ZooKeeper.

```yaml tab="File (YAML)"
providers:
  zooKeeper:
    tls:
      cert: path/to/foo.cert
      key: path/to/foo.key
```

```toml tab="File (TOML)"
[providers.zooKeeper.tls]
  cert = "path/to/foo.cert"
  key = "path/to/foo.key"
```

```bash tab="CLI"
--providers.zookeeper.tls.cert=path/to/foo.cert
--providers.zookeeper.tls.key=path/to/foo.key
```

#### `tls.insecureSkipVerify`

If `insecureSkipVerify` is `true`, the TLS connection to Zookeeper accepts any certificate presented by the server regardless of the hostnames it covers.

```yaml tab="File (YAML)"
providers:
  zooKeeper:
    tls:
      insecureSkipVerify: true
```

```toml tab="File (TOML)"
[providers.zooKeeper.tls]
  insecureSkipVerify = true
```

```bash tab="CLI"
--providers.zookeeper.tls.insecureSkipVerify=true
```
