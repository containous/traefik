# Traefik & Kubernetes

The Kubernetes Ingress Controller.
{: .subtitle }

## Routing Configuration

The provider then watches for incoming ingresses events, such as the example below,
and derives the corresponding dynamic configuration from it,
which in turn will create the resulting routers, services, handlers, etc.

## Configuration Example

??? example "Configuring Kubernetes Ingress Controller"

    ```yaml tab="RBAC"
    ---
    kind: ClusterRole
    apiVersion: rbac.authorization.k8s.io/v1beta1
    metadata:
      name: traefik-ingress-controller
    rules:
      - apiGroups:
          - ""
        resources:
          - services
          - endpoints
          - secrets
        verbs:
          - get
          - list
          - watch
      - apiGroups:
          - extensions
          - networking.k8s.io
        resources:
          - ingresses
          - ingressclasses
        verbs:
          - get
          - list
          - watch
      - apiGroups:
          - extensions
        resources:
          - ingresses/status
        verbs:
          - update

    ---
    kind: ClusterRoleBinding
    apiVersion: rbac.authorization.k8s.io/v1beta1
    metadata:
      name: traefik-ingress-controller
    roleRef:
      apiGroup: rbac.authorization.k8s.io
      kind: ClusterRole
      name: traefik-ingress-controller
    subjects:
      - kind: ServiceAccount
        name: traefik-ingress-controller
        namespace: default
    ```

    ```yaml tab="Ingress"
    kind: Ingress
    apiVersion: networking.k8s.io/v1beta1
    metadata:
      name: myingress
      annotations:
        traefik.ingress.kubernetes.io/router.entrypoints: web

    spec:
      rules:
        - host: example.com
          http:
            paths:
              - path: /bar
                backend:
                  serviceName: whoami
                  servicePort: 80
              - path: /foo
                backend:
                  serviceName: whoami
                  servicePort: 80
    ```
    
    ```yaml tab="Ingress Kubernetes v1.19+"
    kind: Ingress
    apiVersion: networking.k8s.io/v1
    metadata:
      name: myingress
      annotations:
        traefik.ingress.kubernetes.io/router.entrypoints: web
    
    spec:
      rules:
        - host: example.com
          http:
            paths:
              - path: /bar
                backend:
                  service:
                    name:  whoami
                    port:
                      number: 80
              - path: /foo
                backend:
                  service:
                    name:  whoami
                    port:
                      number: 80
    ```
    
    ```yaml tab="Traefik"
    apiVersion: v1
    kind: ServiceAccount
    metadata:
      name: traefik-ingress-controller

    ---
    kind: Deployment
    apiVersion: apps/v1
    metadata:
      name: traefik
      labels:
        app: traefik

    spec:
      replicas: 1
      selector:
        matchLabels:
          app: traefik
      template:
        metadata:
          labels:
            app: traefik
        spec:
          serviceAccountName: traefik-ingress-controller
          containers:
            - name: traefik
              image: traefik:v2.5
              args:
                - --entrypoints.web.address=:80
                - --providers.kubernetesingress
              ports:
                - name: web
                  containerPort: 80

    ---
    apiVersion: v1
    kind: Service
    metadata:
      name: traefik
    spec:
      type: LoadBalancer
      selector:
        app: traefik
      ports:
        - protocol: TCP
          port: 80
          name: web
          targetPort: 80
    ```

    ```yaml tab="Whoami"
    kind: Deployment
    apiVersion: apps/v1
    metadata:
      name: whoami
      labels:
        app: traefiklabs
        name: whoami

    spec:
      replicas: 2
      selector:
        matchLabels:
          app: traefiklabs
          task: whoami
      template:
        metadata:
          labels:
            app: traefiklabs
            task: whoami
        spec:
          containers:
            - name: whoami
              image: traefik/whoami
              ports:
                - containerPort: 80

    ---
    apiVersion: v1
    kind: Service
    metadata:
      name: whoami

    spec:
      ports:
        - name: http
          port: 80
      selector:
        app: traefiklabs
        task: whoami
    ```

## Annotations

#### On Ingress

