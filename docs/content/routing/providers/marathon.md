# Traefik & Marathon

Traefik can be configured to use Marathon as a provider.
{: .subtitle }

See also [Marathon user guide](../../user-guides/marathon.md).

## Routing Configuration

!!! info "Labels"
    
    - Labels are case insensitive.
    - The complete list of labels can be found in [the reference page](../../reference/dynamic-configuration/marathon.md).

### General

Traefik creates, for each Marathon application, a corresponding [service](../services/index.md) and [router](../routers/index.md).

The Service automatically gets a server per instance of the application,
and the router automatically gets a rule defined by defaultRule (if no rule for it was defined in labels).

#### Service definition

--8<-- "content/routing/providers/service-by-label.md"

??? example "Automatic service assignment with labels"

    Service myservice gets automatically assigned to router myproxy.

    ```json
    labels: {
      "traefik.http.routers.myproxy.rule": "Host(`example.net`)",
      "traefik.http.services.myservice.loadbalancer.server.port": "80"
    }
    ```

??? example "Automatic service creation and assignment with labels"

    No service specified or defined, and yet one gets automatically created.
    and assigned to router myproxy.

    ```json
    labels: {
      "traefik.http.routers.myproxy.rule": "Host(`example.net`)"
	}
    ```

### Routers

To update the configuration of the Router automatically attached to the application,
add labels starting with `traefik.http.routers.{router-name-of-your-choice}.` and followed by the option you want to change.

For example, to change the routing rule, you could add the label ```"traefik.http.routers.routername.rule": "Host(`example.com`)"```.

!!! warning "The character `@` is not authorized in the router name `<router_name>`."

