# Traefik & Docker

A Story of Labels & Containers
{: .subtitle }

![Docker](../assets/img/providers/docker.png)

Attach labels to your containers and let Traefik do the rest!

Traefik works with both [Docker (standalone) Engine](https://docs.docker.com/engine/)
and [Docker Swarm Mode](https://docs.docker.com/engine/swarm/).

!!! tip "The Quick Start Uses Docker"

    If you have not already read it, maybe you would like to go through the [quick start guide](../getting-started/quick-start.md) that uses the Docker provider.

## Configuration Examples

??? example "Configuring Docker & Deploying / Exposing Services"

    Enabling the docker provider

    ```yaml tab="File (YAML)"
    providers:
      docker: {}
    ```

    ```toml tab="File (TOML)"
    [providers.docker]
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

??? example "Configuring Docker Swarm & Deploying / Exposing Services"

    Enabling the docker provider (Swarm Mode)

    ```yaml tab="File (YAML)"
    providers:
      docker:
        # swarm classic (1.12-)
        # endpoint: "tcp://127.0.0.1:2375"
        # docker swarm mode (1.12+)
        endpoint: "tcp://127.0.0.1:2377"
        swarmMode: true
    ```

    ```toml tab="File (TOML)"
    [providers.docker]
      # swarm classic (1.12-)
      # endpoint = "tcp://127.0.0.1:2375"
      # docker swarm mode (1.12+)
      endpoint = "tcp://127.0.0.1:2377"
      swarmMode = true
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

## Routing Configuration

When using Docker as a [provider](./overview.md),
Traefik uses [container labels](https://docs.docker.com/engine/reference/commandline/run/#set-metadata-on-container--l---label---label-file) to retrieve its routing configuration.

See the list of labels in the dedicated [routing](../routing/providers/docker.md) section.

### Routing Configuration with Labels

By default, Traefik watches for [container level labels](https://docs.docker.com/config/labels-custom-metadata/) on a standalone Docker Engine.

When using Docker Compose, labels are specified by the directive
[`labels`](https://docs.docker.com/compose/compose-file/compose-file-v3/#labels) from the
["services" objects](https://docs.docker.com/compose/compose-file/compose-file-v3/#service-configuration-reference).

!!! tip "Not Only Docker"

    Please note that any tool like Nomad, Terraform, Ansible, etc.
    that is able to define a Docker container with labels can work
    with Traefik and the Docker provider.

### Port Detection

Traefik retrieves the private IP and port of containers from the Docker API.

Port detection works as follows:

- If a container [exposes](https://docs.docker.com/engine/reference/builder/#expose) a single port,
  then Traefik uses this port for private communication.
- If a container [exposes](https://docs.docker.com/engine/reference/builder/#expose) multiple ports,
  or does not expose any port, then you must manually specify which port Traefik should use for communication
  by using the label `traefik.http.services.<service_name>.loadbalancer.server.port`
  (Read more on this label in the dedicated section in [routing](../routing/providers/docker.md#port)).

### Host networking

When exposing containers that are configured with [host networking](https://docs.docker.com/network/host/),
the IP address of the host is resolved as follows:

<!-- TODO: verify and document the swarm mode case with container.Node.IPAddress coming from the API -->
- try a lookup of `host.docker.internal`
- if the lookup was unsuccessful, fall back to `127.0.0.1`

On Linux, for versions of Docker older than 20.10.0, for `host.docker.internal` to be defined, it should be provided
as an `extra_host` to the Traefik container, using the `--add-host` flag. For example, to set it to the IP address of
the bridge interface (`docker0` by default): `--add-host=host.docker.internal:172.17.0.1`

### Docker API Access

Traefik requires access to the docker socket to get its dynamic configuration.

You can specify which Docker API Endpoint to use with the directive [`endpoint`](#endpoint).

!!! warning "Security Note"

    Accessing the Docker API without any restriction is a security concern:
    If Traefik is attacked, then the attacker might get access to the underlying host.
    {: #security-note }

    As explained in the [Docker Daemon Attack Surface documentation](https://docs.docker.com/engine/security/#docker-daemon-attack-surface):

    !!! quote

        [...] only **trusted** users should be allowed to control your Docker daemon [...]

    ??? success "Solutions"

        Expose the Docker socket over TCP or SSH, instead of the default Unix socket file.
        It allows different implementation levels of the [AAA (Authentication, Authorization, Accounting) concepts](https://en.wikipedia.org/wiki/AAA_(computer_security)), depending on your security assessment:

        - Authentication with Client Certificates as described in ["Protect the Docker daemon socket."](https://docs.docker.com/engine/security/protect-access/)
        - Authorize and filter requests to restrict possible actions with [the TecnativaDocker Socket Proxy](https://github.com/Tecnativa/docker-socket-proxy).
        - Authorization with the [Docker Authorization Plugin Mechanism](https://web.archive.org/web/20190920092526/https://docs.docker.com/engine/extend/plugins_authorization/)
        - Accounting at networking level, by exposing the socket only inside a Docker private network, only available for Traefik.
        - Accounting at container level, by exposing the socket on a another container than Traefik's.
          With Swarm mode, it allows scheduling of Traefik on worker nodes, with only the "socket exposer" container on the manager nodes.
        - Accounting at kernel level, by enforcing kernel calls with mechanisms like [SELinux](https://en.wikipedia.org/wiki/Security-Enhanced_Linux), to only allows an identified set of actions for Traefik's process (or the "socket exposer" process).
        - SSH public key authentication (SSH is supported with Docker > 18.09)

    ??? info "More Resources and Examples"

        - ["Paranoid about mounting /var/run/docker.sock?"](https://medium.com/@containeroo/traefik-2-0-paranoid-about-mounting-var-run-docker-sock-22da9cb3e78c)
        - [Traefik and Docker: A Discussion with Docker Captain, Bret Fisher](https://blog.traefik.io/traefik-and-docker-a-discussion-with-docker-captain-bret-fisher-7f0b9a54ff88)
        - [KubeCon EU 2018 Keynote, Running with Scissors, from Liz Rice](https://www.youtube.com/watch?v=ltrV-Qmh3oY)
        - [Don't expose the Docker socket (not even to a container)](https://www.lvh.io/posts/dont-expose-the-docker-socket-not-even-to-a-container/)
        - [A thread on Stack Overflow about sharing the `/var/run/docker.sock` file](https://news.ycombinator.com/item?id=17983623)
        - [To DinD or not to DinD](https://blog.loof.fr/2018/01/to-dind-or-not-do-dind.html)
        - [Traefik issue GH-4174 about security with Docker socket](https://github.com/traefik/traefik/issues/4174)
        - [Inspecting Docker Activity with Socat](https://developers.redhat.com/blog/2015/02/25/inspecting-docker-activity-with-socat/)
        - [Letting Traefik run on Worker Nodes](https://blog.mikesir87.io/2018/07/letting-traefik-run-on-worker-nodes/)
        - [Docker Socket Proxy from Tecnativa](https://github.com/Tecnativa/docker-socket-proxy)

## Docker Swarm Mode

To enable Docker Swarm (instead of standalone Docker) as a configuration provider,
set the [`swarmMode`](#swarmmode) directive to `true`.

### Routing Configuration with Labels

While in Swarm Mode, Traefik uses labels found on services, not on individual containers.

Therefore, if you use a compose file with Swarm Mode, labels should be defined in the
[`deploy`](https://docs.docker.com/compose/compose-file/compose-file-v3/#labels-1) part of your service.

This behavior is only enabled for docker-compose version 3+ ([Compose file reference](https://docs.docker.com/compose/compose-file/compose-file-v3/)).

### Port Detection

Docker Swarm does not provide any [port detection](#port-detection) information to Traefik.

Therefore, you **must** specify the port to use for communication by using the label `traefik.http.services.<service_name>.loadbalancer.server.port`
(Check the reference for this label in the [routing section for Docker](../routing/providers/docker.md#port)).

### Docker API Access

Docker Swarm Mode follows the same rules as Docker [API Access](#docker-api-access).

Since the Swarm API is only exposed on the [manager nodes](https://docs.docker.com/engine/swarm/how-swarm-mode-works/nodes/#manager-nodes),
these are the nodes that Traefik should be scheduled on by deploying Traefik with a constraint on the node "role":

```shell tab="With Docker CLI"
docker service create \
  --constraint=node.role==manager \
  #... \
```

```yml tab="With Docker Compose"
version: '3'

services:
  traefik:
    # ...
    deploy:
      placement:
        constraints:
          - node.role == manager
```

!!! tip "Scheduling Traefik on Worker Nodes"

    Following the guidelines given in the previous section ["Docker API Access"](#docker-api-access),
    if you expose the Docker API through TCP, then Traefik can be scheduled on any node if the TCP
    socket is reachable.

    Please consider the security implications by reading the [Security Note](#security-note).

    A good example can be found on [Bret Fisher's repository](https://github.com/BretFisher/dogvscat/blob/master/stack-proxy-global.yml#L124).

## Provider Configuration

### `endpoint`

_Required, Default="unix:///var/run/docker.sock"_

See the sections [Docker API Access](#docker-api-access) and [Docker Swarm API Access](#docker-api-access_1) for more information.

??? example "Using the docker.sock"

    The docker-compose file shares the docker sock with the Traefik container

    ```yaml
    version: '3'

    services:
      traefik:
         image: traefik:v2.5 # The official v2 Traefik docker image
         ports:
           - "80:80"
         volumes:
           - /var/run/docker.sock:/var/run/docker.sock
    ```

    We specify the docker.sock in traefik's configuration file.

    ```yaml tab="File (YAML)"
    providers:
      docker:
        endpoint: "unix:///var/run/docker.sock"
         # ...
    ```

    ```toml tab="File (TOML)"
    [providers.docker]
      endpoint = "unix:///var/run/docker.sock"
      # ...
    ```

    ```bash tab="CLI"
    --providers.docker.endpoint=unix:///var/run/docker.sock
    # ...
    ```

??? example "Using SSH"

    Using Docker 18.09+ you can connect Traefik to daemon using SSH
    We specify the SSH host and user in Traefik's configuration file.
    Note that is server requires public keys for authentication you must have those accessible for user who runs Traefik.

    ```yaml tab="File (YAML)"
    providers:
      docker:
        endpoint: "ssh://traefik@192.168.2.5:2022"
         # ...
    ```

    ```toml tab="File (TOML)"
    [providers.docker]
      endpoint = "ssh://traefik@192.168.2.5:2022"
      # ...
    ```

    ```bash tab="CLI"
    --providers.docker.endpoint=ssh://traefik@192.168.2.5:2022
    # ...
    ```

```yaml tab="File (YAML)"
providers:
  docker:
    endpoint: "unix:///var/run/docker.sock"
```

```toml tab="File (TOML)"
[providers.docker]
  endpoint = "unix:///var/run/docker.sock"
```

```bash tab="CLI"
--providers.docker.endpoint=unix:///var/run/docker.sock
```

### `useBindPortIP`

_Optional, Default=false_

Traefik routes requests to the IP/port of the matching container.
When setting `useBindPortIP=true`, you tell Traefik to use the IP/Port attached to the container's _binding_ instead of its inner network IP/Port.

When used in conjunction with the `traefik.http.services.<name>.loadbalancer.server.port` label (that tells Traefik to route requests to a specific port),
Traefik tries to find a binding on port `traefik.http.services.<name>.loadbalancer.server.port`.
If it cannot find such a binding, Traefik falls back on the internal network IP of the container,
but still uses the `traefik.http.services.<name>.loadbalancer.server.port` that is set in the label.

??? example "Examples of `usebindportip` in different situations."

    | port label         | Container's binding                                | Routes to      |
    |--------------------|----------------------------------------------------|----------------|
    |          -         |           -                                        | IntIP:IntPort  |
    |          -         | ExtPort:IntPort                                    | IntIP:IntPort  |
    |          -         | ExtIp:ExtPort:IntPort                              | ExtIp:ExtPort  |
    | LblPort            |           -                                        | IntIp:LblPort  |
    | LblPort            | ExtIp:ExtPort:LblPort                              | ExtIp:ExtPort  |
    | LblPort            | ExtIp:ExtPort:OtherPort                            | IntIp:LblPort  |
    | LblPort            | ExtIp1:ExtPort1:IntPort1 & ExtIp2:LblPort:IntPort2 | ExtIp2:LblPort |

    !!! info ""
        In the above table:

        - `ExtIp` stands for "external IP found in the binding"
        - `IntIp` stands for "internal network container's IP",
        - `ExtPort` stands for "external Port found in the binding"
        - `IntPort` stands for "internal network container's port."

```yaml tab="File (YAML)"
providers:
  docker:
    useBindPortIP: true
    # ...
```

```toml tab="File (TOML)"
[providers.docker]
  useBindPortIP = true
  # ...
```

```bash tab="CLI"
--providers.docker.useBindPortIP=true
# ...
```

### `exposedByDefault`

_Optional, Default=true_

Expose containers by default through Traefik.
If set to `false`, containers that do not have a `traefik.enable=true` label are ignored from the resulting routing configuration.

For additional information, refer to [Restrict the Scope of Service Discovery](./overview.md#restrict-the-scope-of-service-discovery).

```yaml tab="File (YAML)"
providers:
  docker:
    exposedByDefault: false
    # ...
```

```toml tab="File (TOML)"
[providers.docker]
  exposedByDefault = false
  # ...
```

```bash tab="CLI"
--providers.docker.exposedByDefault=false
# ...
```

### `network`

_Optional, Default=""_

Defines a default docker network to use for connections to all containers.

This option can be overridden on a per-container basis with the `traefik.docker.network` label.

```yaml tab="File (YAML)"
providers:
  docker:
    network: test
    # ...
```

```toml tab="File (TOML)"
[providers.docker]
  network = "test"
  # ...
```

```bash tab="CLI"
--providers.docker.network=test
# ...
```

### `defaultRule`

_Optional, Default=```Host(`{{ normalize .Name }}`)```_

The `defaultRule` option defines what routing rule to apply to a container if no rule is defined by a label.

It must be a valid [Go template](https://golang.org/pkg/text/template/), and can use
[sprig template functions](http://masterminds.github.io/sprig/).
The container service name can be accessed with the `Name` identifier,
and the template has access to all the labels defined on this container.

```yaml tab="File (YAML)"
providers:
  docker:
    defaultRule: "Host(`{{ .Name }}.{{ index .Labels \"customLabel\"}}`)"
    # ...
```

```toml tab="File (TOML)"
[providers.docker]
  defaultRule = "Host(`{{ .Name }}.{{ index .Labels \"customLabel\"}}`)"
  # ...
```

```bash tab="CLI"
--providers.docker.defaultRule=Host(`{{ .Name }}.{{ index .Labels \"customLabel\"}}`)
# ...
```

### `swarmMode`

_Optional, Default=false_

Enables the Swarm Mode (instead of standalone Docker).

```yaml tab="File (YAML)"
providers:
  docker:
    swarmMode: true
    # ...
```

```toml tab="File (TOML)"
[providers.docker]
  swarmMode = true
  # ...
```

```bash tab="CLI"
--providers.docker.swarmMode=true
# ...
```

### `swarmModeRefreshSeconds`

_Optional, Default=15_

Defines the polling interval (in seconds) for Swarm Mode.

```yaml tab="File (YAML)"
providers:
  docker:
    swarmModeRefreshSeconds: 30
    # ...
```

```toml tab="File (TOML)"
[providers.docker]
  swarmModeRefreshSeconds = 30
  # ...
```

```bash tab="CLI"
--providers.docker.swarmModeRefreshSeconds=30
# ...
```

### `httpClientTimeout`

_Optional, Default=0_

Defines the client timeout (in seconds) for HTTP connections. If its value is `0`, no timeout is set.

```yaml tab="File (YAML)"
providers:
  docker:
    httpClientTimeout: 300
    # ...
```

```toml tab="File (TOML)"
[providers.docker]
  httpClientTimeout = 300
  # ...
```

```bash tab="CLI"
--providers.docker.httpClientTimeout=300
# ...
```

### `watch`

_Optional, Default=true_

Watch Docker Swarm events.

```yaml tab="File (YAML)"
providers:
  docker:
    watch: false
    # ...
```

```toml tab="File (TOML)"
[providers.docker]
  watch = false
  # ...
```

```bash tab="CLI"
--providers.docker.watch=false
# ...
```

### `constraints`

_Optional, Default=""_

The `constraints` option can be set to an expression that Traefik matches against the container tags to determine whether
to create any route for that container. If none of the container tags match the expression, no route for that container is
created. If the expression is empty, all detected containers are included.

The expression syntax is based on the ```Tag(`tag`)```, and ```TagRegex(`tag`)``` functions,
as well as the usual boolean logic, as shown in examples below.

??? example "Constraints Expression Examples"

    ```toml
    # Includes only containers having a label with key `a.label.name` and value `foo`
    constraints = "Label(`a.label.name`, `foo`)"
    ```

    ```toml
    # Excludes containers having any label with key `a.label.name` and value `foo`
    constraints = "!Label(`a.label.name`, `value`)"
    ```

    ```toml
    # With logical AND.
    constraints = "Label(`a.label.name`, `valueA`) && Label(`another.label.name`, `valueB`)"
    ```

    ```toml
    # With logical OR.
    constraints = "Label(`a.label.name`, `valueA`) || Label(`another.label.name`, `valueB`)"
    ```

    ```toml
    # With logical AND and OR, with precedence set by parentheses.
    constraints = "Label(`a.label.name`, `valueA`) && (Label(`another.label.name`, `valueB`) || Label(`yet.another.label.name`, `valueC`))"
    ```

    ```toml
    # Includes only containers having a label with key `a.label.name` and a value matching the `a.+` regular expression.
    constraints = "LabelRegex(`a.label.name`, `a.+`)"
    ```

For additional information, refer to [Restrict the Scope of Service Discovery](./overview.md#restrict-the-scope-of-service-discovery).

```yaml tab="File (YAML)"
providers:
  docker:
    constraints: "Label(`a.label.name`,`foo`)"
    # ...
```

```toml tab="File (TOML)"
[providers.docker]
  constraints = "Label(`a.label.name`,`foo`)"
  # ...
```

```bash tab="CLI"
--providers.docker.constraints=Label(`a.label.name`,`foo`)
# ...
```

### `tls`

_Optional_

#### `tls.ca`

Certificate Authority used for the secure connection to Docker.

```yaml tab="File (YAML)"
providers:
  docker:
    tls:
      ca: path/to/ca.crt
```

```toml tab="File (TOML)"
[providers.docker.tls]
  ca = "path/to/ca.crt"
```

```bash tab="CLI"
--providers.docker.tls.ca=path/to/ca.crt
```

#### `tls.caOptional`

The value of `tls.caOptional` defines which policy should be used for the secure connection with TLS Client Authentication to Docker.

!!! warning ""

    If `tls.ca` is undefined, this option will be ignored, and no client certificate will be requested during the handshake. Any provided certificate will thus never be verified.

When this option is set to `true`, a client certificate is requested during the handshake but is not required. If a certificate is sent, it is required to be valid.

When this option is set to `false`, a client certificate is requested during the handshake, and at least one valid certificate should be sent by the client.

```yaml tab="File (YAML)"
providers:
  docker:
    tls:
      caOptional: true
```

```toml tab="File (TOML)"
[providers.docker.tls]
  caOptional = true
```

```bash tab="CLI"
--providers.docker.tls.caOptional=true
```

#### `tls.cert`

Public certificate used for the secure connection to Docker.

```yaml tab="File (YAML)"
providers:
  docker:
    tls:
      cert: path/to/foo.cert
      key: path/to/foo.key
```

```toml tab="File (TOML)"
[providers.docker.tls]
  cert = "path/to/foo.cert"
  key = "path/to/foo.key"
```

```bash tab="CLI"
--providers.docker.tls.cert=path/to/foo.cert
--providers.docker.tls.key=path/to/foo.key
```

#### `tls.key`

Private certificate used for the secure connection to Docker.

```yaml tab="File (YAML)"
providers:
  docker:
    tls:
      cert: path/to/foo.cert
      key: path/to/foo.key
```

```toml tab="File (TOML)"
[providers.docker.tls]
  cert = "path/to/foo.cert"
  key = "path/to/foo.key"
```

```bash tab="CLI"
--providers.docker.tls.cert=path/to/foo.cert
--providers.docker.tls.key=path/to/foo.key
```

#### `tls.insecureSkipVerify`

If `insecureSkipVerify` is `true`, the TLS connection to Docker accepts any certificate presented by the server regardless of the hostnames it covers.

```yaml tab="File (YAML)"
providers:
  docker:
    tls:
      insecureSkipVerify: true
```

```toml tab="File (TOML)"
[providers.docker.tls]
  insecureSkipVerify = true
```

```bash tab="CLI"
--providers.docker.tls.insecureSkipVerify=true
```
