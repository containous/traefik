# Traefik & Docker

A Story of Labels & Containers
{: .subtitle }

![Docker](../../assets/img/providers/docker.png)

Attach labels to your containers and let Traefik do the rest!

## Configuration Examples

??? example "Configuring Docker & Deploying / Exposing Services"

    Enabling the docker provider

    ```toml tab="File (TOML)"
    [providers.docker]
    ```

    ```yaml tab="File (YAML)"
    providers:
      docker: {}
    ```

    ```bash tab="CLI"
    --providers.docker=true
    ```

    Attaching labels to containers (in your docker compose file)

    ```yaml
    version: "3"
    services:
      my-container:
        # ...
        labels:
          - traefik.http.routers.my-container.rule=Host(`example.com`)
    ```

??? example "Specify a Custom Port for the Container"

    Forward requests for `http://example.com` to `http://<private IP of container>:12345`:

    ```yaml
    version: "3"
    services:
      my-container:
        # ...
        labels:
          - traefik.http.routers.my-container.rule=Host(`example.com`)
          # Tell Traefik to use the port 12345 to connect to `my-container`
          - traefik.http.services.my-service.loadbalancer.server.port=12345
    ```

    !!! important "Traefik Connecting to the Wrong Port: `HTTP/502 Gateway Error`"
        By default, Traefik uses the first exposed port of a container.

        Setting the label `traefik.http.services.xxx.loadbalancer.server.port`
        overrides that behavior.

??? example "Specifying more than one router and service per container"

    Forwarding requests to more than one port on a container requires referencing the service loadbalancer port definition using the service parameter on the router.

    In this example, requests are forwarded for `http://example-a.com` to `http://<private IP of container>:8000` in addition to `http://example-b.com` forwarding to `http://<private IP of container>:9000`:

    ```yaml
    version: "3"
    services:
      my-container:
        # ...
        labels:
          - traefik.http.routers.www-router.rule=Host(`example-a.com`)
          - traefik.http.routers.www-router.service=www-service
          - traefik.http.services.www-service.loadbalancer.server.port=8000
          - traefik.http.routers.admin-router.rule=Host(`example-b.com`)
          - traefik.http.routers.admin-router.service=admin-service
          - traefik.http.services.admin-service.loadbalancer.server.port=9000
    ```