??? info "`traefik.http.routers.<router_name>.rule`"
    
    See [rule](../routers/index.md#rule) for more information. 
    
    ```json
    "traefik.http.routers.myrouter.rule": "Host(`example.com`)"
    ```

??? info "`traefik.http.routers.<router_name>.entrypoints`"
    
    See [entry points](../routers/index.md#entrypoints) for more information. 
    
    ```json
    "traefik.http.routers.myrouter.entrypoints": "ep1,ep2"
    ```

??? info "`traefik.http.routers.<router_name>.middlewares`"
    
    See [middlewares](../routers/index.md#middlewares) and [middlewares overview](../../middlewares/overview.md) for more information. 
    
    ```json
    "traefik.http.routers.myrouter.middlewares": "auth,prefix,cb"
    ```

??? info "`traefik.http.routers.<router_name>.service`"
    
    See [rule](../routers/index.md#service) for more information. 
    
    ```json
    "traefik.http.routers.myrouter.service": "myservice"
    ```

??? info "`traefik.http.routers.<router_name>.tls`"
    
    See [tls](../routers/index.md#tls) for more information.
    
    ```json
    "traefik.http.routers.myrouter.tls": "true"
    ```

??? info "`traefik.http.routers.<router_name>.tls.certresolver`"
    
    See [certResolver](../routers/index.md#certresolver) for more information.
    
    ```json
    "traefik.http.routers.myrouter.tls.certresolver": "myresolver"
    ```

??? info "`traefik.http.routers.<router_name>.tls.domains[n].main`"
    
    See [domains](../routers/index.md#domains) for more information.
    
    ```json
    "traefik.http.routers.myrouter.tls.domains[0].main": "example.org"
    ```

??? info "`traefik.http.routers.<router_name>.tls.domains[n].sans`"
    
    See [domains](../routers/index.md#domains) for more information.
    
    ```json
    "traefik.http.routers.myrouter.tls.domains[0].sans": "test.example.org,dev.example.org"
    ```

??? info "`traefik.http.routers.<router_name>.tls.options`"
    
    See [options](../routers/index.md#options) for more information.
    
    ```json
    "traefik.http.routers.myrouter.tls.options": "foobar"
    ```

??? info "`traefik.http.routers.<router_name>.priority`"
    
    See [priority](../routers/index.md#priority) for more information.
    
    ```json
    "traefik.http.routers.myrouter.priority": "42"
    ```

### Services

To update the configuration of the Service automatically attached to the container,
add labels starting with `traefik.http.services.{service-name-of-your-choice}.`, followed by the option you want to change.

For example, to change the passHostHeader behavior, you'd add the label `"traefik.http.services.servicename.loadbalancer.passhostheader": "false"`.

!!! warning "The character `@` is not authorized in the service name `<service_name>`."

??? info "`traefik.http.services.<service_name>.loadbalancer.server.port`"
    
    Registers a port.
    Useful when the container exposes multiples ports.
    
    ```json
    "traefik.http.services.myservice.loadbalancer.server.port": "8080"
    ```

??? info "`traefik.http.services.<service_name>.loadbalancer.server.scheme`"
    
    Overrides the default scheme.
    
    ```json
    "traefik.http.services.myservice.loadbalancer.server.scheme": "http"
    ```

??? info "`traefik.http.services.<service_name>.loadbalancer.serverstransport`"

    Allows to reference a ServersTransport resource that is defined either with the File provider or the Kubernetes CRD one.
    See [serverstransport](../services/index.md#serverstransport) for more information.
    
    ```json
    "traefik.http.services.<service_name>.loadbalancer.serverstransport": "foobar@file"
    ```

??? info "`traefik.http.services.<service_name>.loadbalancer.passhostheader`"
    
    See [pass Host header](../services/index.md#pass-host-header) for more information.
    
    ```json
    "traefik.http.services.myservice.loadbalancer.passhostheader": "true"
    ```

??? info "`traefik.http.services.<service_name>.loadbalancer.healthcheck.headers.<header_name>`"
    
    See [health check](../services/index.md#health-check) for more information.
    
    ```json
    "traefik.http.services.myservice.loadbalancer.healthcheck.headers.X-Foo": "foobar"
    ```

??? info "`traefik.http.services.<service_name>.loadbalancer.healthcheck.hostname`"
    
    See [health check](../services/index.md#health-check) for more information.
    
    ```json
    "traefik.http.services.myservice.loadbalancer.healthcheck.hostname": "example.org"
    ```

??? info "`traefik.http.services.<service_name>.loadbalancer.healthcheck.interval`"
    
    See [health check](../services/index.md#health-check) for more information.
    
    ```json
    "traefik.http.services.myservice.loadbalancer.healthcheck.interval": "10"
    ```

??? info "`traefik.http.services.<service_name>.loadbalancer.healthcheck.path`"
    
    See [health check](../services/index.md#health-check) for more information.
    
    ```json
    "traefik.http.services.myservice.loadbalancer.healthcheck.path": "/foo"
    ```

??? info "`traefik.http.services.<service_name>.loadbalancer.healthcheck.port`"
    
    See [health check](../services/index.md#health-check) for more information.
    
    ```json
    "traefik.http.services.myservice.loadbalancer.healthcheck.port": "42"
    ```

??? info "`traefik.http.services.<service_name>.loadbalancer.healthcheck.scheme`"
    
    See [health check](../services/index.md#health-check) for more information.
    
    ```json
    "traefik.http.services.myservice.loadbalancer.healthcheck.scheme": "http"
    ```

??? info "`traefik.http.services.<service_name>.loadbalancer.healthcheck.timeout`"
    
    See [health check](../services/index.md#health-check) for more information.
    
    ```json
    "traefik.http.services.myservice.loadbalancer.healthcheck.timeout": "10"
    ```

??? info "`traefik.http.services.<service_name>.loadbalancer.healthcheck.followredirects`"
    
    See [health check](../services/index.md#health-check) for more information.
    
    ```json
    "traefik.http.services.myservice.loadbalancer.healthcheck.followredirects": "true"
    ```

??? info "`traefik.http.services.<service_name>.loadbalancer.sticky.cookie`"
    
    See [sticky sessions](../services/index.md#sticky-sessions) for more information.
    
    ```json
    "traefik.http.services.myservice.loadbalancer.sticky.cookie": "true"
    ```

??? info "`traefik.http.services.<service_name>.loadbalancer.sticky.cookie.httponly`"
    
    See [sticky sessions](../services/index.md#sticky-sessions) for more information.
    
    ```json
    "traefik.http.services.myservice.loadbalancer.sticky.cookie.httponly": "true"
    ```

??? info "`traefik.http.services.<service_name>.loadbalancer.sticky.cookie.name`"
    
    See [sticky sessions](../services/index.md#sticky-sessions) for more information.
    
    ```json
    "traefik.http.services.myservice.loadbalancer.sticky.cookie.name": "foobar"
    ```

??? info "`traefik.http.services.<service_name>.loadbalancer.sticky.cookie.secure`"
    
    See [sticky sessions](../services/index.md#sticky-sessions) for more information.
    
    ```json
    "traefik.http.services.myservice.loadbalancer.sticky.cookie.secure": "true"
    ```

??? info "`traefik.http.services.<service_name>.loadbalancer.sticky.cookie.samesite`"
    
    See [sticky sessions](../services/index.md#sticky-sessions) for more information.
    
    ```json
    "traefik.http.services.myservice.loadbalancer.sticky.cookie.samesite": "none"
    ```

??? info "`traefik.http.services.<service_name>.loadbalancer.responseforwarding.flushinterval`"
    
    See [response forwarding](../services/index.md#response-forwarding) for more information.

    ```json
    "traefik.http.services.myservice.loadbalancer.responseforwarding.flushinterval": "10"
    ```

### Middleware

You can declare pieces of middleware using labels starting with `traefik.http.middlewares.{middleware-name-of-your-choice}.`, followed by the middleware type/options.

For example, to declare a middleware [`redirectscheme`](../../middlewares/redirectscheme.md) named `my-redirect`, you'd write `"traefik.http.middlewares.my-redirect.redirectscheme.scheme": "https"`.

More information about available middlewares in the dedicated [middlewares section](../../middlewares/overview.md).

!!! warning "The character `@` is not authorized in the middleware name."

??? example "Declaring and Referencing a Middleware"

    ```json
	{
		...
		"labels": {
			"traefik.http.middlewares.my-redirect.redirectscheme.scheme": "https",
			"traefik.http.routers.my-container.middlewares": "my-redirect"
		}
	}
    ```

!!! warning "Conflicts in Declaration"

    If you declare multiple middleware with the same name but with different parameters, the middleware fails to be declared.

### TCP

You can declare TCP Routers and/or Services using labels.

??? example "Declaring TCP Routers and Services"

    ```json
	{
		...
		"labels": {
			"traefik.tcp.routers.my-router.rule": "HostSNI(`example.com`)",
			"traefik.tcp.routers.my-router.tls": "true",
			"traefik.tcp.services.my-service.loadbalancer.server.port": "4123"
		}
	}
    ```

!!! warning "TCP and HTTP"

    If you declare a TCP Router/Service, it will prevent Traefik from automatically creating an HTTP Router/Service (like it does by default if no TCP Router/Service is defined).
    You can declare both a TCP Router/Service and an HTTP Router/Service for the same container (but you have to do so manually).

#### TCP Routers

??? info "`traefik.tcp.routers.<router_name>.entrypoints`"
    
    See [entry points](../routers/index.md#entrypoints_1) for more information.
    
     ```json
     "traefik.tcp.routers.mytcprouter.entrypoints": "ep1,ep2"
     ```


??? info "`traefik.tcp.routers.<router_name>.rule`"
    
    See [rule](../routers/index.md#rule_1) for more information.
    
    ```json
    "traefik.tcp.routers.mytcprouter.rule": "HostSNI(`example.com`)"
    ```

??? info "`traefik.tcp.routers.<router_name>.service`"
    
    See [service](../routers/index.md#services) for more information.
    
    ```json
    "traefik.tcp.routers.mytcprouter.service": "myservice"
    ```

??? info "`traefik.tcp.routers.<router_name>.tls`"
    
    See [TLS](../routers/index.md#tls_1) for more information.
    
    ```json
    "traefik.tcp.routers.mytcprouter.tls": "true
    ```

??? info "`traefik.tcp.routers.<router_name>.tls.certresolver`"
    
    See [certResolver](../routers/index.md#certresolver_1) for more information.
    
    ```json
    "traefik.tcp.routers.mytcprouter.tls.certresolver": "myresolver"
    ```

??? info "`traefik.tcp.routers.<router_name>.tls.domains[n].main`"
    
    See [domains](../routers/index.md#domains_1) for more information.
    
    ```json
    "traefik.tcp.routers.mytcprouter.tls.domains[0].main": "example.org"
    ```

??? info "`traefik.tcp.routers.<router_name>.tls.domains[n].sans`"
    
    See [domains](../routers/index.md#domains_1) for more information.
    
    ```json
    "traefik.tcp.routers.mytcprouter.tls.domains[0].sans": "test.example.org,dev.example.org"
    ```

??? info "`traefik.tcp.routers.<router_name>.tls.options`"
    
    See [options](../routers/index.md#options_1) for more information.
    
    ```json
    "traefik.tcp.routers.mytcprouter.tls.options": "mysoptions"
    ```

??? info "`traefik.tcp.routers.<router_name>.tls.passthrough`"
    
    See [TLS](../routers/index.md#tls_1) for more information.
    
    ```json
    "traefik.tcp.routers.mytcprouter.tls.passthrough": "true"
    ```

#### TCP Services

??? info "`traefik.tcp.services.<service_name>.loadbalancer.server.port`"
    
    Registers a port of the application.
    
    ```json
    "traefik.tcp.services.mytcpservice.loadbalancer.server.port": "423"
    ```

??? info "`traefik.tcp.services.<service_name>.loadbalancer.terminationdelay`"
        
    See [termination delay](../services/index.md#termination-delay) for more information.
    
    ```json
    "traefik.tcp.services.mytcpservice.loadbalancer.terminationdelay": "100"
    ```

??? info "`traefik.tcp.services.<service_name>.loadbalancer.proxyprotocol.version`"
        
    See [PROXY protocol](../services/index.md#proxy-protocol) for more information.
    
    ```json
    "traefik.tcp.services.mytcpservice.loadbalancer.proxyprotocol.version": "1"
    ```

### UDP

You can declare UDP Routers and/or Services using labels.

??? example "Declaring UDP Routers and Services"

    ```json
	{
		...
		"labels": {
			"traefik.udp.routers.my-router.entrypoints": "udp",
			"traefik.udp.services.my-service.loadbalancer.server.port": "4123"
		}
	}
    ```

!!! warning "UDP and HTTP"

    If you declare a UDP Router/Service, it will prevent Traefik from automatically creating an HTTP Router/Service (like it does by default if no UDP Router/Service is defined).
    You can declare both a UDP Router/Service and an HTTP Router/Service for the same container (but you have to do so manually).

#### UDP Routers

??? info "`traefik.udp.routers.<router_name>.entrypoints`"
    
    See [entry points](../routers/index.md#entrypoints_2) for more information.
    
     ```json
     "traefik.udp.routers.myudprouter.entrypoints": "ep1,ep2"
     ```

??? info "`traefik.udp.routers.<router_name>.service`"
    
    See [service](../routers/index.md#services_1) for more information.
    
    ```json
    "traefik.udp.routers.myudprouter.service": "myservice"
    ```

#### UDP Services

??? info "`traefik.udp.services.<service_name>.loadbalancer.server.port`"
    
    Registers a port of the application.
    
    ```json
    "traefik.udp.services.myudpservice.loadbalancer.server.port": "423"
    ```

### Specific Provider Options

#### `traefik.enable`

```json
"traefik.enable": "true"
```

Setting this option controls whether Traefik exposes the application.
It overrides the value of `exposedByDefault`.

#### `traefik.marathon.ipadressidx`

```json
"traefik.marathon.ipadressidx": "1"
```

If a task has several IP addresses, this option specifies which one, in the list of available addresses, to select.
