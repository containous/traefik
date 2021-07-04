# Traefik & Rancher

A Story of Labels, Services & Containers
{: .subtitle }

![Rancher](../assets/img/providers/rancher.png)

Attach labels to your services and let Traefik do the rest!

!!! important "This provider is specific to Rancher 1.x."

    Rancher 2.x requires Kubernetes and does not have a metadata endpoint of its own for Traefik to query.
    As such, Rancher 2.x users should utilize the [Kubernetes CRD provider](./kubernetes-crd.md) directly.

## Configuration Examples

??? example "Configuring Rancher & Deploying / Exposing Services"

    Enabling the Rancher provider

    ```yaml tab="File (YAML)"
    providers:
      rancher: {}
    ```

    ```toml tab="File (TOML)"
    [providers.rancher]
    ```

    ```bash tab="CLI"
    --providers.rancher=true
    ```

    Attaching labels to services

    ```yaml
    labels:
      - traefik.http.services.my-service.rule=Host(`example.com`)
    ```

## Routing Configuration

See the dedicated section in [routing](../routing/providers/rancher.md).

## Provider Configuration

??? tip "Browse the Reference"

    For an overview of all the options that can be set with the Rancher provider, see the following snippets:

    ```yaml tab="File (YAML)"
    --8<-- "content/providers/rancher.yml"
    ```

    ```toml tab="File (TOML)"
    --8<-- "content/providers/rancher.toml"
    ```

    ```bash tab="CLI"
    --8<-- "content/providers/rancher.txt"
    ```

### `exposedByDefault`

_Optional, Default=true_

Expose Rancher services by default in Traefik.
If set to `false`, services that do not have a `traefik.enable=true` label are ignored from the resulting routing configuration.

For additional information, refer to [Restrict the Scope of Service Discovery](./overview.md#restrict-the-scope-of-service-discovery).

```yaml tab="File (YAML)"
providers:
  rancher:
    exposedByDefault: false
    # ...
```

```toml tab="File (TOML)"
[providers.rancher]
  exposedByDefault = false
  # ...
```

```bash tab="CLI"
--providers.rancher.exposedByDefault=false
# ...
```

### `defaultRule`

_Optional, Default=```Host(`{{ normalize .Name }}`)```_

The default host rule for all services.

The `defaultRule` option defines what routing rule to apply to a container if no rule is defined by a label.

It must be a valid [Go template](https://golang.org/pkg/text/template/), and can use
[sprig template functions](http://masterminds.github.io/sprig/).
The service name can be accessed with the `Name` identifier,
and the template has access to all the labels defined on this container.

This option can be overridden on a container basis with the `traefik.http.routers.Router1.rule` label.

```yaml tab="File (YAML)"
providers:
  rancher:
    defaultRule: "Host(`{{ .Name }}.{{ index .Labels \"customLabel\"}}`)"
    # ...
```

```toml tab="File (TOML)"
[providers.rancher]
  defaultRule = "Host(`{{ .Name }}.{{ index .Labels \"customLabel\"}}`)"
  # ...
```

```bash tab="CLI"
--providers.rancher.defaultRule=Host(`{{ .Name }}.{{ index .Labels \"customLabel\"}}`)
# ...
```

### `enableServiceHealthFilter`

_Optional, Default=true_

Filter out services with unhealthy states and inactive states.

```yaml tab="File (YAML)"
providers:
  rancher:
    enableServiceHealthFilter: false
    # ...
```

```toml tab="File (TOML)"
[providers.rancher]
  enableServiceHealthFilter = false
  # ...
```

```bash tab="CLI"
--providers.rancher.enableServiceHealthFilter=false
# ...
```

### `refreshSeconds`

_Optional, Default=15_

Defines the polling interval (in seconds).

```yaml tab="File (YAML)"
providers:
  rancher:
    refreshSeconds: 30
    # ...
```

```toml tab="File (TOML)"
[providers.rancher]
  refreshSeconds = 30
  # ...
```

```bash tab="CLI"
--providers.rancher.refreshSeconds=30
# ...
```

### `intervalPoll`

_Optional, Default=false_

Poll the Rancher metadata service for changes every `rancher.refreshSeconds`,
which is less accurate than the default long polling technique which provides near instantaneous updates to Traefik.

```yaml tab="File (YAML)"
providers:
  rancher:
    intervalPoll: true
    # ...
```

```toml tab="File (TOML)"
[providers.rancher]
  intervalPoll = true
  # ...
```

```bash tab="CLI"
--providers.rancher.intervalPoll=true
# ...
```

### `prefix`

_Optional, Default="/latest"_

Prefix used for accessing the Rancher metadata service.

```yaml tab="File (YAML)"
providers:
  rancher:
    prefix: "/test"
    # ...
```

```toml tab="File (TOML)"
[providers.rancher]
  prefix = "/test"
  # ...
```

```bash tab="CLI"
--providers.rancher.prefix=/test
# ...
```

### `constraints`

_Optional, Default=""_

The `constraints` option can be set to an expression that Traefik matches against the container labels to determine whether
to create any route for that container. If none of the container tags match the expression, no route for that container is
created. If the expression is empty, all detected containers are included.

The expression syntax is based on the `Label("key", "value")`, and `LabelRegex("key", "value")` functions, as well as
the usual boolean logic, as shown in examples below.

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
  rancher:
    constraints: "Label(`a.label.name`,`foo`)"
    # ...
```

```toml tab="File (TOML)"
[providers.rancher]
  constraints = "Label(`a.label.name`,`foo`)"
  # ...
```

```bash tab="CLI"
--providers.rancher.constraints=Label(`a.label.name`,`foo`)
# ...
```
