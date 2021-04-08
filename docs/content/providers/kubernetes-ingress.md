# Traefik & Kubernetes

The Kubernetes Ingress Controller.
{: .subtitle }

The Traefik Kubernetes Ingress provider is a Kubernetes Ingress controller; that is to say,
it manages access to cluster services by supporting the [Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/) specification.

## Requirements

Traefik supports `1.14+` Kubernetes clusters.

## Routing Configuration

See the dedicated section in [routing](../routing/providers/kubernetes-ingress.md).

## Enabling and Using the Provider

You can enable the provider in the static configuration:

```toml tab="File (TOML)"
[providers.kubernetesIngress]
```

```yaml tab="File (YAML)"
providers:
  kubernetesIngress: {}
```

```bash tab="CLI"
--providers.kubernetesingress=true
```

The provider then watches for incoming ingresses events, such as the example below,
and derives the corresponding dynamic configuration from it,
which in turn creates the resulting routers, services, handlers, etc.

```yaml tab="Ingress"
kind: Ingress
apiVersion: networking.k8s.io/v1beta1
metadata:
  name: "foo"
  namespace: production

spec:
  rules:
    - host: example.net
      http:
        paths:
          - path: /bar
            backend:
              serviceName: service1
              servicePort: 80
          - path: /foo
            backend:
              serviceName: service1
              servicePort: 80
```

```yaml tab="Ingress Kubernetes v1.19+"
kind: Ingress
apiVersion: networking.k8s.io/v1
metadata:
  name: "foo"
  namespace: production

spec:
  rules:
    - host: example.net
      http:
        paths:
          - path: /bar
            backend:
              service:
                name:  service1
                port:
                  number: 80
          - path: /foo
            backend:
              service:
                name:  service1
                port:
                  number: 80
```

## LetsEncrypt Support with the Ingress Provider

By design, Traefik is a stateless application,
meaning that it only derives its configuration from the environment it runs in,
without additional configuration.
For this reason, users can run multiple instances of Traefik at the same time to achieve HA,
as is a common pattern in the kubernetes ecosystem.

