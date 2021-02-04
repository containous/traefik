# Traefik & Marathon

Traefik can be configured to use Marathon as a provider.
{: .subtitle }

See also [Marathon user guide](../user-guides/marathon.md).

## Configuration Examples

??? example "Configuring Marathon & Deploying / Exposing Applications"

    Enabling the marathon provider

    ```toml tab="File (TOML)"
    [providers.marathon]
    ```
    
    ```yaml tab="File (YAML)"
    providers:
      marathon: {}
    ```
    
    ```bash tab="CLI"
    --providers.marathon=true
    ```

    Attaching labels to marathon applications

    ```json
	{
		"id": "/whoami",
		"container": {
			"type": "DOCKER",
			"docker": {
				"image": "traefik/whoami",
				"network": "BRIDGE",
				"portMappings": [
					{
						"containerPort": 80,
						"hostPort": 0,
						"protocol": "tcp"
					}
				]
			}
		},
		"labels": {
			"traefik.http.Routers.app.Rule": "PathPrefix(`/app`)"
		}
	}
    ```

## Routing Configuration

See the dedicated section in [routing](../routing/providers/marathon.md).

## Provider Configuration

### `basic`

_Optional_

```toml tab="File (TOML)"
[providers.marathon.basic]
  httpBasicAuthUser = "foo"
  httpBasicPassword = "bar"
```

```yaml tab="File (YAML)"
providers:
  marathon:
    basic:
      httpBasicAuthUser: foo
      httpBasicPassword: bar
```

```bash tab="CLI"
--providers.marathon.basic.httpbasicauthuser=foo
--providers.marathon.basic.httpbasicpassword=bar
```

Enables Marathon basic authentication.

### `dcosToken`

_Optional_

```toml tab="File (TOML)"
[providers.marathon]
  dcosToken = "xxxxxx"
  # ...
```

```toml tab="File (YAML)"
providers:
  marathon:
    dcosToken: "xxxxxx"
    # ...
```

```bash tab="CLI"
--providers.marathon.dcosToken=xxxxxx
```

DCOSToken for DCOS environment.

If set, it overrides the Authorization header.

### `defaultRule`

_Optional, Default=```Host(`{{ normalize .Name }}`)```_