??? info "`traefik.ingress.kubernetes.io/router.entrypoints`"

    See [entry points](../routers/index.md#entrypoints) for more information.

    ```yaml
    traefik.ingress.kubernetes.io/router.entrypoints: ep1,ep2
    ```

??? info "`traefik.ingress.kubernetes.io/router.middlewares`"

    See [middlewares](../routers/index.md#middlewares) and [middlewares overview](../../middlewares/overview.md) for more information.

    ```yaml
    traefik.ingress.kubernetes.io/router.middlewares: auth@file,prefix@kubernetescrd,cb@file
    ```

??? info "`traefik.ingress.kubernetes.io/router.priority`"

    See [priority](../routers/index.md#priority) for more information.

    ```yaml
    traefik.ingress.kubernetes.io/router.priority: "42"
    ```

??? info "`traefik.ingress.kubernetes.io/router.pathmatcher`"

    Overrides the default router rule type used for a path.  
    Only path-related matcher name can be specified: `Path`, `PathPrefix`.

    Default `PathPrefix`

    ```yaml
    traefik.ingress.kubernetes.io/router.pathmatcher: Path
    ```

??? info "`traefik.ingress.kubernetes.io/router.tls`"

    See [tls](../routers/index.md#tls) for more information.

    ```yaml
    traefik.ingress.kubernetes.io/router.tls: "true"
    ```

??? info "`traefik.ingress.kubernetes.io/router.tls.certresolver`"

    See [certResolver](../routers/index.md#certresolver) for more information.

    ```yaml
    traefik.ingress.kubernetes.io/router.tls.certresolver: myresolver
    ```

??? info "`traefik.ingress.kubernetes.io/router.tls.domains.n.main`"

    See [domains](../routers/index.md#domains) for more information.

    ```yaml
    traefik.ingress.kubernetes.io/router.tls.domains.0.main: example.org
    ```

??? info "`traefik.ingress.kubernetes.io/router.tls.domains.n.sans`"

    See [domains](../routers/index.md#domains) for more information.

    ```yaml
    traefik.ingress.kubernetes.io/router.tls.domains.0.sans: test.example.org,dev.example.org
    ```

??? info "`traefik.ingress.kubernetes.io/router.tls.options`"

    See [options](../routers/index.md#options) for more information.

    ```yaml
    traefik.ingress.kubernetes.io/router.tls.options: foobar
    ```

#### On Service

??? info "`traefik.ingress.kubernetes.io/service.serversscheme`"

    Overrides the default scheme.

    ```yaml
    traefik.ingress.kubernetes.io/service.serversscheme: h2c
    ```

??? info "`traefik.ingress.kubernetes.io/service.serverstransport`"

    See [ServersTransport](../services/index.md#serverstransport) for more information.

    ```yaml
    traefik.ingress.kubernetes.io/service.serverstransport: foobar@file
    ```

??? info "`traefik.ingress.kubernetes.io/service.passhostheader`"

    See [pass Host header](../services/index.md#pass-host-header) for more information.

    ```yaml
    traefik.ingress.kubernetes.io/service.passhostheader: "true"
    ```

??? info "`traefik.ingress.kubernetes.io/service.sticky.cookie`"

    See [sticky sessions](../services/index.md#sticky-sessions) for more information.

    ```yaml
    traefik.ingress.kubernetes.io/service.sticky.cookie: "true"
    ```

??? info "`traefik.ingress.kubernetes.io/service.sticky.cookie.name`"

    See [sticky sessions](../services/index.md#sticky-sessions) for more information.

    ```yaml
    traefik.ingress.kubernetes.io/service.sticky.cookie.name: foobar
    ```

??? info "`traefik.ingress.kubernetes.io/service.sticky.cookie.secure`"

    See [sticky sessions](../services/index.md#sticky-sessions) for more information.

    ```yaml
    traefik.ingress.kubernetes.io/service.sticky.cookie.secure: "true"
    ```

??? info "`traefik.ingress.kubernetes.io/service.sticky.cookie.samesite`"

    See [sticky sessions](../services/index.md#sticky-sessions) for more information.

    ```yaml
    traefik.ingress.kubernetes.io/service.sticky.cookie.samesite: "none"
    ```

??? info "`traefik.ingress.kubernetes.io/service.sticky.cookie.httponly`"

    See [sticky sessions](../services/index.md#sticky-sessions) for more information.

    ```yaml
    traefik.ingress.kubernetes.io/service.sticky.cookie.httponly: "true"
    ```

## Path Types on Kubernetes 1.18+

If the Kubernetes cluster version is 1.18+,
the new `pathType` property can be leveraged to define the rules matchers:

- `Exact`: This path type forces the rule matcher to `Path`
- `Prefix`: This path type forces the rule matcher to `PathPrefix`

Please see [this documentation](https://kubernetes.io/docs/concepts/services-networking/ingress/#path-types) for more information.

!!! warning "Multiple Matches"
    In the case of multiple matches, Traefik will not ensure the priority of a Path matcher over a PathPrefix matcher,
    as stated in [this documentation](https://kubernetes.io/docs/concepts/services-networking/ingress/#multiple-matches).

## TLS

### Enabling TLS via HTTP Options on Entrypoint

TLS can be enabled through the [HTTP options](../entrypoints.md#tls) of an Entrypoint:

```bash tab="CLI"
# Static configuration
--entrypoints.websecure.address=:443
--entrypoints.websecure.http.tls
```

```yaml tab="File (YAML)"
# Static configuration
entryPoints:
  websecure:
    address: ':443'
    http:
      tls: {}
```

```toml tab="File (TOML)"
# Static configuration
[entryPoints.websecure]
  address = ":443"

    [entryPoints.websecure.http.tls]
```

This way, any Ingress attached to this Entrypoint will have TLS termination by default.

??? example "Configuring Kubernetes Ingress Controller with TLS on Entrypoint"

    ```yaml tab="RBAC"
    ---
    kind: ClusterRole
    apiVersion: rbac.authorization.k8s.io/v1beta1
    metadata:
      name: traefik-ingress-controller
    rules:
      - apiGroups:
          - ""
        resources:
          - services
          - endpoints
          - secrets
        verbs:
          - get
          - list
          - watch
      - apiGroups:
          - extensions
          - networking.k8s.io
        resources:
          - ingresses
          - ingressclasses
        verbs:
          - get
          - list
          - watch
      - apiGroups:
          - extensions
        resources:
          - ingresses/status
        verbs:
          - update

    ---
    kind: ClusterRoleBinding
    apiVersion: rbac.authorization.k8s.io/v1beta1
    metadata:
      name: traefik-ingress-controller
    roleRef:
      apiGroup: rbac.authorization.k8s.io
      kind: ClusterRole
      name: traefik-ingress-controller
    subjects:
      - kind: ServiceAccount
        name: traefik-ingress-controller
        namespace: default
    ```

    ```yaml tab="Ingress"
    kind: Ingress
    apiVersion: networking.k8s.io/v1beta1
    metadata:
      name: myingress
      annotations:
        traefik.ingress.kubernetes.io/router.entrypoints: websecure

    spec:
      rules:
        - host: example.com
          http:
            paths:
              - path: /bar
                backend:
                  serviceName: whoami
                  servicePort: 80
              - path: /foo
                backend:
                  serviceName: whoami
                  servicePort: 80
    ```
    
    ```yaml tab="Ingress Kubernetes v1.19+"
    kind: Ingress
    apiVersion: networking.k8s.io/v1
    metadata:
      name: myingress
      annotations:
        traefik.ingress.kubernetes.io/router.entrypoints: websecure
    
    spec:
      rules:
        - host: example.com
          http:
            paths:
              - path: /bar
                backend:
                  service:
                    name:  whoami
                    port:
                      number: 80
              - path: /foo
                backend:
                  service:
                    name:  whoami
                    port:
                      number: 80
    ```
    
    ```yaml tab="Traefik"
    apiVersion: v1
    kind: ServiceAccount
    metadata:
      name: traefik-ingress-controller

    ---
    kind: Deployment
    apiVersion: apps/v1
    metadata:
      name: traefik
      labels:
        app: traefik

    spec:
      replicas: 1
      selector:
        matchLabels:
          app: traefik
      template:
        metadata:
          labels:
            app: traefik
        spec:
          serviceAccountName: traefik-ingress-controller
          containers:
            - name: traefik
              image: traefik:v2.5
              args:
                - --entrypoints.websecure.address=:443
                - --entrypoints.websecure.http.tls
                - --providers.kubernetesingress
              ports:
                - name: websecure
                  containerPort: 443

    ---
    apiVersion: v1
    kind: Service
    metadata:
      name: traefik
    spec:
      type: LoadBalancer
      selector:
        app: traefik
      ports:
        - protocol: TCP
          port: 443
          name: websecure
          targetPort: 443
    ```

    ```yaml tab="Whoami"
    kind: Deployment
    apiVersion: apps/v1
    metadata:
      name: whoami
      labels:
        app: traefiklabs
        name: whoami

    spec:
      replicas: 2
      selector:
        matchLabels:
          app: traefiklabs
          task: whoami
      template:
        metadata:
          labels:
            app: traefiklabs
            task: whoami
        spec:
          containers:
            - name: whoami
              image: traefik/whoami
              ports:
                - containerPort: 80

    ---
    apiVersion: v1
    kind: Service
    metadata:
      name: whoami

    spec:
      ports:
        - name: http
          port: 80
      selector:
        app: traefiklabs
        task: whoami
    ```

### Enabling TLS via Annotations

To enable TLS on the underlying router created from an Ingress, one should configure it through annotations:

```yaml
traefik.ingress.kubernetes.io/router.tls: "true"
```

For more options, please refer to the available [annotations](#on-ingress).

??? example "Configuring Kubernetes Ingress Controller with TLS"

    ```yaml tab="RBAC"
    ---
    kind: ClusterRole
    apiVersion: rbac.authorization.k8s.io/v1beta1
    metadata:
      name: traefik-ingress-controller
    rules:
      - apiGroups:
          - ""
        resources:
          - services
          - endpoints
          - secrets
        verbs:
          - get
          - list
          - watch
      - apiGroups:
          - extensions
          - networking.k8s.io
        resources:
          - ingresses
          - ingressclasses
        verbs:
          - get
          - list
          - watch
      - apiGroups:
          - extensions
        resources:
          - ingresses/status
        verbs:
          - update

    ---
    kind: ClusterRoleBinding
    apiVersion: rbac.authorization.k8s.io/v1beta1
    metadata:
      name: traefik-ingress-controller
    roleRef:
      apiGroup: rbac.authorization.k8s.io
      kind: ClusterRole
      name: traefik-ingress-controller
    subjects:
      - kind: ServiceAccount
        name: traefik-ingress-controller
        namespace: default
    ```

    ```yaml tab="Ingress"
    kind: Ingress
    apiVersion: networking.k8s.io/v1beta1
    metadata:
      name: myingress
      annotations:
        traefik.ingress.kubernetes.io/router.entrypoints: websecure
        traefik.ingress.kubernetes.io/router.tls: true

    spec:
      rules:
        - host: example.com
          http:
            paths:
              - path: /bar
                backend:
                  serviceName: whoami
                  servicePort: 80
              - path: /foo
                backend:
                  serviceName: whoami
                  servicePort: 80
    ```
    
    ```yaml tab="Ingress Kubernetes v1.19+"
    kind: Ingress
    apiVersion: networking.k8s.io/v1
    metadata:
      name: myingress
      annotations:
        traefik.ingress.kubernetes.io/router.entrypoints: websecure
        traefik.ingress.kubernetes.io/router.tls: true
    
    spec:
      rules:
        - host: example.com
          http:
            paths:
              - path: /bar
                backend:
                  service:
                    name:  whoami
                    port:
                      number: 80
              - path: /foo
                backend:
                  service:
                    name:  whoami
                    port:
                      number: 80
    ```
    
    ```yaml tab="Traefik"
    apiVersion: v1
    kind: ServiceAccount
    metadata:
      name: traefik-ingress-controller

    ---
    kind: Deployment
    apiVersion: apps/v1
    metadata:
      name: traefik
      labels:
        app: traefik

    spec:
      replicas: 1
      selector:
        matchLabels:
          app: traefik
      template:
        metadata:
          labels:
            app: traefik
        spec:
          serviceAccountName: traefik-ingress-controller
          containers:
            - name: traefik
              image: traefik:v2.5
              args:
                - --entrypoints.websecure.address=:443
                - --providers.kubernetesingress
              ports:
                - name: websecure
                  containerPort: 443

    ---
    apiVersion: v1
    kind: Service
    metadata:
      name: traefik
    spec:
      type: LoadBalancer
      selector:
        app: traefik
      ports:
        - protocol: TCP
          port: 443
          name: websecure
          targetPort: 443
    ```

    ```yaml tab="Whoami"
    kind: Deployment
    apiVersion: apps/v1
    metadata:
      name: whoami
      labels:
        app: traefiklabs
        name: whoami

    spec:
      replicas: 2
      selector:
        matchLabels:
          app: traefiklabs
          task: whoami
      template:
        metadata:
          labels:
            app: traefiklabs
            task: whoami
        spec:
          containers:
            - name: whoami
              image: traefik/whoami
              ports:
                - containerPort: 80

    ---
    apiVersion: v1
    kind: Service
    metadata:
      name: whoami

    spec:
      ports:
        - name: http
          port: 80
      selector:
        app: traefiklabs
        task: whoami
    ```

### Certificates Management

??? example "Using a secret"

    ```yaml tab="Ingress"
    kind: Ingress
    apiVersion: networking.k8s.io/v1beta1
    metadata:
      name: foo
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
      # Only selects which certificate(s) should be loaded from the secret, in order to terminate TLS.
      # Doesn't enable TLS for that ingress (hence for the underlying router).
      # Please see the TLS annotations on ingress made for that purpose.
      tls:
      - secretName: supersecret
    ```

    ```yaml tab="Ingress Kubernetes v1.19+"
    kind: Ingress
    apiVersion: networking.k8s.io/v1
    metadata:
      name: foo
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
      # Only selects which certificate(s) should be loaded from the secret, in order to terminate TLS.
      # Doesn't enable TLS for that ingress (hence for the underlying router).
      # Please see the TLS annotations on ingress made for that purpose.
      tls:
      - secretName: supersecret
    ```
      
    ```yaml tab="Secret"
    apiVersion: v1
    kind: Secret
    metadata:
      name: supersecret

    data:
      tls.crt: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCi0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0=
      tls.key: LS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0tCi0tLS0tRU5EIFBSSVZBVEUgS0VZLS0tLS0=
    ```

TLS certificates can be managed in Secrets objects.

!!! info

    Only TLS certificates provided by users can be stored in Kubernetes Secrets.
    [Let's Encrypt](../../https/acme.md) certificates cannot be managed in Kubernetes Secrets yet.

### Communication Between Traefik and Pods

Traefik automatically requests endpoint information based on the service provided in the ingress spec.
Although Traefik will connect directly to the endpoints (pods),
it still checks the service port to see if TLS communication is required.

There are 3 ways to configure Traefik to use https to communicate with pods:

1. If the service port defined in the ingress spec is `443` (note that you can still use `targetPort` to use a different port on your pod).
1. If the service port defined in the ingress spec has a name that starts with https (such as `https-api`, `https-web` or just `https`).
1. If the service spec includes the annotation `traefik.ingress.kubernetes.io/service.serversscheme: https`.

If either of those configuration options exist, then the backend communication protocol is assumed to be TLS,
and will connect via TLS automatically.

!!! info

    Please note that by enabling TLS communication between traefik and your pods,
    you will have to have trusted certificates that have the proper trust chain and IP subject name.
    If this is not an option, you may need to skip TLS certificate verification.
    See the [insecureSkipVerify](../../routing/overview.md#insecureskipverify) setting for more details.

## Global Default Backend Ingresses

Ingresses can be created that look like the following:

```yaml tab="Ingress"
apiVersion: networking.k8s.io/v1beta1
kind: Ingress
metadata:
 name: cheese

spec:
  defaultBackend:
    serviceName: stilton
    serverPort: 80
```

```yaml tab="Ingress Kubernetes v1.19+"
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
 name: cheese

spec:
  defaultBackend:
    service:
      name: stilton
      port:
        number: 80
```

This ingress follows the Global Default Backend property of ingresses.
This will allow users to create a "default router" that will match all unmatched requests.

!!! info

    Due to Traefik's use of priorities, you may have to set this ingress priority lower than other ingresses in your environment,
    to avoid this global ingress from satisfying requests that could match other ingresses.

    To do this, use the `traefik.ingress.kubernetes.io/router.priority` annotation (as seen in [Annotations on Ingress](#on-ingress)) on your ingresses accordingly.