??? example "Configuring Docker Swarm & Deploying / Exposing Services"

    Enabling the docker provider (Swarm Mode)

    ```toml tab="File (TOML)"
    [providers.docker]
      # swarm classic (1.12-)
      # endpoint = "tcp://127.0.0.1:2375"
      # docker swarm mode (1.12+)
      endpoint = "tcp://127.0.0.1:2377"
      swarmMode = true
    ```

    ```yaml tab="File (YAML)"
    providers:
      docker:
        # swarm classic (1.12-)
        # endpoint: "tcp://127.0.0.1:2375"
        # docker swarm mode (1.12+)
        endpoint: "tcp://127.0.0.1:2377"
        swarmMode: true
    ```

    ```bash tab="CLI"
    # swarm classic (1.12-)
    # --providers.docker.endpoint=tcp://127.0.0.1:2375
    # docker swarm mode (1.12+)
    --providers.docker.endpoint=tcp://127.0.0.1:2377
    --providers.docker.swarmMode=true
    ```

    Attach labels to services (not to containers) while in Swarm mode (in your docker compose file)

    ```yaml
    version: "3"
    services:
      my-container:
        deploy:
          labels:
            - traefik.http.routers.my-container.rule=Host(`example.com`)
            - traefik.http.services.my-container-service.loadbalancer.server.port=8080
    ```

    !!! important "Labels in Docker Swarm Mode"
        While in Swarm Mode, Traefik uses labels found on services, not on individual containers.
        Therefore, if you use a compose file with Swarm Mode, labels should be defined in the `deploy` part of your service.
        This behavior is only enabled for docker-compose version 3+ ([Compose file reference](https://docs.docker.com/compose/compose-file/compose-file-v3/#labels-1)).

## Routing Configuration

!!! info "Labels"

    - Labels are case insensitive.
    - The complete list of labels can be found in [the reference page](../../reference/dynamic-configuration/docker.md).

### General

Traefik creates, for each container, a corresponding [service](../services/index.md) and [router](../routers/index.md).

The Service automatically gets a server per instance of the container,
and the router automatically gets a rule defined by `defaultRule` (if no rule for it was defined in labels).

#### Service definition

--8<-- "content/routing/providers/service-by-label.md"

??? example "Automatic service assignment with labels"

    With labels in a compose file

    ```yaml
    labels:
      - "traefik.http.routers.myproxy.rule=Host(`example.net`)"
      # service myservice gets automatically assigned to router myproxy
      - "traefik.http.services.myservice.loadbalancer.server.port=80"
    ```

??? example "Automatic service creation and assignment with labels"

    With labels in a compose file

    ```yaml
    labels:
      # no service specified or defined and yet one gets automatically created
      # and assigned to router myproxy.
      - "traefik.http.routers.myproxy.rule=Host(`example.net`)"
    ```

### Routers

To update the configuration of the Router automatically attached to the container,
add labels starting with `traefik.http.routers.<name-of-your-choice>.` and followed by the option you want to change.

For example, to change the rule, you could add the label ```traefik.http.routers.my-container.rule=Host(`example.com`)```.

!!! warning "The character `@` is not authorized in the router name `<router_name>`."

??? info "`traefik.http.routers.<router_name>.rule`"

    See [rule](../routers/index.md#rule) for more information.

    ```yaml
    - "traefik.http.routers.myrouter.rule=Host(`example.com`)"
    ```

??? info "`traefik.http.routers.<router_name>.entrypoints`"

    See [entry points](../routers/index.md#entrypoints) for more information.

    ```yaml
    - "traefik.http.routers.myrouter.entrypoints=ep1,ep2"
    ```

??? info "`traefik.http.routers.<router_name>.middlewares`"

    See [middlewares](../routers/index.md#middlewares) and [middlewares overview](../../middlewares/overview.md) for more information.

    ```yaml
    - "traefik.http.routers.myrouter.middlewares=auth,prefix,cb"
    ```

??? info "`traefik.http.routers.<router_name>.service`"

    See [service](../routers/index.md#service) for more information.

    ```yaml
    - "traefik.http.routers.myrouter.service=myservice"
    ```

??? info "`traefik.http.routers.<router_name>.tls`"

    See [tls](../routers/index.md#tls) for more information.

    ```yaml
    - "traefik.http.routers.myrouter.tls=true"
    ```

??? info "`traefik.http.routers.<router_name>.tls.certresolver`"

    See [certResolver](../routers/index.md#certresolver) for more information.

    ```yaml
    - "traefik.http.routers.myrouter.tls.certresolver=myresolver"
    ```

??? info "`traefik.http.routers.<router_name>.tls.domains[n].main`"

    See [domains](../routers/index.md#domains) for more information.

    ```yaml
    - "traefik.http.routers.myrouter.tls.domains[0].main=example.org"
    ```

??? info "`traefik.http.routers.<router_name>.tls.domains[n].sans`"

    See [domains](../routers/index.md#domains) for more information.

    ```yaml
    - "traefik.http.routers.myrouter.tls.domains[0].sans=test.example.org,dev.example.org"
    ```

??? info "`traefik.http.routers.<router_name>.tls.options`"

    See [options](../routers/index.md#options) for more information.

    ```yaml
    - "traefik.http.routers.myrouter.tls.options=foobar"
    ```

??? info "`traefik.http.routers.<router_name>.priority`"

    See [priority](../routers/index.md#priority) for more information.

    ```yaml
    - "traefik.http.routers.myrouter.priority=42"
    ```

### Services

To update the configuration of the Service automatically attached to the container,
add labels starting with `traefik.http.services.<name-of-your-choice>.`, followed by the option you want to change.

For example, to change the `passHostHeader` behavior,
you'd add the label `traefik.http.services.<name-of-your-choice>.loadbalancer.passhostheader=false`.

!!! warning "The character `@` is not authorized in the service name `<service_name>`."

??? info "`traefik.http.services.<service_name>.loadbalancer.server.port`"
    
    Registers a port.
    Useful when the container exposes multiples ports.

    Mandatory for Docker Swarm (see the section ["Port Detection with Docker Swarm"](../../providers/docker.md#port-detection_1)).
    {: #port }

    ```yaml
    - "traefik.http.services.myservice.loadbalancer.server.port=8080"
    ```

??? info "`traefik.http.services.<service_name>.loadbalancer.server.scheme`"

    Overrides the default scheme.

    ```yaml
    - "traefik.http.services.myservice.loadbalancer.server.scheme=http"
    ```

??? info "`traefik.http.services.<service_name>.loadbalancer.serverstransport`"

    Allows to reference a ServersTransport resource that is defined either with the File provider or the Kubernetes CRD one.
    See [serverstransport](../services/index.md#serverstransport) for more information.
    
    ```yaml
    - "traefik.http.services.<service_name>.loadbalancer.serverstransport=foobar@file"
    ```

??? info "`traefik.http.services.<service_name>.loadbalancer.passhostheader`"

    See [pass Host header](../services/index.md#pass-host-header) for more information.

    ```yaml
    - "traefik.http.services.myservice.loadbalancer.passhostheader=true"
    ```

??? info "`traefik.http.services.<service_name>.loadbalancer.healthcheck.headers.<header_name>`"

    See [health check](../services/index.md#health-check) for more information.

    ```yaml
    - "traefik.http.services.myservice.loadbalancer.healthcheck.headers.X-Foo=foobar"
    ```

??? info "`traefik.http.services.<service_name>.loadbalancer.healthcheck.hostname`"

    See [health check](../services/index.md#health-check) for more information.

    ```yaml
    - "traefik.http.services.myservice.loadbalancer.healthcheck.hostname=example.org"
    ```

??? info "`traefik.http.services.<service_name>.loadbalancer.healthcheck.interval`"

    See [health check](../services/index.md#health-check) for more information.

    ```yaml
    - "traefik.http.services.myservice.loadbalancer.healthcheck.interval=10s"
    ```

??? info "`traefik.http.services.<service_name>.loadbalancer.healthcheck.path`"

    See [health check](../services/index.md#health-check) for more information.

    ```yaml
    - "traefik.http.services.myservice.loadbalancer.healthcheck.path=/foo"
    ```

??? info "`traefik.http.services.<service_name>.loadbalancer.healthcheck.port`"

    See [health check](../services/index.md#health-check) for more information.

    ```yaml
    - "traefik.http.services.myservice.loadbalancer.healthcheck.port=42"
    ```

??? info "`traefik.http.services.<service_name>.loadbalancer.healthcheck.scheme`"

    See [health check](../services/index.md#health-check) for more information.

    ```yaml
    - "traefik.http.services.myservice.loadbalancer.healthcheck.scheme=http"
    ```

??? info "`traefik.http.services.<service_name>.loadbalancer.healthcheck.timeout`"

    See [health check](../services/index.md#health-check) for more information.

    ```yaml
    - "traefik.http.services.myservice.loadbalancer.healthcheck.timeout=10"
    ```

??? info "`traefik.http.services.<service_name>.loadbalancer.healthcheck.followredirects`"

    See [health check](../services/index.md#health-check) for more information.

    ```yaml
    - "traefik.http.services.myservice.loadbalancer.healthcheck.followredirects=true"
    ```

??? info "`traefik.http.services.<service_name>.loadbalancer.sticky.cookie`"

    See [sticky sessions](../services/index.md#sticky-sessions) for more information.

    ```yaml
    - "traefik.http.services.myservice.loadbalancer.sticky.cookie=true"
    ```

??? info "`traefik.http.services.<service_name>.loadbalancer.sticky.cookie.httponly`"

    See [sticky sessions](../services/index.md#sticky-sessions) for more information.

    ```yaml
    - "traefik.http.services.myservice.loadbalancer.sticky.cookie.httponly=true"
    ```

??? info "`traefik.http.services.<service_name>.loadbalancer.sticky.cookie.name`"

    See [sticky sessions](../services/index.md#sticky-sessions) for more information.

    ```yaml
    - "traefik.http.services.myservice.loadbalancer.sticky.cookie.name=foobar"
    ```

??? info "`traefik.http.services.<service_name>.loadbalancer.sticky.cookie.secure`"

    See [sticky sessions](../services/index.md#sticky-sessions) for more information.

    ```yaml
    - "traefik.http.services.myservice.loadbalancer.sticky.cookie.secure=true"
    ```

??? info "`traefik.http.services.<service_name>.loadbalancer.sticky.cookie.samesite`"
    
    See [sticky sessions](../services/index.md#sticky-sessions) for more information.
    
    ```yaml
    - "traefik.http.services.myservice.loadbalancer.sticky.cookie.samesite=none"
    ```

??? info "`traefik.http.services.<service_name>.loadbalancer.responseforwarding.flushinterval`"

    See [response forwarding](../services/index.md#response-forwarding) for more information.

    ```yaml
    - "traefik.http.services.myservice.loadbalancer.responseforwarding.flushinterval=10"
    ```

### Middleware

You can declare pieces of middleware using labels starting with `traefik.http.middlewares.<name-of-your-choice>.`,
followed by the middleware type/options.

For example, to declare a middleware [`redirectscheme`](../../middlewares/redirectscheme.md) named `my-redirect`,
you'd write `traefik.http.middlewares.my-redirect.redirectscheme.scheme=https`.

More information about available middlewares in the dedicated [middlewares section](../../middlewares/overview.md).

!!! warning "The character `@` is not authorized in the middleware name."

??? example "Declaring and Referencing a Middleware"

    ```yaml
       services:
         my-container:
           # ...
           labels:
             # Declaring a middleware
             - traefik.http.middlewares.my-redirect.redirectscheme.scheme=https
             # Referencing a middleware
             - traefik.http.routers.my-container.middlewares=my-redirect
    ```

!!! warning "Conflicts in Declaration"

    If you declare multiple middleware with the same name but with different parameters, the middleware fails to be declared.

### TCP

You can declare TCP Routers and/or Services using labels.

??? example "Declaring TCP Routers and Services"

    ```yaml
       services:
         my-container:
           # ...
           labels:
             - "traefik.tcp.routers.my-router.rule=HostSNI(`example.com`)"
             - "traefik.tcp.routers.my-router.tls=true"
             - "traefik.tcp.services.my-service.loadbalancer.server.port=4123"
    ```

!!! warning "TCP and HTTP"

    If you declare a TCP Router/Service, it will prevent Traefik from automatically creating an HTTP Router/Service (like it does by default if no TCP Router/Service is defined).
    You can declare both a TCP Router/Service and an HTTP Router/Service for the same container (but you have to do so manually).

#### TCP Routers

??? info "`traefik.tcp.routers.<router_name>.entrypoints`"

    See [entry points](../routers/index.md#entrypoints_1) for more information.

    ```yaml
    - "traefik.tcp.routers.mytcprouter.entrypoints=ep1,ep2"
    ```

??? info "`traefik.tcp.routers.<router_name>.rule`"

    See [rule](../routers/index.md#rule_1) for more information.

    ```yaml
    - "traefik.tcp.routers.mytcprouter.rule=HostSNI(`example.com`)"
    ```

??? info "`traefik.tcp.routers.<router_name>.service`"

    See [service](../routers/index.md#services) for more information.

    ```yaml
    - "traefik.tcp.routers.mytcprouter.service=myservice"
    ```

??? info "`traefik.tcp.routers.<router_name>.tls`"

    See [TLS](../routers/index.md#tls_1) for more information.

    ```yaml
    - "traefik.tcp.routers.mytcprouter.tls=true"
    ```

??? info "`traefik.tcp.routers.<router_name>.tls.certresolver`"

    See [certResolver](../routers/index.md#certresolver_1) for more information.

    ```yaml
    - "traefik.tcp.routers.mytcprouter.tls.certresolver=myresolver"
    ```

??? info "`traefik.tcp.routers.<router_name>.tls.domains[n].main`"

    See [domains](../routers/index.md#domains_1) for more information.

    ```yaml
    - "traefik.tcp.routers.mytcprouter.tls.domains[0].main=example.org"
    ```

??? info "`traefik.tcp.routers.<router_name>.tls.domains[n].sans`"

    See [domains](../routers/index.md#domains_1) for more information.

    ```yaml
    - "traefik.tcp.routers.mytcprouter.tls.domains[0].sans=test.example.org,dev.example.org"
    ```

??? info "`traefik.tcp.routers.<router_name>.tls.options`"

    See [options](../routers/index.md#options_1) for more information.

    ```yaml
    - "traefik.tcp.routers.mytcprouter.tls.options=mysoptions"
    ```

??? info "`traefik.tcp.routers.<router_name>.tls.passthrough`"

    See [TLS](../routers/index.md#tls_1) for more information.

    ```yaml
    - "traefik.tcp.routers.mytcprouter.tls.passthrough=true"
    ```

#### TCP Services

??? info "`traefik.tcp.services.<service_name>.loadbalancer.server.port`"

    Registers a port of the application.

    ```yaml
    - "traefik.tcp.services.mytcpservice.loadbalancer.server.port=423"
    ```

??? info "`traefik.tcp.services.<service_name>.loadbalancer.terminationdelay`"

    See [termination delay](../services/index.md#termination-delay) for more information.

    ```yaml
    - "traefik.tcp.services.mytcpservice.loadbalancer.terminationdelay=100"
    ```

??? info "`traefik.tcp.services.<service_name>.loadbalancer.proxyprotocol.version`"

    See [PROXY protocol](../services/index.md#proxy-protocol) for more information.

    ```yaml
    - "traefik.tcp.services.mytcpservice.loadbalancer.proxyprotocol.version=1"
    ```

### UDP

You can declare UDP Routers and/or Services using labels.

??? example "Declaring UDP Routers and Services"

    ```yaml
       services:
         my-container:
           # ...
           labels:
             - "traefik.udp.routers.my-router.entrypoints=udp"
             - "traefik.udp.services.my-service.loadbalancer.server.port=4123"
    ```

!!! warning "UDP and HTTP"

    If you declare a UDP Router/Service, it will prevent Traefik from automatically creating an HTTP Router/Service (like it does by default if no UDP Router/Service is defined).
    You can declare both a UDP Router/Service and an HTTP Router/Service for the same container (but you have to do so manually).

#### UDP Routers

??? info "`traefik.udp.routers.<router_name>.entrypoints`"

    See [entry points](../routers/index.md#entrypoints_2) for more information.

    ```yaml
    - "traefik.udp.routers.myudprouter.entrypoints=ep1,ep2"
    ```

??? info "`traefik.udp.routers.<router_name>.service`"

    See [service](../routers/index.md#services_1) for more information.

    ```yaml
    - "traefik.udp.routers.myudprouter.service=myservice"
    ```

#### UDP Services

??? info "`traefik.udp.services.<service_name>.loadbalancer.server.port`"

    Registers a port of the application.

    ```yaml
    - "traefik.udp.services.myudpservice.loadbalancer.server.port=423"
    ```

### Specific Provider Options

#### `traefik.enable`

```yaml
- "traefik.enable=true"
```

You can tell Traefik to consider (or not) the container by setting `traefik.enable` to true or false.

This option overrides the value of `exposedByDefault`.

#### `traefik.docker.network`

```yaml
- "traefik.docker.network=mynetwork"
```

Overrides the default docker network to use for connections to the container.

If a container is linked to several networks, be sure to set the proper network name (you can check this with `docker inspect <container_id>`),
otherwise it will randomly pick one (depending on how docker is returning them).

!!! warning
    When deploying a stack from a compose file `stack`, the networks defined are prefixed with `stack`.

#### `traefik.docker.lbswarm`

```yaml
- "traefik.docker.lbswarm=true"
```

Enables Swarm's inbuilt load balancer (only relevant in Swarm Mode).

If you enable this option, Traefik will use the virtual IP provided by docker swarm instead of the containers IPs.
Which means that Traefik will not perform any kind of load balancing and will delegate this task to swarm.