```toml tab="File (TOML)"
[providers.marathon]
  defaultRule = "Host(`{{ .Name }}.{{ index .Labels \"customLabel\"}}`)"
  # ...
```

```yaml tab="File (YAML)"
providers:
  marathon:
    defaultRule: "Host(`{{ .Name }}.{{ index .Labels \"customLabel\"}}`)"
    # ...
```

```bash tab="CLI"
--providers.marathon.defaultRule=Host(`{{ .Name }}.{{ index .Labels \"customLabel\"}}`)
# ...
```

For a given application if no routing rule was defined by a label, it is defined by this defaultRule instead.

It must be a valid [Go template](https://golang.org/pkg/text/template/),
augmented with the [sprig template functions](http://masterminds.github.io/sprig/).

The app ID can be accessed as the Name identifier,
and the template has access to all the labels defined on this Marathon application.

### `dialerTimeout`

_Optional, Default=5s_

```toml tab="File (TOML)"
[providers.marathon]
  dialerTimeout = "10s"
  # ...
```

```toml tab="File (YAML)"
providers:
  marathon:
    dialerTimeout: "10s"
    # ...
```

```bash tab="CLI"
--providers.marathon.dialerTimeout=10s
```

Overrides DialerTimeout.

Amount of time the Marathon provider should wait before timing out,
when trying to open a TCP connection to a Marathon master.

Can be provided in a format supported by [time.ParseDuration](https://golang.org/pkg/time/#ParseDuration),
or directly as a number of seconds.

### `endpoint`

_Optional, Default=http://127.0.0.1:8080_

```toml tab="File (TOML)"
[providers.marathon]
  endpoint = "http://10.241.1.71:8080,10.241.1.72:8080,10.241.1.73:8080"
  # ...
```

```toml tab="File (YAML)"
providers:
  marathon:
    endpoint: "http://10.241.1.71:8080,10.241.1.72:8080,10.241.1.73:8080"
    # ...
```

```bash tab="CLI"
--providers.marathon.endpoint=http://10.241.1.71:8080,10.241.1.72:8080,10.241.1.73:8080
```

Marathon server endpoint.

You can optionally specify multiple endpoints.

### `queryParameters`

_Optional, Default=```embed=apps.tasks&embed=apps.deployments&embed=apps.readiness```_

```toml tab="File (TOML)"
[providers.marathon]
  queryParameters = "label=namespace==traefik&embed=apps.tasks&embed=apps.deployments&embed=apps.readiness"
  # ...
```

```toml tab="File (YAML)"
providers:
  marathon:
    queryParameters: "label=namespace==traefik&embed=apps.tasks&embed=apps.deployments&embed=apps.readiness"
    # ...
```

```bash tab="CLI"
--providers.marathon.queryParameters="label=namespace==traefik&embed=apps.tasks&embed=apps.deployments&embed=apps.readiness"
```

Overrides QueryParameters. The query defines which applications will be pulled from Marathon api.

Used to filter the application list received from Marathon, i.e, the filtering is done by Marathon and not by traefik.

The query is written in [URL query string](https://en.wikipedia.org/wiki/Query_string) format.
Supported parameters described in Marathon `/v2/apps` GET endpoint [documentation](https://github.com/mesosphere/marathon/blob/54edfdcd65d6ed1da51bfde3e54e32b660095fc2/docs/docs/rest-api/public/api/v2/apps.raml#L9).

**Note**: Most of this functionality can be obtained by Traefik side filtering using [Constraints](./#constraints).
This option should be used if you wish to reduce the amount of data received from Marathon api. 

### `exposedByDefault`

_Optional, Default=true_

```toml tab="File (TOML)"
[providers.marathon]
  exposedByDefault = false
  # ...
```

```yaml tab="File (YAML)"
providers:
  marathon:
    exposedByDefault: false
    # ...
```

```bash tab="CLI"
--providers.marathon.exposedByDefault=false
# ...
```

Exposes Marathon applications by default through Traefik.

If set to false, applications that don't have a `traefik.enable=true` label will be ignored from the resulting routing configuration.

See also [Restrict the Scope of Service Discovery](./overview.md#restrict-the-scope-of-service-discovery).

### `constraints`

_Optional, Default=""_

```toml tab="File (TOML)"
[providers.marathon]
  constraints = "Label(`a.label.name`,`foo`)"
  # ...
```

```yaml tab="File (YAML)"
providers:
  marathon:
    constraints: "Label(`a.label.name`,`foo`)"
    # ...
```

```bash tab="CLI"
--providers.marathon.constraints=Label(`a.label.name`,`foo`)
# ...
```

Constraints is an expression that Traefik matches against the application's labels to determine whether to create any route for that application.
That is to say, if none of the application's labels match the expression, no route for the application is created.
In addition, the expression also matched against the application's constraints, such as described in [Marathon constraints](https://mesosphere.github.io/marathon/docs/constraints.html).
If the expression is empty, all detected applications are included.

The expression syntax is based on the `Label("key", "value")`, and `LabelRegex("key", "value")`, as well as the usual boolean logic.
In addition, to match against marathon constraints, the function `MarathonConstraint("field:operator:value")` can be used, where the field, operator, and value parts are joined together in a single string with the `:` separator.

??? example "Constraints Expression Examples"

    ```toml
    # Includes only applications having a label with key `a.label.name` and value `foo`
    constraints = "Label(`a.label.name`, `foo`)"
    ```
    
    ```toml
    # Excludes applications having any label with key `a.label.name` and value `foo`
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
    # Includes only applications having a label with key `a.label.name` and a value matching the `a.+` regular expression.
    constraints = "LabelRegex(`a.label.name`, `a.+`)"
    ```

    ```toml
    # Includes only applications having a Marathon constraint with field `A`, operator `B`, and value `C`.
    constraints = "MarathonConstraint(`A:B:C`)"
    ```

    ```toml
    # Uses both Marathon constraint and application label with logical operator.
    constraints = "MarathonConstraint(`A:B:C`) && Label(`a.label.name`, `value`)"
    ```

See also [Restrict the Scope of Service Discovery](./overview.md#restrict-the-scope-of-service-discovery).

### `forceTaskHostname`

_Optional, Default=false_

```toml tab="File (TOML)"
[providers.marathon]
  forceTaskHostname = true
  # ...
```

```yaml tab="File (YAML)"
providers:
  marathon:
    forceTaskHostname: true
    # ...
```

```bash tab="CLI"
--providers.marathon.forceTaskHostname=true
# ...
```

By default, a task's IP address (as returned by the Marathon API) is used as backend server if an IP-per-task configuration can be found;
otherwise, the name of the host running the task is used.
The latter behavior can be enforced by enabling this switch.

### `keepAlive`

_Optional, Default=10s_

```toml tab="File (TOML)"
[providers.marathon]
  keepAlive = "30s"
  # ...
```

```yaml tab="File (YAML)"
providers:
  marathon:
    keepAlive: "30s"
    # ...
```

```bash tab="CLI"
--providers.marathon.keepAlive=30s
# ...
```

Set the TCP Keep Alive interval for the Marathon HTTP Client.
Can be provided in a format supported by [time.ParseDuration](https://golang.org/pkg/time/#ParseDuration),
or directly as a number of seconds.

### `respectReadinessChecks`

_Optional, Default=false_

```toml tab="File (TOML)"
[providers.marathon]
  respectReadinessChecks = true
  # ...
```

```yaml tab="File (YAML)"
providers:
  marathon:
    respectReadinessChecks: true
    # ...
```

```bash tab="CLI"
--providers.marathon.respectReadinessChecks=true
# ...
```

Applications may define readiness checks which are probed by Marathon during deployments periodically, and these check results are exposed via the API.
Enabling respectReadinessChecks causes Traefik to filter out tasks whose readiness checks have not succeeded.
Note that the checks are only valid at deployment times.

See the Marathon guide for details.

### `responseHeaderTimeout`

_Optional, Default=60s_

```toml tab="File (TOML)"
[providers.marathon]
  responseHeaderTimeout = "66s"
  # ...
```

```yaml tab="File (YAML)"
providers:
  marathon:
    responseHeaderTimeout: "66s"
    # ...
```

```bash tab="CLI"
--providers.marathon.responseHeaderTimeout=66s
# ...
```

Overrides ResponseHeaderTimeout.
Amount of time the Marathon provider should wait before timing out,
when waiting for the first response header from a Marathon master.

Can be provided in a format supported by [time.ParseDuration](https://golang.org/pkg/time/#ParseDuration), or directly as a number of seconds.

### `tls`

_Optional_

#### `tls.ca`

Certificate Authority used for the secured connection to Marathon.

```toml tab="File (TOML)"
[providers.marathon.tls]
  ca = "path/to/ca.crt"
```

```yaml tab="File (YAML)"
providers:
  marathon:
    tls:
      ca: path/to/ca.crt
```

```bash tab="CLI"
--providers.marathon.tls.ca=path/to/ca.crt
```

#### `tls.caOptional`

Policy followed for the secured connection to Marathon with TLS Client Authentication.
Requires `tls.ca` to be defined.

- `true`: VerifyClientCertIfGiven
- `false`: RequireAndVerifyClientCert
- if `tls.ca` is undefined NoClientCert

```toml tab="File (TOML)"
[providers.marathon.tls]
  caOptional = true
```

```yaml tab="File (YAML)"
providers:
  marathon:
    tls:
      caOptional: true
```

```bash tab="CLI"
--providers.marathon.tls.caOptional=true
```

#### `tls.cert`

Public certificate used for the secured connection to Marathon.

```toml tab="File (TOML)"
[providers.marathon.tls]
  cert = "path/to/foo.cert"
  key = "path/to/foo.key"
```

```yaml tab="File (YAML)"
providers:
  marathon:
    tls:
      cert: path/to/foo.cert
      key: path/to/foo.key
```

```bash tab="CLI"
--providers.marathon.tls.cert=path/to/foo.cert
--providers.marathon.tls.key=path/to/foo.key
```

#### `tls.key`

Private certificate used for the secured connection to Marathon.

```toml tab="File (TOML)"
[providers.marathon.tls]
  cert = "path/to/foo.cert"
  key = "path/to/foo.key"
```

```yaml tab="File (YAML)"
providers:
  marathon:
    tls:
      cert: path/to/foo.cert
      key: path/to/foo.key
```

```bash tab="CLI"
--providers.marathon.tls.cert=path/to/foo.cert
--providers.marathon.tls.key=path/to/foo.key
```

#### `tls.insecureSkipVerify`

If `insecureSkipVerify` is `true`, TLS for the connection to Marathon accepts any certificate presented by the server and any host name in that certificate.

```toml tab="File (TOML)"
[providers.marathon.tls]
  insecureSkipVerify = true
```

```yaml tab="File (YAML)"
providers:
  marathon:
    tls:
      insecureSkipVerify: true
```

```bash tab="CLI"
--providers.marathon.tls.insecureSkipVerify=true
```

### `tlsHandshakeTimeout`

_Optional, Default=5s_

```toml tab="File (TOML)"
[providers.marathon]
  responseHeaderTimeout = "10s"
  # ...
```

```yaml tab="File (YAML)"
providers:
  marathon:
    responseHeaderTimeout: "10s"
    # ...
```

```bash tab="CLI"
--providers.marathon.responseHeaderTimeout=10s
# ...
```

Overrides TLSHandshakeTimeout.

Amount of time the Marathon provider should wait before timing out,
when waiting for the TLS handshake to complete.
Can be provided in a format supported by [time.ParseDuration](https://golang.org/pkg/time/#ParseDuration),
or directly as a number of seconds.

### `trace`

_Optional, Default=false_

```toml tab="File (TOML)"
[providers.marathon]
  trace = true
  # ...
```

```yaml tab="File (YAML)"
providers:
  marathon:
    trace: true
    # ...
```

```bash tab="CLI"
--providers.marathon.trace=true
# ...
```

Displays additional provider logs (if available).

### `watch`

_Optional, Default=true_

```toml tab="File (TOML)"
[providers.marathon]
  watch = false
  # ...
```

```yaml tab="File (YAML)"
providers:
  marathon:
    watch: false
    # ...
```

```bash tab="CLI"
--providers.marathon.watch=false
# ...
```

Enables watching for Marathon changes.