When using a single instance of Traefik Proxy with Let's Encrypt, you should encounter no issues.
However, this could be a single point of failure.
Unfortunately, it is not possible to run multiple instances of Traefik 2.0 with Let's Encrypt enabled,
because there is no way to ensure that the correct instance of Traefik receives the challenge request, and subsequent responses.
Previous versions of Traefik used a [KV store](https://doc.traefik.io/traefik/v1.7/configuration/acme/#storage) to attempt to achieve this,
but due to sub-optimal performance that feature was dropped in 2.0.

If you need Let's Encrypt with high availability in a Kubernetes environment,
we recommend using [Traefik Enterprise](https://traefik.io/traefik-enterprise/) which includes distributed Let's Encrypt as a supported feature.

If you want to keep using Traefik Proxy,
LetsEncrypt HA can be achieved by using a Certificate Controller such as [Cert-Manager](https://docs.cert-manager.io/en/latest/index.html).
When using Cert-Manager to manage certificates,
it creates secrets in your namespaces that can be referenced as TLS secrets in your [ingress objects](https://kubernetes.io/docs/concepts/services-networking/ingress/#tls).

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
[providers.kubernetesIngress]
  endpoint = "http://localhost:8080"
  # ...
```

```yaml tab="File (YAML)"
providers:
  kubernetesIngress:
    endpoint: "http://localhost:8080"
    # ...
```

```bash tab="CLI"
--providers.kubernetesingress.endpoint=http://localhost:8080
```

### `token`

_Optional, Default=""_

Bearer token used for the Kubernetes client configuration.

```toml tab="File (TOML)"
[providers.kubernetesIngress]
  token = "mytoken"
  # ...
```

```yaml tab="File (YAML)"
providers:
  kubernetesIngress:
    token: "mytoken"
    # ...
```

```bash tab="CLI"
--providers.kubernetesingress.token=mytoken
```

### `certAuthFilePath`

_Optional, Default=""_

Path to the certificate authority file.
Used for the Kubernetes client configuration.

```toml tab="File (TOML)"
[providers.kubernetesIngress]
  certAuthFilePath = "/my/ca.crt"
  # ...
```

```yaml tab="File (YAML)"
providers:
  kubernetesIngress:
    certAuthFilePath: "/my/ca.crt"
    # ...
```

```bash tab="CLI"
--providers.kubernetesingress.certauthfilepath=/my/ca.crt
```

### `namespaces`

_Optional, Default: []_

Array of namespaces to watch.
If left empty, watches all namespaces if the value of `namespaces`.

```toml tab="File (TOML)"
[providers.kubernetesIngress]
  namespaces = ["default", "production"]
  # ...
```

```yaml tab="File (YAML)"
providers:
  kubernetesIngress:
    namespaces:
      - "default"
      - "production"
    # ...
```

```bash tab="CLI"
--providers.kubernetesingress.namespaces=default,production
```

### `labelSelector`

_Optional, Default: ""_

A label selector can be defined to filter on specific Ingress objects only.
If left empty, Traefik processes all Ingress objects in the configured namespaces.

See [label-selectors](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors) for details.

```toml tab="File (TOML)"
[providers.kubernetesIngress]
  labelSelector = "app=traefik"
  # ...
```

```yaml tab="File (YAML)"
providers:
  kubernetesIngress:
    labelselector: "app=traefik"
    # ...
```

```bash tab="CLI"
--providers.kubernetesingress.labelselector="app=traefik"
```

### `ingressClass`

_Optional, Default: ""_

Value of `kubernetes.io/ingress.class` annotation that identifies Ingress objects to be processed.

If the parameter is set, only Ingresses containing an annotation with the same value are processed.
Otherwise, Ingresses missing the annotation, having an empty value, or the value `traefik` are processed.

??? info "Kubernetes 1.18+"

    If the Kubernetes cluster version is 1.18+,
    the new `IngressClass` resource can be leveraged to identify Ingress objects that should be processed.
    In that case, Traefik will look for an `IngressClass` in the cluster with the controller value equal to *traefik.io/ingress-controller*.

    In addition to the controller value matching mechanism, the property `ingressClass` (if set) will be used to select IngressClasses by applying a strict matching on their name.

    Please see [this article](https://kubernetes.io/blog/2020/04/02/improvements-to-the-ingress-api-in-kubernetes-1.18/) for more information or the example below.

    ```yaml tab="IngressClass"
    apiVersion: networking.k8s.io/v1beta1
    kind: IngressClass
    metadata:
      name: traefik-lb
    spec:
      controller: traefik.io/ingress-controller
    ```

    ```yaml tab="Ingress"
    apiVersion: "networking.k8s.io/v1beta1"
    kind: "Ingress"
    metadata:
      name: "example-ingress"
    spec:
      ingressClassName: "traefik-lb"
      rules:
      - host: "*.example.com"
        http:
          paths:
          - path: "/example"
            backend:
              serviceName: "example-service"
              servicePort: 80
    ```

??? info "Kubernetes 1.19+"

    If the Kubernetes cluster version is 1.19+,
    prefer using the `networking.k8s.io/v1` [apiVersion](https://v1-19.docs.kubernetes.io/docs/setup/release/notes/#api-change) of `Ingress` and `IngressClass`.

    ```yaml tab="IngressClass"
    apiVersion: networking.k8s.io/v1
    kind: IngressClass
    metadata:
      name: traefik-lb
    spec:
      controller: traefik.io/ingress-controller
    ```

    ```yaml tab="Ingress"
    apiVersion: "networking.k8s.io/v1"
    kind: "Ingress"
    metadata:
      name: "example-ingress"
    spec:
      ingressClassName: "traefik-lb"
      rules:
      - host: "*.example.com"
        http:
          paths:
          - path: "/example"
            backend:
              service:
                name: "example-service"
                port:
                    number: 80
    ```

```toml tab="File (TOML)"
[providers.kubernetesIngress]
  ingressClass = "traefik-internal"
  # ...
```

```yaml tab="File (YAML)"
providers:
  kubernetesIngress:
    ingressClass: "traefik-internal"
    # ...
```

```bash tab="CLI"
--providers.kubernetesingress.ingressclass=traefik-internal
```

### `ingressEndpoint`

#### `hostname`

_Optional, Default: ""_

Hostname used for Kubernetes Ingress endpoints.

```toml tab="File (TOML)"
[providers.kubernetesIngress.ingressEndpoint]
  hostname = "example.net"
  # ...
```

```yaml tab="File (YAML)"
providers:
  kubernetesIngress:
    ingressEndpoint:
      hostname: "example.net"
    # ...
```

```bash tab="CLI"
--providers.kubernetesingress.ingressendpoint.hostname=example.net
```

#### `ip`

_Optional, Default: ""_

IP used for Kubernetes Ingress endpoints.

```toml tab="File (TOML)"
[providers.kubernetesIngress.ingressEndpoint]
  ip = "1.2.3.4"
  # ...
```

```yaml tab="File (YAML)"
providers:
  kubernetesIngress:
    ingressEndpoint:
      ip: "1.2.3.4"
    # ...
```

```bash tab="CLI"
--providers.kubernetesingress.ingressendpoint.ip=1.2.3.4
```

#### `publishedService`

_Optional, Default: ""_

Published Kubernetes Service to copy status from.
Format: `namespace/servicename`.

```toml tab="File (TOML)"
[providers.kubernetesIngress.ingressEndpoint]
  publishedService = "namespace/foo-service"
  # ...
```

```yaml tab="File (YAML)"
providers:
  kubernetesIngress:
    ingressEndpoint:
      publishedService: "namespace/foo-service"
    # ...
```

```bash tab="CLI"
--providers.kubernetesingress.ingressendpoint.publishedservice=namespace/foo-service
```

### `throttleDuration`

_Optional, Default: 0_

The `throttleDuration` option defines how often the provider is allowed to handle events from Kubernetes. This prevents
a Kubernetes cluster that updates many times per second from continuously changing your Traefik configuration.

If left empty, the provider does not apply any throttling and does not drop any Kubernetes events.

The value of `throttleDuration` should be provided in seconds or as a valid duration format,
see [time.ParseDuration](https://golang.org/pkg/time/#ParseDuration).

```toml tab="File (TOML)"
[providers.kubernetesIngress]
  throttleDuration = "10s"
  # ...
```

```yaml tab="File (YAML)"
providers:
  kubernetesIngress:
    throttleDuration: "10s"
    # ...
```

```bash tab="CLI"
--providers.kubernetesingress.throttleDuration=10s
```

### Further

To learn more about the various aspects of the Ingress specification that Traefik supports,
many examples of Ingresses definitions are located in the test [examples](https://github.com/traefik/traefik/tree/v2.4/pkg/provider/kubernetes/ingress/fixtures) of the Traefik repository.
