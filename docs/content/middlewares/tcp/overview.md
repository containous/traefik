# TCP Middlewares

Controlling connections
{: .subtitle }

![Overview](../../assets/img/middleware/overview.png)

## Configuration Example

```yaml tab="Docker"
# As a Docker Label
whoami:
  #  A container that exposes an API to show its IP address
  image: traefik/whoami
  labels:
    # Create a middleware named `foo-ip-whitelist`
    - "traefik.tcp.middlewares.foo-ip-whitelist.ipwhitelist.sourcerange=127.0.0.1/32, 192.168.1.7"
    # Apply the middleware named `foo-ip-whitelist` to the router named `router1`
    - "traefik.tcp.routers.router1.middlewares=foo-ip-whitelist@docker"
```

```yaml tab="Kubernetes IngressRoute"
# As a Kubernetes Traefik IngressRoute
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: middlewaretcps.traefik.containo.us
spec:
  group: traefik.containo.us
  version: v1alpha1
  names:
    kind: MiddlewareTCP
    plural: middlewaretcps
    singular: middlewaretcp
  scope: Namespaced

---
apiVersion: traefik.containo.us/v1alpha1
kind: Middleware
metadata:
  name: foo-ip-whitelist
spec:
  ipWhiteList:
    sourcerange:
      - 127.0.0.1/32
      - 192.168.1.7

---
apiVersion: traefik.containo.us/v1alpha1
kind: IngressRoute
metadata:
  name: ingressroute
spec:
# more fields...
  routes:
    # more fields...
    middlewares:
      - name: foo-ip-whitelist
```

```yaml tab="Consul Catalog"
# Create a middleware named `foo-ip-whitelist`
- "traefik.tcp.middlewares.foo-ip-whitelist.ipwhitelist.sourcerange=127.0.0.1/32, 192.168.1.7"
# Apply the middleware named `foo-ip-whitelist` to the router named `router1`
- "traefik.tcp.routers.router1.middlewares=foo-ip-whitelist@consulcatalog"
```

```json tab="Marathon"
"labels": {
  "traefik.tcp.middlewares.foo-ip-whitelist.ipwhitelist.sourcerange=127.0.0.1/32, 192.168.1.7",
  "traefik.tcp.routers.router1.middlewares=foo-ip-whitelist@marathon"
}
```

```yaml tab="Rancher"
# As a Rancher Label
labels:
  # Create a middleware named `foo-ip-whitelist`
  - "traefik.tcp.middlewares.foo-ip-whitelist.ipwhitelist.sourcerange=127.0.0.1/32, 192.168.1.7"
  # Apply the middleware named `foo-ip-whitelist` to the router named `router1`
  - "traefik.tcp.routers.router1.middlewares=foo-ip-whitelist@rancher"
```

```toml tab="File (TOML)"
# As TOML Configuration File
[tcp.routers]
  [tcp.routers.router1]
    service = "myService"
    middlewares = ["foo-ip-whitelist"]
    rule = "Host(`example.com`)"

[tcp.middlewares]
  [tcp.middlewares.foo-ip-whitelist.ipWhiteList]
    sourceRange = ["127.0.0.1/32", "192.168.1.7"]

[tcp.services]
  [tcp.services.service1]
    [tcp.services.service1.loadBalancer]
    [[tcp.services.service1.loadBalancer.servers]]
      address = "10.0.0.10:4000"
    [[tcp.services.service1.loadBalancer.servers]]
      address = "10.0.0.11:4000"
```

```yaml tab="File (YAML)"
# As YAML Configuration File
tcp:
  routers:
    router1:
      service: myService
      middlewares:
        - "foo-ip-whitelist"
      rule: "Host(`example.com`)"

  middlewares:
    foo-ip-whitelist:
      ipWhiteList:
        sourceRange:
          - "127.0.0.1/32"
          - "192.168.1.7"

  services:
    service1:
      loadBalancer:
        servers:
        - address: "10.0.0.10:4000"
        - address: "10.0.0.11:4000"
```

## Available TCP Middlewares

| Middleware                                | Purpose                                           | Area                        |
|-------------------------------------------|---------------------------------------------------|-----------------------------|
| [IPWhiteList](ipwhitelist.md)             | Limit the allowed client IPs                      | Security, Request lifecycle |
