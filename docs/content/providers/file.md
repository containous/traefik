# Traefik & File

Good Old Configuration File
{: .subtitle }

The file provider lets you define the [dynamic configuration](./overview.md) in a YAML or TOML file.

It supports providing configuration through a [single configuration file](#filename) or [multiple separate files](#directory).

!!! info

    The file provider is the default format used throughout the documentation to show samples of the configuration for many features.

!!! tip

    The file provider can be a good solution for reusing common elements from other providers (e.g. declaring whitelist middlewares, basic authentication, ...)

## Configuration Examples

??? example "Declaring Routers, Middlewares & Services"

    Enabling the file provider:

    ```yaml tab="File (YAML)"
    providers:
      file:
        directory: "/path/to/dynamic/conf"
    ```

    ```toml tab="File (TOML)"
    [providers.file]
      directory = "/path/to/dynamic/conf"
    ```

    ```bash tab="CLI"
    --providers.file.directory=/path/to/dynamic/conf
    ```

    Declaring Routers, Middlewares & Services:

    ```yaml tab="YAML"
    http:
      # Add the router
      routers:
        router0:
          entryPoints:
          - web
          middlewares:
          - my-basic-auth
          service: service-foo
          rule: Path(`/foo`)

      # Add the middleware
      middlewares:
        my-basic-auth:
          basicAuth:
            users:
            - test:$apr1$H6uskkkW$IgXLP6ewTrSuBkTrqE8wj/
            - test2:$apr1$d9hr9HBB$4HxwgUir3HP4EsggP/QNo0
            usersFile: etc/traefik/.htpasswd

      # Add the service
      services:
        service-foo:
          loadBalancer:
            servers:
            - url: http://foo/
            - url: http://bar/
            passHostHeader: false
    ```

    ```toml tab="TOML"
    [http]
      # Add the router
      [http.routers]
        [http.routers.router0]
          entryPoints = ["web"]
          middlewares = ["my-basic-auth"]
          service = "service-foo"
          rule = "Path(`/foo`)"

      # Add the middleware
      [http.middlewares]
        [http.middlewares.my-basic-auth.basicAuth]
          users = ["test:$apr1$H6uskkkW$IgXLP6ewTrSuBkTrqE8wj/",
                    "test2:$apr1$d9hr9HBB$4HxwgUir3HP4EsggP/QNo0"]
          usersFile = "etc/traefik/.htpasswd"

      # Add the service
      [http.services]
        [http.services.service-foo]
          [http.services.service-foo.loadBalancer]
            [[http.services.service-foo.loadBalancer.servers]]
              url = "http://foo/"
            [[http.services.service-foo.loadBalancer.servers]]
              url = "http://bar/"
    ```

## Provider Configuration

For an overview of all the options that can be set with the file provider, see the [dynamic configuration](../reference/dynamic-configuration/file.md) and [static configuration](../reference/static-configuration/overview.md) references.

!!! warning "Limitations"

    With the file provider, Traefik listens for file system notifications to update the dynamic configuration.

    If you use a mounted/bound file system in your orchestrator (like docker or kubernetes), the way the files are linked may be a source of errors.
    If the link between the file systems is broken, when a source file/directory is changed/renamed, nothing will be reported to the linked file/directory, so the file system notifications will be neither triggered nor caught.

    For example, in Docker, if the host file is renamed, the link to the mounted file is broken and the container's file is no longer updated.
    To avoid this kind of issue, it is recommended to:

    * set the Traefik [**directory**](#directory) configuration with the parent directory
    * mount/bind the parent directory

    As it is very difficult to listen to all file system notifications, Traefik uses [fsnotify](https://github.com/fsnotify/fsnotify).
    If using a directory with a mounted directory does not fix your issue, please check your file system compatibility with fsnotify.

### `filename`

Defines the path to the configuration file.

!!! warning ""

    The `filename` and `directory` options are mutually exclusive.
    It is recommended to use `directory`.

```yaml tab="File (YAML)"
providers:
  file:
    filename: /path/to/config/dynamic_conf.yml
```

```toml tab="File (TOML)"
[providers]
  [providers.file]
    filename = "/path/to/config/dynamic_conf.toml"
```

```bash tab="CLI"
--providers.file.filename=/path/to/config/dynamic_conf.yml
```

### `directory`

Defines the path to the directory that contains the configuration files.

!!! warning ""

    The `filename` and `directory` options are mutually exclusive.
    It is recommended to use `directory`.

```yaml tab="File (YAML)"
providers:
  file:
    directory: /path/to/config
```

```toml tab="File (TOML)"
[providers]
  [providers.file]
    directory = "/path/to/config"
```

```bash tab="CLI"
--providers.file.directory=/path/to/config
```

### `watch`

Set the `watch` option to `true` to allow Traefik to automatically watch for file changes.
It works with both the `filename` and the `directory` options.

```yaml tab="File (YAML)"
providers:
  file:
    directory: /path/to/dynamic/conf
    watch: true
```

```toml tab="File (TOML)"
[providers]
  [providers.file]
    directory = "/path/to/dynamic/conf"
    watch = true
```

```bash tab="CLI"
--providers.file.directory=/my/path/to/dynamic/conf
--providers.file.watch=true
```

### Go Templating

!!! warning

    Go Templating only works with dedicated dynamic configuration files.
    Templating does not work in the Traefik main static configuration file.

Traefik supports using Go templating to automatically generate repetitive sections of configuration files.
These sections must be a valid [Go template](https://golang.org/pkg/text/template/), and can use
[sprig template functions](http://masterminds.github.io/sprig/).

To illustrate, it is possible to easily define multiple routers, services, and TLS certificates as described in the following examples:

??? example "Configuring Using Templating"

    ```yaml tab="YAML"
    http:
      routers:
        {{range $i, $e := until 100 }}
        router{{ $e }}-{{ env "MY_ENV_VAR" }}:
          # ...
        {{end}}

      services:
        {{range $i, $e := until 100 }}
        application{{ $e }}:
          # ...
        {{end}}

    tcp:
      routers:
        {{range $i, $e := until 100 }}
        router{{ $e }}:
          # ...
        {{end}}

      services:
        {{range $i, $e := until 100 }}
        service{{ $e }}:
          # ...
        {{end}}

    tls:
      certificates:
      {{ range $i, $e := until 10 }}
      - certFile: "/etc/traefik/cert-{{ $e }}.pem"
        keyFile: "/etc/traefik/cert-{{ $e }}.key"
        store:
        - "my-store-foo-{{ $e }}"
        - "my-store-bar-{{ $e }}"
      {{end}}
    ```

    ```toml tab="TOML"
    # template-rules.toml
    [http]

      [http.routers]
      {{ range $i, $e := until 100 }}
        [http.routers.router{{ $e }}-{{ env "MY_ENV_VAR" }}]
        # ...
      {{ end }}

      [http.services]
      {{ range $i, $e := until 100 }}
          [http.services.service{{ $e }}]
          # ...
      {{ end }}

    [tcp]

      [tcp.routers]
      {{ range $i, $e := until 100 }}
        [tcp.routers.router{{ $e }}]
        # ...
      {{ end }}

      [tcp.services]
      {{ range $i, $e := until 100 }}
          [http.services.service{{ $e }}]
          # ...
      {{ end }}

    {{ range $i, $e := until 10 }}
    [[tls.certificates]]
      certFile = "/etc/traefik/cert-{{ $e }}.pem"
      keyFile = "/etc/traefik/cert-{{ $e }}.key"
      stores = ["my-store-foo-{{ $e }}", "my-store-bar-{{ $e }}"]
    {{ end }}

    [tls.config]
    {{ range $i, $e := until 10 }}
      [tls.config.TLS{{ $e }}]
      # ...
    {{ end }}
    ```
