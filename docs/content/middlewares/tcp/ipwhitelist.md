# IPWhiteList

Limiting Clients to Specific IPs
{: .subtitle }

IPWhitelist accepts / refuses connections based on the client IP.

## Configuration Examples

```yaml tab="Docker"
# Accepts connections from defined IP
labels:
  - "traefik.tcp.middlewares.test-ipwhitelist.ipwhitelist.sourcerange=127.0.0.1/32, 192.168.1.7"
```

```yaml tab="Kubernetes"
apiVersion: traefik.containo.us/v1alpha1
kind: MiddlewareTCP
metadata:
  name: test-ipwhitelist
spec:
  ipWhiteList:
    sourceRange:
      - 127.0.0.1/32
      - 192.168.1.7
```

```yaml tab="Consul Catalog"
# Accepts request from defined IP
- "traefik.tcp.middlewares.test-ipwhitelist.ipwhitelist.sourcerange=127.0.0.1/32, 192.168.1.7"
```

```json tab="Marathon"
"labels": {
  "traefik.tcp.middlewares.test-ipwhitelist.ipwhitelist.sourcerange": "127.0.0.1/32,192.168.1.7"
}
```

```yaml tab="Rancher"
# Accepts request from defined IP
labels:
  - "traefik.tcp.middlewares.test-ipwhitelist.ipwhitelist.sourcerange=127.0.0.1/32, 192.168.1.7"
```

```toml tab="File (TOML)"
# Accepts request from defined IP
[tcp.middlewares]
  [tcp.middlewares.test-ipwhitelist.ipWhiteList]
    sourceRange = ["127.0.0.1/32", "192.168.1.7"]
```

```yaml tab="File (YAML)"
# Accepts request from defined IP
http:
  middlewares:
    test-ipwhitelist:
      ipWhiteList:
        sourceRange:
          - "127.0.0.1/32"
          - "192.168.1.7"
```

## Configuration Options

### `sourceRange`

The `sourceRange` option sets the allowed IPs (or ranges of allowed IPs by using CIDR notation).
