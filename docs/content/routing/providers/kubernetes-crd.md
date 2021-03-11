# Traefik & Kubernetes

The Kubernetes Ingress Controller, The Custom Resource Way.
{: .subtitle }

## Configuration Examples

??? example "Configuring KubernetesCRD and Deploying/Exposing Services"

    ```yaml tab="Resource Definition"
    # All resources definition must be declared
    --8<-- "content/reference/dynamic-configuration/kubernetes-crd-definition-v1.yml"
    ```
    
    ```yaml tab="RBAC"
    --8<-- "content/reference/dynamic-configuration/kubernetes-crd-rbac.yml"
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
              image: traefik:v2.4
              args:
                - --log.level=DEBUG
                - --api
                - --api.insecure
                - --entrypoints.web.address=:80
                - --entrypoints.tcpep.address=:8000
                - --entrypoints.udpep.address=:9000/udp
                - --providers.kubernetescrd
              ports:
                - name: web
                  containerPort: 80
                - name: admin
                  containerPort: 8080
                - name: tcpep
                  containerPort: 8000
                - name: udpep
                  containerPort: 9000
    
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
        - protocol: TCP
          port: 8080
          name: admin
          targetPort: 8080
        - protocol: TCP
          port: 8000
          name: tcpep
          targetPort: 8000
     
    ---
    apiVersion: v1
    kind: Service
    metadata:
      name: traefikudp
    spec:
      type: LoadBalancer
      selector:
        app: traefik
      ports:
        - protocol: UDP
          port: 9000
          name: udpep
          targetPort: 9000
    ```
    
    ```yaml tab="IngressRoute"
    apiVersion: traefik.containo.us/v1alpha1
    kind: IngressRoute
    metadata:
      name: myingressroute
      namespace: default
    
    spec:
      entryPoints:
        - web
    
      routes:
      - match: Host(`foo`) && PathPrefix(`/bar`)
        kind: Rule
        services:
        - name: whoami
          port: 80
    
    ---
    apiVersion: traefik.containo.us/v1alpha1
    kind: IngressRouteTCP
    metadata:
      name: ingressroute.tcp
      namespace: default
    
    spec:
      entryPoints:
        - tcpep
      routes:
      - match: HostSNI(`bar`)
        kind: Rule
        services:
          - name: whoamitcp
            port: 8080
    
    ---
    apiVersion: traefik.containo.us/v1alpha1
    kind: IngressRouteUDP
    metadata:
      name: ingressroute.udp
      namespace: default
    
    spec:
      entryPoints:
        - udpep
      routes:
      - kind: Rule
        services:
          - name: whoamiudp
            port: 8080
    ```
    
    ```yaml tab="Whoami"
    kind: Deployment
    apiVersion: apps/v1
    metadata:
      name: whoami
      namespace: default
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
      namespace: default
    
    spec:
      ports:
        - name: http
          port: 80
      selector:
        app: traefiklabs
        task: whoami
    
    ---
    kind: Deployment
    apiVersion: apps/v1
    metadata:
      name: whoamitcp
      namespace: default
      labels:
        app: traefiklabs
        name: whoamitcp
    
    spec:
      replicas: 2
      selector:
        matchLabels:
          app: traefiklabs
          task: whoamitcp
      template:
        metadata:
          labels:
            app: traefiklabs
            task: whoamitcp
        spec:
          containers:
            - name: whoamitcp
              image: traefik/whoamitcp
              ports:
                - containerPort: 8080
    
    ---
    apiVersion: v1
    kind: Service
    metadata:
      name: whoamitcp
      namespace: default
    
    spec:
      ports:
        - protocol: TCP
          port: 8080
      selector:
        app: traefiklabs
        task: whoamitcp
    
    ---
    kind: Deployment
    apiVersion: apps/v1
    metadata:
      name: whoamiudp
      namespace: default
      labels:
        app: traefiklabs
        name: whoamiudp
    
    spec:
      replicas: 2
      selector:
        matchLabels:
          app: traefiklabs
          task: whoamiudp
      template:
        metadata:
          labels:
            app: traefiklabs
            task: whoamiudp
        spec:
          containers:
            - name: whoamiudp
              image: traefik/whoamiudp:latest
              ports:
                - containerPort: 8080
    
    ---
    apiVersion: v1
    kind: Service
    metadata:
      name: whoamiudp
      namespace: default
    
    spec:
      ports:
        - port: 8080
      selector:
        app: traefiklabs
        task: whoamiudp
    ```

## Routing Configuration

### Custom Resource Definition (CRD)

