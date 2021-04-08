# Traefik & Kubernetes

The Kubernetes Gateway API, The Experimental Way.
{: .subtitle }

## Configuration Examples

??? example "Configuring Kubernetes Gateway provider and Deploying/Exposing Services"

    ```yaml tab="Gateway API"
    --8<-- "content/reference/dynamic-configuration/kubernetes-gateway-simple-https.yml"
    ```

    ```yaml tab="Whoami Service"
    --8<-- "content/reference/dynamic-configuration/kubernetes-whoami-svc.yml"
    ```
    
    ```yaml tab="Traefik Service"
    --8<-- "content/reference/dynamic-configuration/kubernetes-gateway-traefik-lb-svc.yml"
    ```

    ```yaml tab="RBAC"
    --8<-- "content/reference/dynamic-configuration/kubernetes-gateway-rbac.yml"
    ```

## Routing Configuration

### Custom Resource Definition (CRD)

* You can find an exhaustive list, of the custom resources and their attributes in
[the reference page](../../reference/dynamic-configuration/kubernetes-gateway.md) or in the Kubernetes Sigs `Gateway API` [repository](https://github.com/kubernetes-sigs/gateway-api).
* Validate that [the prerequisites](../../providers/kubernetes-gateway.md#configuration-requirements) are fulfilled before using the Traefik Kubernetes Gateway Provider.
    
You can find an excerpt of the supported Kubernetes Gateway API resources in the table below:

| Kind                               | Purpose                                                                   | Concept Behind                                                              |
|------------------------------------|---------------------------------------------------------------------------|-----------------------------------------------------------------------------|
| [GatewayClass](#kind-gatewayclass) | Defines a set of Gateways that share a common configuration and behaviour | [GatewayClass](https://gateway-api.sigs.k8s.io/api-overview/#gatewayclass)  |
| [Gateway](#kind-gateway)           | Describes how traffic can be translated to Services within the cluster    | [Gateway](https://gateway-api.sigs.k8s.io/api-overview/#gateway)            |
| [HTTPRoute](#kind-httproute)       | HTTP rules for mapping requests from a Gateway to Kubernetes Services     | [Route](https://gateway-api.sigs.k8s.io/api-overview/#httptcpfooroute)      |

### Kind: `GatewayClass`

`GatewayClass` is cluster-scoped resource defined by the infrastructure provider. This resource represents a class of Gateways that can be instantiated.
More details on the GatewayClass [official documentation](https://gateway-api.sigs.k8s.io/gatewayclass/).

The `GatewayClass` should be declared by the infrastructure provider, otherwise please register the `GatewayClass`
[definition](../../reference/dynamic-configuration/kubernetes-gateway.md#definitions) in the Kubernetes cluster before 
creating `GatewayClass` objects.

!!! info "Declaring GatewayClass"

    ```yaml
    kind: GatewayClass
    apiVersion: networking.x-k8s.io/v1alpha1
    metadata:
      name: my-gateway-class
    spec:
      # Controller is a domain/path string that indicates
      # the controller that is managing Gateways of this class.
      controller: traefik.io/gateway-controller
    ```

### Kind: `Gateway`

A `Gateway` is 1:1 with the life cycle of the configuration of infrastructure. When a user creates a Gateway, 
some load balancing infrastructure is provisioned or configured by the GatewayClass controller. 
More details on the Gateway [official documentation](https://gateway-api.sigs.k8s.io/gateway/).

Register the `Gateway` [definition](../../reference/dynamic-configuration/kubernetes-gateway.md#definitions) in the
Kubernetes cluster before creating `Gateway` objects.

!!! info "Declaring Gateway"

    ```
    kind: Gateway
    apiVersion: networking.x-k8s.io/v1alpha1
    metadata:
      name: my-gateway
      namespace: default
    spec:
      gatewayClassName: my-gateway-class        # [1]
      listeners:                                # [2]
        - protocol: HTTPS                       # [3] 
          port: 443                             # [4]
          tls:                                  # [5]
            certificateRef:                     # [6]
              group: "core"
              kind: "Secret"
              name: "mysecret"
          routes:                               # [7]
            kind: HTTPRoute                     # [8]
            selector:                           # [9]
              matchLabels:                      # [10]
                app: foo
    ```

| Ref  | Attribute          | Description                                                                                                                 |
|------|--------------------|-----------------------------------------------------------------------------------------------------------------------------|
| [1]  | `gatewayClassName` | GatewayClassName used for this Gateway. This is the name of a GatewayClass resource.                                        |
| [2]  | `listeners`        | Logical endpoints that are bound on this Gateway's addresses. At least one Listener MUST be specified.                      |
| [3]  | `protocol`         | The network protocol this listener expects to receive (only HTTP and HTTPS are implemented).                                |
| [4]  | `port`             | The network port.                                                                                                           |
| [5]  | `tls`              | TLS configuration for the Listener. This field is required if the Protocol field is "HTTPS" or "TLS" and ignored otherwise. |
| [6]  | `certificateRef`   | The reference to Kubernetes object that contains a TLS certificate and private key.                                         |
| [7]  | `routes`           | A schema for associating routes with the Listener using selectors.                                                          |
| [8]  | `kind`             | The kind of the referent.                                                                                                   |
| [9]  | `selector`         | Routes in namespaces selected by the selector may be used by this Gateway routes to associate with the Gateway.   |
| [10] | `matchLabels`      | A set of route labels used for selecting routes to associate with the Gateway.                                              |

### Kind: `HTTPRoute`

`HTTPRoute` defines HTTP rules for mapping requests from a `Gateway` to Kubernetes Services. 

Register the `HTTPRoute` [definition](../../reference/dynamic-configuration/kubernetes-gateway.md#definitions) in the
Kubernetes cluster before creating `HTTPRoute` objects.

!!! info "Declaring HTTPRoute"

    ```yaml
    kind: HTTPRoute
    apiVersion: networking.x-k8s.io/v1alpha1
    metadata:
      name: http-app-1
      namespace: default
      labels:                                   # [1]
        app: foo
    spec:
      hostnames:                                # [2]
        - "whoami"
      rules:                                    # [3]
        - matches:                              # [4]
            - path:                             # [5]
                type: Exact                     # [6]
                value: /bar                     # [7]
            - headers:                          # [8]
                type: Exact                     # [9]
                values:                         # [10]
                  - foo: bar
          forwardTo:                            # [11]
            - serviceName: whoami               # [12]
              weight: 1                         # [13]
              port: 80                          # [14]
            - backendRef:                       # [15]
                group: traefik.containo.us      # [16]
                kind: TraefikService            # [17]
                name: api@internal              # [18]
              port: 80
              weight: 1
    ```

| Ref  | Attribute     | Description                                                                                                                                                                                                                               |
|------|---------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| [1]  | `labels`      | Labels to match with the `Gateway` labelselector.                                                                                                                                                                                         |
| [2]  | `hostnames`   | A set of hostname that should match against the HTTP Host header to select a HTTPRoute to process the request.                                                                                                                            |
| [3]  | `rules`       | A list of HTTP matchers, filters and actions.                                                                                                                                                                                             |
| [4]  | `matches`     | Conditions used for matching the rule against incoming HTTP requests. Each match is independent, i.e. this rule will be matched if **any** one of the matches is satisfied.                                                               |
| [5]  | `path`        | An HTTP request path matcher. If this field is not specified, a default prefix match on the "/" path is provided.                                                                                                                         |
| [6]  | `type`        | Type of match against the path Value (supported types: `Exact`, `Prefix`).                                                                                                                                                                |
| [7]  | `value`       | The value of the HTTP path to match against.                                                                                                                                                                                              |
| [8]  | `headers`     | Conditions to select a HTTP route by matching HTTP request headers.                                                                                                                                                                       |
| [9]  | `type`        | Type of match for the HTTP request header match against the `values` (supported types: `Exact`).                                                                                                                                          |
| [10] | `values`      | A map of HTTP Headers to be matched. It MUST contain at least one entry.                                                                                                                                                                  |
| [11] | `forwardTo`   | The upstream target(s) where the request should be sent.                                                                                                                                                                                  |
| [12] | `serviceName` | The name of the referent service.                                                                                                                                                                                                         |
| [13] | `weight`      | The proportion of traffic forwarded to a targetRef, computed as weight/(sum of all weights in targetRefs).                                                                                                                                |
| [14] | `port`        | The port of the referent service.                                                                                                                                                                                                         |
| [15] | `backendRef`  | The BackendRef is a reference to a backend (API object within a known namespace) to forward matched requests to. If both BackendRef and ServiceName are specified, ServiceName will be given precedence. Only `TraefikService` is supported. |
| [16] | `group`       | Group is the group of the referent. Only `traefik.containo.us` value is supported.                                                                                                                                                        |
| [17] | `kind`        | Kind is kind of the referent. Only `TraefikService` value is supported.                                                                                                                                                                   |
| [18] | `name`        | Name is the name of the referent.                                                                                                                                                                                                         |
