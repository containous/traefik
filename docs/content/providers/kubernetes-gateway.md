# Traefik & Kubernetes with Gateway API

The Kubernetes Gateway API, The Experimental Way.
{: .subtitle }

Gateway API is the evolution of Kubernetes APIs that relate to `Services`, such as `Ingress`.
The Gateway API project is part of Kubernetes, working under SIG-NETWORK.

The Kubernetes Gateway provider is a Traefik implementation of the [Gateway API](https://gateway-api.sigs.k8s.io/)
specifications from the Kubernetes Special Interest Groups (SIGs).

This provider is proposed as an experimental feature and partially supports the Gateway API [v0.2.0](https://github.com/kubernetes-sigs/gateway-api/releases/tag/v0.2.0) specification.

!!! warning "Enabling The Experimental Kubernetes Gateway Provider"

    Since this provider is still experimental, it needs to be activated in the experimental section of the static configuration.

    ```toml tab="File (TOML)"
    [experimental]
      kubernetesGateway = true

    [providers.kubernetesGateway]
    #...
    ```

    ```yaml tab="File (YAML)"
    experimental:
      kubernetesGateway: true

    providers:
      kubernetesGateway: {}
      #...
    ```

    ```bash tab="CLI"
    --experimental.kubernetesgateway=true --providers.kubernetesgateway=true #...
    ```

## Configuration Requirements

!!! tip "All Steps for a Successful Deployment"

    * Add/update the Kubernetes Gateway API [definitions](../reference/dynamic-configuration/kubernetes-gateway.md#definitions).
    * Add/update the [RBAC](https://kubernetes.io/docs/reference/access-authn-authz/rbac/) for the Traefik custom resources.
    * Add all needed Kubernetes Gateway API [resources](../reference/dynamic-configuration/kubernetes-gateway.md#resources).

## Examples

??? example "Kubernetes Gateway Provider Basic Example"

    ```yaml tab="Gateway API"
    --8<-- "content/reference/dynamic-configuration/kubernetes-gateway-simple-https.yml"
    ```

    ```yaml tab="Whoami Service"
    --8<-- "content/reference/dynamic-configuration/kubernetes-whoami-svc.yml"
    ```

    ```yaml tab="Traefik Service"
    --8<-- "content/reference/dynamic-configuration/kubernetes-gateway-traefik-lb-svc.yml"
    ```

    ```yaml tab="Gateway API CRDs"
    # All resources definition must be declared
    --8<-- "content/reference/dynamic-configuration/networking.x-k8s.io_gatewayclasses.yaml"
    --8<-- "content/reference/dynamic-configuration/networking.x-k8s.io_gateways.yaml"
    --8<-- "content/reference/dynamic-configuration/networking.x-k8s.io_httproutes.yaml"
    ```

    ```yaml tab="RBAC"
    --8<-- "content/reference/dynamic-configuration/kubernetes-gateway-rbac.yml"
    ```

The Kubernetes Gateway API project provides several guides on how to use the APIs.
These guides can help you to go further than the example above.
The [getting started guide](https://gateway-api.sigs.k8s.io/guides/getting-started/) details how to install the CRDs from their repository.

!!! note ""

    Keep in mind that the Traefik Gateway provider only supports the `v0.2.0`.

For now, the Traefik Gateway Provider can be used while following the below guides:

* [Simple Gateway](https://gateway-api.sigs.k8s.io/guides/simple-gateway/)
* [HTTP routing](https://gateway-api.sigs.k8s.io/guides/http-routing/)
* [TLS](https://gateway-api.sigs.k8s.io/guides/tls/)

## Resource Configuration

When using Kubernetes Gateway API as a provider, Traefik uses Kubernetes
[Custom Resource Definitions](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)
to retrieve its routing configuration.

All concepts can be found in the official API concepts [documentation](https://gateway-api.sigs.k8s.io/concepts/api-overview/).
Traefik implements the following resources:

* `GatewayClass` defines a set of Gateways that share a common configuration and behaviour.
* `Gateway` describes how traffic can be translated to Services within the cluster.
* `HTTPRoute` defines HTTP rules for mapping requests from a Gateway to Kubernetes Services.
* `TCPRoute` defines TCP rules for mapping requests from a Gateway to Kubernetes Services.
* `TLSRoute` defines TLS rules for mapping requests from a Gateway to Kubernetes Services.

## Provider Configuration

### `endpoint`

_Optional, Default=""_

The Kubernetes server endpoint URL.

When deployed into Kubernetes, Traefik reads the environment variables `KUBERNETES_SERVICE_HOST` and `KUBERNETES_SERVICE_PORT` or `KUBECONFIG` to construct the endpoint.

The access token is looked up in `/var/run/secrets/kubernetes.io/serviceaccount/token` and the SSL CA certificate in `/var/run/secrets/kubernetes.io/serviceaccount/ca.crt`.
Both are mounted automatically when deployed inside Kubernetes.

The endpoint may be specified to override the environment variable values inside a cluster.

When the environment variables are not found, Traefik tries to connect to the Kubernetes API server with an external-cluster client.
In this case, the endpoint is required.
Specifically, it may be set to the URL used by `kubectl proxy` to connect to a Kubernetes cluster using the granted authentication and authorization of the associated kubeconfig.

```toml tab="File (TOML)"
[providers.kubernetesGateway]
  endpoint = "http://localhost:8080"
  # ...
```

```yaml tab="File (YAML)"
providers:
  kubernetesGateway:
    endpoint: "http://localhost:8080"
    # ...
```

```bash tab="CLI"
--providers.kubernetesgateway.endpoint=http://localhost:8080
```

### `token`

_Optional, Default=""_

Bearer token used for the Kubernetes client configuration.

```toml tab="File (TOML)"
[providers.kubernetesGateway]
  token = "mytoken"
  # ...
```

```yaml tab="File (YAML)"
providers:
  kubernetesGateway:
    token: "mytoken"
    # ...
```

```bash tab="CLI"
--providers.kubernetesgateway.token=mytoken
```

### `certAuthFilePath`

_Optional, Default=""_

Path to the certificate authority file.
Used for the Kubernetes client configuration.

```toml tab="File (TOML)"
[providers.kubernetesGateway]
  certAuthFilePath = "/my/ca.crt"
  # ...
```

```yaml tab="File (YAML)"
providers:
  kubernetesGateway:
    certAuthFilePath: "/my/ca.crt"
    # ...
```

```bash tab="CLI"
--providers.kubernetesgateway.certauthfilepath=/my/ca.crt
```

### `namespaces`

_Optional, Default: []_

Array of namespaces to watch.
If left empty, watches all namespaces if the value of `namespaces`.

```toml tab="File (TOML)"
[providers.kubernetesGateway]
  namespaces = ["default", "production"]
  # ...
```

```yaml tab="File (YAML)"
providers:
  kubernetesGateway:
    namespaces:
    - "default"
    - "production"
    # ...
```

```bash tab="CLI"
--providers.kubernetesgateway.namespaces=default,production
```

### `labelselector`

_Optional, Default: ""_

A label selector can be defined to filter on specific GatewayClass objects only.
If left empty, Traefik processes all GatewayClass objects in the configured namespaces.

See [label-selectors](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors) for details.

```toml tab="File (TOML)"
[providers.kubernetesGateway]
  labelselector = "app=traefik"
  # ...
```

```yaml tab="File (YAML)"
providers:
  kubernetesGateway:
    labelselector: "app=traefik"
    # ...
```

```bash tab="CLI"
--providers.kubernetesgateway.labelselector="app=traefik"
```

### `throttleDuration`

_Optional, Default: 0_

The `throttleDuration` option defines how often the provider is allowed to handle events from Kubernetes. This prevents
a Kubernetes cluster that updates many times per second from continuously changing your Traefik configuration.

If left empty, the provider does not apply any throttling and does not drop any Kubernetes events.

The value of `throttleDuration` should be provided in seconds or as a valid duration format,
see [time.ParseDuration](https://golang.org/pkg/time/#ParseDuration).

```toml tab="File (TOML)"
[providers.kubernetesGateway]
  throttleDuration = "10s"
  # ...
```

```yaml tab="File (YAML)"
providers:
  kubernetesGateway:
    throttleDuration: "10s"
    # ...
```

```bash tab="CLI"
--providers.kubernetesgateway.throttleDuration=10s
```