* You can find an exhaustive list, generated from Traefik's source code, of the custom resources and their attributes in [the reference page](../../reference/dynamic-configuration/kubernetes-crd.md).
* Validate that [the prerequisites](../../providers/kubernetes-crd.md#configuration-requirements) are fulfilled before using the Traefik custom resources.
* Traefik CRDs are building blocks that you can assemble according to your needs.
    
You can find an excerpt of the available custom resources in the table below:

| Kind                                       | Purpose                                                            | Concept Behind                                                 |
|--------------------------------------------|--------------------------------------------------------------------|----------------------------------------------------------------|
| [IngressRoute](#kind-ingressroute)         | HTTP Routing                                                       | [HTTP router](../routers/index.md#configuring-http-routers)    |
| [Middleware](#kind-middleware)             | Tweaks the HTTP requests before they are sent to your service      | [HTTP Middlewares](../../middlewares/overview.md)              |
| [TraefikService](#kind-traefikservice)     | Abstraction for HTTP loadbalancing/mirroring                       | [HTTP service](../services/index.md#configuring-http-services) |
| [IngressRouteTCP](#kind-ingressroutetcp)   | TCP Routing                                                        | [TCP router](../routers/index.md#configuring-tcp-routers)      |
| [IngressRouteUDP](#kind-ingressrouteudp)   | UDP Routing                                                        | [UDP router](../routers/index.md#configuring-udp-routers)      |
| [TLSOptions](#kind-tlsoption)              | Allows to configure some parameters of the TLS connection          | [TLSOptions](../../https/tls.md#tls-options)                   |
| [TLSStores](#kind-tlsstore)                | Allows to configure the default TLS store                          | [TLSStores](../../https/tls.md#certificates-stores)            |
| [ServersTransport](#kind-serverstransport) | Allows to configure the transport between Traefik and the backends | [ServersTransport](../../services/#serverstransport_1)         |

### Kind: `IngressRoute`

`IngressRoute` is the CRD implementation of a [Traefik HTTP router](../routers/index.md#configuring-http-routers).

Register the `IngressRoute` [kind](../../reference/dynamic-configuration/kubernetes-crd.md#definitions) in the Kubernetes cluster before creating `IngressRoute` objects.

!!! info "IngressRoute Attributes"

    ```yaml
    apiVersion: traefik.containo.us/v1alpha1
    kind: IngressRoute
    metadata:
      name: foo
      namespace: bar
    spec:
      entryPoints:                      # [1]
        - foo
      routes:                           # [2]
      - kind: Rule
        match: Host(`test.example.com`) # [3]
        priority: 10                    # [4]
        middlewares:                    # [5]
        - name: middleware1             # [6]
          namespace: default            # [7]
        services:                       # [8]
        - kind: Service
          name: foo
          namespace: default
          passHostHeader: true
          port: 80                      # [9]
          responseForwarding:
            flushInterval: 1ms
          scheme: https
          serversTransport: transport
          sticky:
            cookie:
              httpOnly: true
              name: cookie
              secure: true
              sameSite: none
          strategy: RoundRobin
          weight: 10
      tls:                              # [10]
        secretName: supersecret         # [11]
        options:                        # [12]
          name: opt                     # [13]
          namespace: default            # [14]
        certResolver: foo               # [15]
        domains:                        # [16]
        - main: example.net             # [17]
          sans:                         # [18]
          - a.example.net
          - b.example.net
    ```

| Ref  | Attribute                    | Purpose                                                                                                                                                                                                                  |
|------|------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| [1]  | `entryPoints`                | List of [entry points](../routers/index.md#entrypoints) names                                                                                                                                                            |
| [2]  | `routes`                     | List of routes                                                                                                                                                                                                           |
| [3]  | `routes[n].match`            | Defines the [rule](../routers/index.md#rule) corresponding to an underlying router.                                                                                                                                      |
| [4]  | `routes[n].priority`         | [Disambiguate](../routers/index.md#priority) rules of the same length, for route matching                                                                                                                                |
| [5]  | `routes[n].middlewares`      | List of reference to [Middleware](#kind-middleware)                                                                                                                                                                      |
| [6]  | `middlewares[n].name`        | Defines the [Middleware](#kind-middleware) name                                                                                                                                                                          |
| [7]  | `middlewares[n].namespace`   | Defines the [Middleware](#kind-middleware) namespace                                                                                                                                                                     |
| [8]  | `routes[n].services`         | List of any combination of [TraefikService](#kind-traefikservice) and reference to a [Kubernetes service](https://kubernetes.io/docs/concepts/services-networking/service/) (See below for `ExternalName Service` setup) |
| [9]  | `services[n].port`           | Defines the port of a [Kubernetes service](https://kubernetes.io/docs/concepts/services-networking/service/). This can be a reference to a named port.                                                                   |
| [10] | `tls`                        | Defines [TLS](../routers/index.md#tls) certificate configuration                                                                                                                                                         |
| [11] | `tls.secretName`             | Defines the [secret](https://kubernetes.io/docs/concepts/configuration/secret/) name used to store the certificate (in the `IngressRoute` namespace)                                                                     |
| [12] | `tls.options`                | Defines the reference to a [TLSOption](#kind-tlsoption)                                                                                                                                                                  |
| [13] | `options.name`               | Defines the [TLSOption](#kind-tlsoption) name                                                                                                                                                                            |
| [14] | `options.namespace`          | Defines the [TLSOption](#kind-tlsoption) namespace                                                                                                                                                                       |
| [15] | `tls.certResolver`           | Defines the reference to a [CertResolver](../routers/index.md#certresolver)                                                                                                                                              |
| [16] | `tls.domains`                | List of [domains](../routers/index.md#domains)                                                                                                                                                                           |
| [17] | `domains[n].main`            | Defines the main domain name                                                                                                                                                                                             |
| [18] | `domains[n].sans`            | List of SANs (alternative domains)                                                                                                                                                                                       |

??? example "Declaring an IngressRoute"

    ```yaml tab="IngressRoute"
    # All resources definition must be declared
    apiVersion: traefik.containo.us/v1alpha1
    kind: IngressRoute
    metadata:
      name: testName
      namespace: default
    spec:
      entryPoints:
        - web
      routes:
      - kind: Rule
        match: Host(`test.example.com`)
        middlewares:
        - name: middleware1
          namespace: default
        priority: 10
        services:
        - kind: Service
          name: foo
          namespace: default
          passHostHeader: true
          port: 80
          responseForwarding:
            flushInterval: 1ms
          scheme: https
          sticky:
            cookie:
              httpOnly: true
              name: cookie
              secure: true
          strategy: RoundRobin
          weight: 10
      tls:
        certResolver: foo
        domains:
        - main: example.net
          sans:
          - a.example.net
          - b.example.net
        options:
          name: opt
          namespace: default
        secretName: supersecret
    ```

    ```yaml tab="Middlewares"
    # All resources definition must be declared
    # Prefixing with /foo
    apiVersion: traefik.containo.us/v1alpha1
    kind: Middleware
    metadata:
      name: middleware1
      namespace: default
    spec:
      addPrefix:
        prefix: /foo
    ```

    ```yaml tab="TLSOption"
    apiVersion: traefik.containo.us/v1alpha1
    kind: TLSOption
    metadata:
      name: opt
      namespace: default
    
    spec:
      minVersion: VersionTLS12
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

!!! important "Configuring Backend Protocol"

    There are 3 ways to configure the backend protocol for communication between Traefik and your pods:
	
    - Setting the scheme explicitly (http/https/h2c)
    - Configuring the name of the kubernetes service port to start with https (https)
    - Setting the kubernetes service port to use port 443 (https)

    If you do not configure the above, Traefik will assume an http connection.
    

!!! important "Using Kubernetes ExternalName Service"

    Traefik backends creation needs a port to be set, however Kubernetes [ExternalName Service](https://kubernetes.io/fr/docs/concepts/services-networking/service/#externalname) could be defined without any port.
    Accordingly, Traefik supports defining a port in two ways:
    
    - only on `IngressRoute` service
    - on both sides, you'll be warned if the ports don't match, and the `IngressRoute` service port is used
    
    Thus, in case of two sides port definition, Traefik expects a match between ports.
    
    ??? example "Examples"
        
        ```yaml tab="IngressRoute"
        ---
        apiVersion: traefik.containo.us/v1alpha1
        kind: IngressRoute
        metadata:
          name: test.route
          namespace: default
        
        spec:
          entryPoints:
            - foo
        
          routes:
          - match: Host(`example.net`)
            kind: Rule
            services:
            - name: external-svc
              port: 80
        
        ---
        apiVersion: v1
        kind: Service
        metadata:
          name: external-svc
          namespace: default
        spec:
          externalName: external.domain
          type: ExternalName
        ```
        
        ```yaml tab="ExternalName Service"
        ---
        apiVersion: traefik.containo.us/v1alpha1
        kind: IngressRoute
        metadata:
          name: test.route
          namespace: default
        
        spec:
          entryPoints:
            - foo
        
          routes:
          - match: Host(`example.net`)
            kind: Rule
            services:
            - name: external-svc
        
        ---
        apiVersion: v1
        kind: Service
        metadata:
          name: external-svc
          namespace: default
        spec:
          externalName: external.domain
          type: ExternalName
          ports:
            - port: 80
        ```
        
        ```yaml tab="Both sides"
        ---
        apiVersion: traefik.containo.us/v1alpha1
        kind: IngressRoute
        metadata:
          name: test.route
          namespace: default
        
        spec:
          entryPoints:
            - foo
        
          routes:
          - match: Host(`example.net`)
            kind: Rule
            services:
            - name: external-svc
              port: 80
        
        ---
        apiVersion: v1
        kind: Service
        metadata:
          name: external-svc
          namespace: default
        spec:
          externalName: external.domain
          type: ExternalName
          ports:
            - port: 80
        ```

### Kind: `Middleware`

`Middleware` is the CRD implementation of a [Traefik middleware](../../middlewares/overview.md).

Register the `Middleware` [kind](../../reference/dynamic-configuration/kubernetes-crd.md#definitions) in the Kubernetes cluster before creating `Middleware` objects or referencing middlewares in the [`IngressRoute`](#kind-ingressroute) objects.

??? "Declaring and Referencing a Middleware"
    
    ```yaml tab="Middleware"
    apiVersion: traefik.containo.us/v1alpha1
    kind: Middleware
    metadata:
      name: stripprefix
      namespace: foo
    
    spec:
      stripPrefix:
        prefixes:
          - /stripit
    ```
    
    ```yaml tab="IngressRoute"
    apiVersion: traefik.containo.us/v1alpha1
    kind: IngressRoute
    metadata:
      name: ingressroutebar
    
    spec:
      entryPoints:
        - web
      routes:
      - match: Host(`example.com`) && PathPrefix(`/stripit`)
        kind: Rule
        services:
        - name: whoami
          port: 80
        middlewares:
        - name: stripprefix
          namespace: foo
    ```

!!! important "Cross-provider namespace"

    As Kubernetes also has its own notion of namespace, one should not confuse the kubernetes namespace of a resource
    (in the reference to the middleware) with the [provider namespace](../../providers/overview.md#provider-namespace),
    when the definition of the middleware comes from another provider.
    In this context, specifying a namespace when referring to the resource does not make any sense, and will be ignored.
    Additionally, when you want to reference a Middleware from the CRD Provider,
    you have to append the namespace of the resource in the resource-name as Traefik appends the namespace internally automatically.

More information about available middlewares in the dedicated [middlewares section](../../middlewares/overview.md).

### Kind: `TraefikService`

`TraefikService` is the CRD implementation of a ["Traefik Service"](../services/index.md).

Register the `TraefikService` [kind](../../reference/dynamic-configuration/kubernetes-crd.md#definitions) in the Kubernetes cluster before creating `TraefikService` objects,
referencing services in the [`IngressRoute`](#kind-ingressroute) objects, or recursively in others `TraefikService` objects.

!!! info "Disambiguate Traefik and Kubernetes Services "

    As the field `name` can reference different types of objects, use the field `kind` to avoid any ambiguity.
    
    The field `kind` allows the following values:
    
    * `Service` (default value): to reference a [Kubernetes Service](https://kubernetes.io/docs/concepts/services-networking/service/)
    * `TraefikService`: to reference another [Traefik Service](../services/index.md)

`TraefikService` object allows to use any (valid) combinations of:

* servers [load balancing](#server-load-balancing).  
* services [Weighted Round Robin](#weighted-round-robin) load balancing.
* services [mirroring](#mirroring).

#### Server Load Balancing

More information in the dedicated server [load balancing](../services/index.md#load-balancing) section.

??? "Declaring and Using Server Load Balancing"

    ```yaml tab="IngressRoute"
    apiVersion: traefik.containo.us/v1alpha1
    kind: IngressRoute
    metadata:
      name: ingressroutebar
      namespace: default
    
    spec:
      entryPoints:
        - web
      routes:
      - match: Host(`example.com`) && PathPrefix(`/foo`)
        kind: Rule
        services:
        - name: svc1
          namespace: default
        - name: svc2
          namespace: default
    ```
    
    ```yaml tab="K8s Service"
    apiVersion: v1
    kind: Service
    metadata:
      name: svc1
      namespace: default
    
    spec:
      ports:
        - name: http
          port: 80
      selector:
        app: traefiklabs
        task: app1
    ---
    apiVersion: v1
    kind: Service
    metadata:
      name: svc2
      namespace: default
    
    spec:
      ports:
        - name: http
          port: 80
      selector:
        app: traefiklabs
        task: app2
    ```

#### Weighted Round Robin

More information in the dedicated [Weighted Round Robin](../services/index.md#weighted-round-robin-service) service load balancing section.

??? "Declaring and Using Weighted Round Robin"

    ```yaml tab="IngressRoute"
    apiVersion: traefik.containo.us/v1alpha1
    kind: IngressRoute
    metadata:
      name: ingressroutebar
      namespace: default
    
    spec:
      entryPoints:
        - web
      routes:
      - match: Host(`example.com`) && PathPrefix(`/foo`)
        kind: Rule
        services:
        - name: wrr1
          namespace: default
          kind: TraefikService
    ```
    
    ```yaml tab="Weighted Round Robin"
    apiVersion: traefik.containo.us/v1alpha1
    kind: TraefikService
    metadata:
      name: wrr1
      namespace: default
    
    spec:
      weighted:
        services:
          - name: svc1
            port: 80
            weight: 1
          - name: wrr2
            kind: TraefikService
            weight: 1
          - name: mirror1
            kind: TraefikService
            weight: 1

    ---
    apiVersion: traefik.containo.us/v1alpha1
    kind: TraefikService
    metadata:
      name: wrr2
      namespace: default
    
    spec:
      weighted:
        services:
          - name: svc2
            port: 80
            weight: 1
          - name: svc3
            port: 80
            weight: 1
    ```
    
    ```yaml tab="K8s Service"
    apiVersion: v1
    kind: Service
    metadata:
      name: svc1
      namespace: default
    
    spec:
      ports:
        - name: http
          port: 80
      selector:
        app: traefiklabs
        task: app1
    ---
    apiVersion: v1
    kind: Service
    metadata:
      name: svc2
      namespace: default
    
    spec:
      ports:
        - name: http
          port: 80
      selector:
        app: traefiklabs
        task: app2
    ---
    apiVersion: v1
    kind: Service
    metadata:
      name: svc3
      namespace: default
    
    spec:
      ports:
        - name: http
          port: 80
      selector:
        app: traefiklabs
        task: app3
    ```

#### Mirroring

More information in the dedicated [mirroring](../services/index.md#mirroring-service) service section.

??? "Declaring and Using Mirroring"

    ```yaml tab="IngressRoute"
    apiVersion: traefik.containo.us/v1alpha1
    kind: IngressRoute
    metadata:
      name: ingressroutebar
      namespace: default
    
    spec:
      entryPoints:
        - web
      routes:
      - match: Host(`example.com`) && PathPrefix(`/foo`)
        kind: Rule
        services:
        - name: mirror1
          namespace: default
          kind: TraefikService
    ```
    
    ```yaml tab="Mirroring k8s Service"
    # Mirroring from a k8s Service
    apiVersion: traefik.containo.us/v1alpha1
    kind: TraefikService
    metadata:
      name: mirror1
      namespace: default
    
    spec:
      mirroring:
        name: svc1
        port: 80
        mirrors:
          - name: svc2
            port: 80
            percent: 20
          - name: svc3
            kind: TraefikService
            percent: 20
    ```
    
    ```yaml tab="Mirroring Traefik Service"
    # Mirroring from a Traefik Service
    apiVersion: traefik.containo.us/v1alpha1
    kind: TraefikService
    metadata:
      name: mirror1
      namespace: default
    
    spec:
      mirroring:
        name: wrr1
        kind: TraefikService
         mirrors:
           - name: svc2
             port: 80
             percent: 20
           - name: svc3
             kind: TraefikService
             percent: 20
    ```

    ```yaml tab="K8s Service"
    apiVersion: v1
    kind: Service
    metadata:
      name: svc1
      namespace: default
    
    spec:
      ports:
        - name: http
          port: 80
      selector:
        app: traefiklabs
        task: app1
    ---
    apiVersion: v1
    kind: Service
    metadata:
      name: svc2
      namespace: default
    
    spec:
      ports:
        - name: http
          port: 80
      selector:
        app: traefiklabs
        task: app2
    ```

!!! important "References and namespaces"

    If the optional `namespace` attribute is not set, the configuration will be applied with the namespace of the current resource.
    
    Additionally, when the definition of the `TraefikService` is from another provider,
    the cross-provider syntax (`service@provider`) should be used to refer to the `TraefikService`, just as in the middleware case.
    
    Specifying a namespace attribute in this case would not make any sense, and will be ignored (except if the provider is `kubernetescrd`).

#### Stickiness and load-balancing

As explained in the section about [Sticky sessions](../../services/#sticky-sessions), for stickiness to work all the way,
it must be specified at each load-balancing level.

For instance, in the example below, there is a first level of load-balancing because there is a (Weighted Round Robin) load-balancing of the two `whoami` services,
and there is a second level because each whoami service is a `replicaset` and is thus handled as a load-balancer of servers.

??? "Stickiness on two load-balancing levels"

    ```yaml tab="IngressRoute"
    apiVersion: traefik.containo.us/v1alpha1
    kind: IngressRoute
    metadata:
      name: ingressroutebar
      namespace: default

    spec:
      entryPoints:
        - web
      routes:
      - match: Host(`example.com`) && PathPrefix(`/foo`)
        kind: Rule
        services:
        - name: wrr1
          namespace: default
          kind: TraefikService
    ```

    ```yaml tab="Weighted Round Robin"
    apiVersion: traefik.containo.us/v1alpha1
    kind: TraefikService
    metadata:
      name: wrr1
      namespace: default

    spec:
      weighted:
        services:
          - name: whoami1
            kind: Service
            port: 80
            weight: 1
            sticky:
              cookie:
                name: lvl2
          - name: whoami2
            kind: Service
            weight: 1
            port: 80
            sticky:
              cookie:
                name: lvl2
        sticky:
          cookie:
            name: lvl1
    ```

    ```yaml tab="K8s Service"
    apiVersion: v1
    kind: Service
    metadata:
      name: whoami1

    spec:
      ports:
        - protocol: TCP
          name: web
          port: 80
      selector:
        app: whoami1

    ---
    apiVersion: v1
    kind: Service
    metadata:
      name: whoami2

    spec:
      ports:
        - protocol: TCP
          name: web
          port: 80
      selector:
        app: whoami2
    ```

    ```yaml tab="Deployment (to illustrate replicas)"
    kind: Deployment
    apiVersion: apps/v1
    metadata:
      namespace: default
      name: whoami1
      labels:
        app: whoami1

    spec:
      replicas: 2
      selector:
        matchLabels:
          app: whoami1
      template:
        metadata:
          labels:
            app: whoami1
        spec:
          containers:
            - name: whoami1
              image: traefik/whoami
              ports:
                - name: web
                  containerPort: 80

    ---
    kind: Deployment
    apiVersion: apps/v1
    metadata:
      namespace: default
      name: whoami2
      labels:
        app: whoami2

    spec:
      replicas: 2
      selector:
        matchLabels:
          app: whoami2
      template:
        metadata:
          labels:
            app: whoami2
        spec:
          containers:
            - name: whoami2
              image: traefik/whoami
              ports:
                - name: web
                  containerPort: 80
    ```

    To keep a session open with the same server, the client would then need to specify the two levels within the cookie for each request, e.g. with curl:

    ```bash
    curl -H Host:example.com -b "lvl1=default-whoami1-80; lvl2=http://10.42.0.6:80" http://localhost:8000/foo
    ```

    assuming `10.42.0.6` is the IP address of one of the replicas (a pod then) of the `whoami1` service.

### Kind `IngressRouteTCP`

`IngressRouteTCP` is the CRD implementation of a [Traefik TCP router](../routers/index.md#configuring-tcp-routers).

Register the `IngressRouteTCP` [kind](../../reference/dynamic-configuration/kubernetes-crd.md#definitions) in the Kubernetes cluster before creating `IngressRouteTCP` objects.

!!! info "IngressRouteTCP Attributes"

    ```yaml
    apiVersion: traefik.containo.us/v1alpha1
    kind: IngressRouteTCP
    metadata:
      name: ingressroutetcpfoo
    
    spec:
      entryPoints:                  # [1]
        - footcp
      routes:                       # [2]
      - match: HostSNI(`*`)         # [3]
        services:                   # [4]
        - name: foo                 # [5]
          port: 8080                # [6]
          weight: 10                # [7]
          terminationDelay: 400     # [8]
          proxyProtocol:            # [9]
            version: 1              # [10]
      tls:                          # [11]
        secretName: supersecret     # [12]
        options:                    # [13]
          name: opt                 # [14]
          namespace: default        # [15]
        certResolver: foo           # [16]
        domains:                    # [17]
        - main: example.net         # [18]
          sans:                     # [19]
          - a.example.net
          - b.example.net
        passthrough: false          # [20]
    ```

| Ref  | Attribute                      | Purpose                                                                                                                                                                                                                                                                                                                                                                              |
|------|--------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| [1]  | `entryPoints`                  | List of [entrypoints](../routers/index.md#entrypoints_1) names                                                                                                                                                                                                                                                                                                                       |
| [2]  | `routes`                       | List of routes                                                                                                                                                                                                                                                                                                                                                                       |
| [3]  | `routes[n].match`              | Defines the [rule](../routers/index.md#rule_1) corresponding to an underlying router                                                                                                                                                                                                                                                                                                 |
| [4]  | `routes[n].services`           | List of [Kubernetes service](https://kubernetes.io/docs/concepts/services-networking/service/) definitions  (See below for `ExternalName Service` setup)                                                                                                                                                                                                                             |
| [5]  | `services[n].name`             | Defines the name of a [Kubernetes service](https://kubernetes.io/docs/concepts/services-networking/service/)                                                                                                                                                                                                                                                                         |
| [6]  | `services[n].port`             | Defines the port of a [Kubernetes service](https://kubernetes.io/docs/concepts/services-networking/service/). This can be a reference to a named port.                                                                                                                                                                                                                               |
| [7]  | `services[n].weight`           | Defines the weight to apply to the server load balancing                                                                                                                                                                                                                                                                                                                             |
| [8]  | `services[n].terminationDelay` | corresponds to the deadline that the proxy sets, after one of its connected peers indicates it has closed the writing capability of its connection, to close the reading capability as well, hence fully terminating the connection. It is a duration in milliseconds, defaulting to 100. A negative value means an infinite deadline (i.e. the reading capability is never closed). |
| [9]  | `proxyProtocol`                | Defines the [PROXY protocol](../services/index.md#proxy-protocol) configuration                                                                                                                                                                                                                                                                                                      |
| [10] | `version`                      | Defines the [PROXY protocol](../services/index.md#proxy-protocol) version                                                                                                                                                                                                                                                                                                            |
| [11] | `tls`                          | Defines [TLS](../routers/index.md#tls_1) certificate configuration                                                                                                                                                                                                                                                                                                                   |
| [12] | `tls.secretName`               | Defines the [secret](https://kubernetes.io/docs/concepts/configuration/secret/) name used to store the certificate (in the `IngressRoute` namespace)                                                                                                                                                                                                                                 |
| [13] | `tls.options`                  | Defines the reference to a [TLSOption](#kind-tlsoption)                                                                                                                                                                                                                                                                                                                              |
| [14] | `options.name`                 | Defines the [TLSOption](#kind-tlsoption) name                                                                                                                                                                                                                                                                                                                                        |
| [15] | `options.namespace`            | Defines the [TLSOption](#kind-tlsoption) namespace                                                                                                                                                                                                                                                                                                                                   |
| [16] | `tls.certResolver`             | Defines the reference to a [CertResolver](../routers/index.md#certresolver_1)                                                                                                                                                                                                                                                                                                        |
| [17] | `tls.domains`                  | List of [domains](../routers/index.md#domains_1)                                                                                                                                                                                                                                                                                                                                     |
| [18] | `domains[n].main`              | Defines the main domain name                                                                                                                                                                                                                                                                                                                                                         |
| [19] | `domains[n].sans`              | List of SANs (alternative domains)                                                                                                                                                                                                                                                                                                                                                   |
| [20] | `tls.passthrough`              | If `true`, delegates the TLS termination to the backend                                                                                                                                                                                                                                                                                                                              |

??? example "Declaring an IngressRouteTCP"

    ```yaml tab="IngressRouteTCP"
    apiVersion: traefik.containo.us/v1alpha1
    kind: IngressRouteTCP
    metadata:
      name: ingressroutetcpfoo
    
    spec:
      entryPoints:
        - footcp
      routes:
      # Match is the rule corresponding to an underlying router.
      - match: HostSNI(`*`)
        services:
        - name: foo
          port: 8080
          terminationDelay: 400
          weight: 10
        - name: bar
          port: 8081
          terminationDelay: 500
          weight: 10
      tls:
        certResolver: foo
        domains:
        - main: example.net
          sans:
          - a.example.net
          - b.example.net
        options:
          name: opt
          namespace: default
        secretName: supersecret
        passthrough: false
    ```
    
    ```yaml tab="TLSOption"
    apiVersion: traefik.containo.us/v1alpha1
    kind: TLSOption
    metadata:
      name: opt
      namespace: default
    
    spec:
      minVersion: VersionTLS12
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

!!! important "Using Kubernetes ExternalName Service"

    Traefik backends creation needs a port to be set, however Kubernetes [ExternalName Service](https://kubernetes.io/fr/docs/concepts/services-networking/service/#externalname) could be defined without any port.
    Accordingly, Traefik supports defining a port in two ways:
    
    - only on `IngressRouteTCP` service
    - on both sides, you'll be warned if the ports don't match, and the `IngressRouteTCP` service port is used
    
    Thus, in case of two sides port definition, Traefik expects a match between ports.
    
    ??? example "Examples"
        
        ```yaml tab="IngressRouteTCP"
        ---
        apiVersion: traefik.containo.us/v1alpha1
        kind: IngressRouteTCP
        metadata:
          name: test.route
          namespace: default
        
        spec:
          entryPoints:
            - foo
        
          routes:
          - match: HostSNI(`*`)
            kind: Rule
            services:
            - name: external-svc
              port: 80
        
        ---
        apiVersion: v1
        kind: Service
        metadata:
          name: external-svc
          namespace: default
        spec:
          externalName: external.domain
          type: ExternalName
        ```
        
        ```yaml tab="ExternalName Service"
        ---
        apiVersion: traefik.containo.us/v1alpha1
        kind: IngressRouteTCP
        metadata:
          name: test.route
          namespace: default
        
        spec:
          entryPoints:
            - foo
        
          routes:
          - match: HostSNI(`*`)
            kind: Rule
            services:
            - name: external-svc
        
        ---
        apiVersion: v1
        kind: Service
        metadata:
          name: external-svc
          namespace: default
        spec:
          externalName: external.domain
          type: ExternalName
          ports:
            - port: 80
        ```
        
        ```yaml tab="Both sides"
        ---
        apiVersion: traefik.containo.us/v1alpha1
        kind: IngressRouteTCP
        metadata:
          name: test.route
          namespace: default
        
        spec:
          entryPoints:
            - foo
        
          routes:
          - match: HostSNI(`*`)
            kind: Rule
            services:
            - name: external-svc
              port: 80
        
        ---
        apiVersion: v1
        kind: Service
        metadata:
          name: external-svc
          namespace: default
        spec:
          externalName: external.domain
          type: ExternalName
          ports:
            - port: 80
        ```

### Kind `IngressRouteUDP`

`IngressRouteUDP` is the CRD implementation of a [Traefik UDP router](../routers/index.md#configuring-udp-routers).

Register the `IngressRouteUDP` [kind](../../reference/dynamic-configuration/kubernetes-crd.md#definitions) in the Kubernetes cluster before creating `IngressRouteUDP` objects.

!!! info "IngressRouteUDP Attributes"

    ```yaml
    apiVersion: traefik.containo.us/v1alpha1
    kind: IngressRouteUDP
    metadata:
      name: ingressrouteudpfoo
    
    spec:
      entryPoints:                  # [1]
        - fooudp
      routes:                       # [2]
      - services:                   # [3]
        - name: foo                 # [4]
          port: 8080                # [5]
          weight: 10                # [6]
    ```

| Ref  | Attribute                      | Purpose                                                                                                                                                                                                                                                                                                                                                                                  |
|------|--------------------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| [1]  | `entryPoints`                  | List of [entrypoints](../routers/index.md#entrypoints_1) names                                                                                                                                                                                                                                                                                                                           |
| [2]  | `routes`                       | List of routes                                                                                                                                                                                                                                                                                                                                                                           |
| [3]  | `routes[n].services`           | List of [Kubernetes service](https://kubernetes.io/docs/concepts/services-networking/service/) definitions (See below for `ExternalName Service` setup)                                                                                                                                                                                                                                  |
| [4]  | `services[n].name`             | Defines the name of a [Kubernetes service](https://kubernetes.io/docs/concepts/services-networking/service/)                                                                                                                                                                                                                                                                             |
| [6]  | `services[n].port`             | Defines the port of a [Kubernetes service](https://kubernetes.io/docs/concepts/services-networking/service/). This can be a reference to a named port.                                                                                                                                                                                                                                   |
| [7]  | `services[n].weight`           | Defines the weight to apply to the server load balancing                                                                                                                                                                                                                                                                                                                                 |

??? example "Declaring an IngressRouteUDP"

    ```yaml
    apiVersion: traefik.containo.us/v1alpha1
    kind: IngressRouteUDP
    metadata:
      name: ingressrouteudpfoo
    
    spec:
      entryPoints:
        - fooudp
      routes:
      - services:
        - name: foo
          port: 8080
          weight: 10
        - name: bar
          port: 8081
          weight: 10
    ```

!!! important "Using Kubernetes ExternalName Service"

    Traefik backends creation needs a port to be set, however Kubernetes [ExternalName Service](https://kubernetes.io/fr/docs/concepts/services-networking/service/#externalname) could be defined without any port.
    Accordingly, Traefik supports defining a port in two ways:
    
    - only on `IngressRouteUDP` service
    - on both sides, you'll be warned if the ports don't match, and the `IngressRouteUDP` service port is used
    
    Thus, in case of two sides port definition, Traefik expects a match between ports.
    
    ??? example "Examples"
        
        ```yaml tab="IngressRouteUDP"
        ---
        apiVersion: traefik.containo.us/v1alpha1
        kind: IngressRouteUDP
        metadata:
          name: test.route
          namespace: default
        
        spec:
          entryPoints:
            - foo
        
          routes:
          - services:
            - name: external-svc
              port: 80
        
        ---
        apiVersion: v1
        kind: Service
        metadata:
          name: external-svc
          namespace: default
        spec:
          externalName: external.domain
          type: ExternalName
        ```
        
        ```yaml tab="ExternalName Service"
        ---
        apiVersion: traefik.containo.us/v1alpha1
        kind: IngressRouteUDP
        metadata:
          name: test.route
          namespace: default
        
        spec:
          entryPoints:
            - foo
        
          routes:
          - services:
            - name: external-svc
        
        ---
        apiVersion: v1
        kind: Service
        metadata:
          name: external-svc
          namespace: default
        spec:
          externalName: external.domain
          type: ExternalName
          ports:
            - port: 80
        ```
        
        ```yaml tab="Both sides"
        ---
        apiVersion: traefik.containo.us/v1alpha1
        kind: IngressRouteUDP
        metadata:
          name: test.route
          namespace: default
        
        spec:
          entryPoints:
            - foo
        
          routes:
          - services:
            - name: external-svc
              port: 80
        
        ---
        apiVersion: v1
        kind: Service
        metadata:
          name: external-svc
          namespace: default
        spec:
          externalName: external.domain
          type: ExternalName
          ports:
            - port: 80
        ```

### Kind: `TLSOption`

`TLSOption` is the CRD implementation of a [Traefik "TLS Option"](../../https/tls.md#tls-options).

Register the `TLSOption` [kind](../../reference/dynamic-configuration/kubernetes-crd.md#definitions) in the Kubernetes cluster before creating `TLSOption` objects
or referencing TLS options in the [`IngressRoute`](#kind-ingressroute) / [`IngressRouteTCP`](#kind-ingressroutetcp) objects.

!!! info "TLSOption Attributes"
   
    ```yaml tab="TLSOption"
    apiVersion: traefik.containo.us/v1alpha1
    kind: TLSOption
    metadata:
      name: mytlsoption
      namespace: default
    
    spec:
      minVersion: VersionTLS12                      # [1]
      maxVersion: VersionTLS13                      # [1]
      curvePreferences:                             # [3]
        - CurveP521
        - CurveP384
      cipherSuites:                                 # [4]
        - TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256
        - TLS_RSA_WITH_AES_256_GCM_SHA384
      clientAuth:                                   # [5]
        secretNames:                                # [6]
          - secretCA1
          - secretCA2
        clientAuthType: VerifyClientCertIfGiven     # [7]
      sniStrict: true                               # [8]
    ```

| Ref | Attribute                   | Purpose                                                                                                                                                                    |
|-----|-----------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| [1] | `minVersion`                | Defines the [minimum TLS version](../../https/tls.md#minimum-tls-version) that is acceptable                                                                               |
| [2] | `maxVersion`                | Defines the [maximum TLS version](../../https/tls.md#maximum-tls-version) that is acceptable                                                                               |
| [3] | `cipherSuites`              | list of supported [cipher suites](../../https/tls.md#cipher-suites) for TLS versions up to TLS 1.2                                                                         |
| [4] | `curvePreferences`          | List of the [elliptic curves references](../../https/tls.md#curve-preferences) that will be used in an ECDHE handshake, in preference order                                |
| [5] | `clientAuth`                | determines the server's policy for TLS [Client Authentication](../../https/tls.md#client-authentication-mtls)                                                              |
| [6] | `clientAuth.secretNames`    | list of names of the referenced Kubernetes [Secrets](https://kubernetes.io/docs/concepts/configuration/secret/) (in TLSOption namespace)                                   |
| [7] | `clientAuth.clientAuthType` | defines the client authentication type to apply. The available values are: `NoClientCert`, `RequestClientCert`, `VerifyClientCertIfGiven` and `RequireAndVerifyClientCert` |
| [8] | `sniStrict`                 | if `true`, Traefik won't allow connections from clients connections that do not specify a server_name extension                                                            |

??? example "Declaring and referencing a TLSOption"
   
    ```yaml tab="TLSOption"
    apiVersion: traefik.containo.us/v1alpha1
    kind: TLSOption
    metadata:
      name: mytlsoption
      namespace: default
    
    spec:
      minVersion: VersionTLS12
      sniStrict: true
      cipherSuites:
        - TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256
        - TLS_RSA_WITH_AES_256_GCM_SHA384
      clientAuth:
        secretNames:
          - secretCA1
          - secretCA2
        clientAuthType: VerifyClientCertIfGiven
    ```
    
    ```yaml tab="IngressRoute"
    apiVersion: traefik.containo.us/v1alpha1
    kind: IngressRoute
    metadata:
      name: ingressroutebar
    
    spec:
      entryPoints:
        - web
      routes:
      - match: Host(`example.com`) && PathPrefix(`/stripit`)
        kind: Rule
        services:
        - name: whoami
          port: 80
      tls:
        options: 
          name: mytlsoption
          namespace: default
    ```

    ```yaml tab="Secrets"
    apiVersion: v1
    kind: Secret
    metadata:
      name: secretCA1
      namespace: default
    
    data:
      tls.ca: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCi0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0=
    
    ---
    apiVersion: v1
    kind: Secret
    metadata:
      name: secretCA2
      namespace: default
    
    data:
      tls.ca: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCi0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0=
    ```
        
!!! important "References and namespaces"

    If the optional `namespace` attribute is not set, the configuration will be applied with the namespace of the IngressRoute.

	Additionally, when the definition of the TLS option is from another provider,
	the cross-provider [syntax](../../providers/overview.md#provider-namespace) (`middlewarename@provider`) should be used to refer to the TLS option.
	Specifying a namespace attribute in this case would not make any sense, and will be ignored.

### Kind: `TLSStore`

`TLSStore` is the CRD implementation of a [Traefik "TLS Store"](../../https/tls.md#certificates-stores).

Register the `TLSStore` kind in the Kubernetes cluster before creating `TLSStore` objects
or referencing TLS stores in the [`IngressRoute`](#kind-ingressroute) / [`IngressRouteTCP`](#kind-ingressroutetcp) objects.

!!! important "Default TLS Store"

    Traefik currently only uses the [TLS Store named "default"](../../https/tls.md#certificates-stores).
    This means that if you have two stores that are named default in different kubernetes namespaces,
    they may be randomly chosen.
    For the time being, please only configure one TLSSTore named default.

!!! info "TLSStore Attributes"
   
    ```yaml tab="TLSStore"
    apiVersion: traefik.containo.us/v1alpha1
    kind: TLSStore
    metadata:
      name: default
      namespace: default
    
    spec:
      defaultCertificate:
        secretName: mySecret                      # [1]
    ```

| Ref | Attribute    | Purpose                                                                                                                                                     |
|-----|--------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------|
| [1] | `secretName` | The name of the referenced Kubernetes [Secret](https://kubernetes.io/docs/concepts/configuration/secret/) that holds the default certificate for the store. |

??? example "Declaring and referencing a TLSStore"
   
    ```yaml tab="TLSStore"
    apiVersion: traefik.containo.us/v1alpha1
    kind: TLSStore
    metadata:
      name: default
      namespace: default
    
    spec:
      defaultCertificate:
        secretName:  supersecret
    ```
    
    ```yaml tab="IngressRoute"
    apiVersion: traefik.containo.us/v1alpha1
    kind: IngressRoute
    metadata:
      name: ingressroutebar
    
    spec:
      entryPoints:
        - web
      routes:
      - match: Host(`example.com`) && PathPrefix(`/stripit`)
        kind: Rule
        services:
        - name: whoami
          port: 80
      tls:
        store: 
          name: default
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

### Kind: `ServersTransport`

`ServersTransport` is the CRD implementation of a [ServersTransport](../services/index.md#serverstransport).

!!! important "Default serversTransport"
    If no `serversTransport` is specified, the `default@internal` will be used. 
    The `default@internal` serversTransport is created from the [static configuration](../overview.md#transport-configuration). 

!!! info "ServersTransport Attributes"
   
    ```yaml tab="TLSStore"
    apiVersion: traefik.containo.us/v1alpha1
    kind: ServersTransport
    metadata:
      name: mytransport
      namespace: default
    
    spec:
      serverName: foobar               # [1]
      insecureSkipVerify: true         # [2]
      rootCAsSecrets:                  # [3]
        - foobar
        - foobar
      certificatesSecrets:             # [4]
        - foobar
        - foobar
      maxIdleConnsPerHost: 1           # [5]
      forwardingTimeouts:              # [6]
        dialTimeout: 42s               # [7]
        responseHeaderTimeout: 42s     # [8]
        idleConnTimeout: 42s           # [9]
    ```

| Ref | Attribute               | Purpose                                                                                                                                              |
|-----|-------------------------|------------------------------------------------------------------------------------------------------------------------------------------------------|
| [1] | `serverName`            | ServerName used to contact the server.                                                                                                               |
| [2] | `insecureSkipVerify`    | Disable SSL certificate verification.                                                                                                                |
| [3] | `rootCAsSecrets`        | Add cert file for self-signed certificate.                                                                                                           |
| [4] | `certificatesSecrets`   | Certificates for mTLS.                                                                                                                               |
| [5] | `maxIdleConnsPerHost`   | If non-zero, controls the maximum idle (keep-alive) to keep per-host. If zero, `defaultMaxIdleConnsPerHost` is used.                                 |
| [6] | `forwardingTimeouts`    | Timeouts for requests forwarded to the backend servers.                                                                                              |
| [7] | `dialTimeout`           | The amount of time to wait until a connection to a backend server can be established. If zero, no timeout exists.                                    |
| [8] | `responseHeaderTimeout` | The amount of time to wait for a server's response headers after fully writing the request (including its body, if any). If zero, no timeout exists. |
| [9] | `idleConnTimeout`       | The maximum period for which an idle HTTP keep-alive connection will remain open before closing itself.                                              |

??? example "Declaring and referencing a ServersTransport"
   
    ```yaml tab="ServersTransport"
    apiVersion: traefik.containo.us/v1alpha1
    kind: ServersTransport
    metadata:
      name: mytransport
      namespace: default
    
    spec:
      serverName: example.org
      insecureSkipVerify: true
    ```
    
    ```yaml tab="IngressRoute"
    apiVersion: traefik.containo.us/v1alpha1
    kind: IngressRoute
    metadata:
      name: testroute
      namespace: default
    
    spec:
      entryPoints:
        - web
      routes:
      - match: Host(`example.com`)
        kind: Rule
        services:
        - name: whoami
          port: 80
          serversTransport: mytransport
    ```

## Further

Also see the [full example](../../user-guides/crd-acme/index.md) with Let's Encrypt.
