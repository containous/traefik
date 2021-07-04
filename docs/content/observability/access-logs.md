# Access Logs

Who Calls Whom?
{.subtitle}

By default, logs are written to stdout, in text format.

## Configuration

To enable the access logs:

```yaml tab="File (YAML)"
accessLog: {}
```

```toml tab="File (TOML)"
[accessLog]
```

```bash tab="CLI"
--accesslog=true
```

### `filePath`

By default access logs are written to the standard output.
To write the logs into a log file, use the `filePath` option.

```yaml tab="File (YAML)"
accessLog:
  filePath: "/path/to/access.log"
```

```toml tab="File (TOML)"
[accessLog]
  filePath = "/path/to/access.log"
```

```bash tab="CLI"
--accesslog.filepath=/path/to/access.log
```

### `format`

By default, logs are written using the Common Log Format (CLF).
To write logs in JSON, use `json` in the `format` option.
If the given format is unsupported, the default (CLF) is used instead.

!!! info "Common Log Format"

    ```html
    <remote_IP_address> - <client_user_name_if_available> [<timestamp>] "<request_method> <request_path> <request_protocol>" <origin_server_HTTP_status> <origin_server_content_size> "<request_referrer>" "<request_user_agent>" <number_of_requests_received_since_Traefik_started> "<Traefik_router_name>" "<Traefik_server_URL>" <request_duration_in_ms>ms
    ```

### `bufferingSize`

To write the logs in an asynchronous fashion, specify a  `bufferingSize` option.
This option represents the number of log lines Traefik will keep in memory before writing them to the selected output.
In some cases, this option can greatly help performances.

```yaml tab="File (YAML)"
# Configuring a buffer of 100 lines
accessLog:
  filePath: "/path/to/access.log"
  bufferingSize: 100
```

```toml tab="File (TOML)"
# Configuring a buffer of 100 lines
[accessLog]
  filePath = "/path/to/access.log"
  bufferingSize = 100
```

```bash tab="CLI"
# Configuring a buffer of 100 lines
--accesslog.filepath=/path/to/access.log
--accesslog.bufferingsize=100
```

### Filtering

To filter logs, you can specify a set of filters which are logically "OR-connected".
Thus, specifying multiple filters will keep more access logs than specifying only one.

The available filters are:

- `statusCodes`, to limit the access logs to requests with a status codes in the specified range
- `retryAttempts`, to keep the access logs when at least one retry has happened
- `minDuration`, to keep access logs when requests take longer than the specified duration (provided in seconds or as a valid duration format, see [time.ParseDuration](https://golang.org/pkg/time/#ParseDuration))

```yaml tab="File (YAML)"
# Configuring Multiple Filters
accessLog:
  filePath: "/path/to/access.log"
  format: json
  filters:
    statusCodes:
      - "200"
      - "300-302"
    retryAttempts: true
    minDuration: "10ms"
```

```toml tab="File (TOML)"
# Configuring Multiple Filters
[accessLog]
  filePath = "/path/to/access.log"
  format = "json"

  [accessLog.filters]
    statusCodes = ["200", "300-302"]
    retryAttempts = true
    minDuration = "10ms"
```

```bash tab="CLI"
# Configuring Multiple Filters
--accesslog.filepath=/path/to/access.log
--accesslog.format=json
--accesslog.filters.statuscodes=200,300-302
--accesslog.filters.retryattempts
--accesslog.filters.minduration=10ms
```

### Limiting the Fields/Including Headers

You can decide to limit the logged fields/headers to a given list with the `fields.names` and `fields.headers` options.

Each field can be set to:

- `keep` to keep the value
- `drop` to drop the value
- `redact` to replace the value with "redacted"

The `defaultMode` for `fields.headers` is `drop`.

  [accessLog.fields]
    defaultMode = "keep"

```yaml tab="File (YAML)"
# Limiting the Logs to Specific Fields
accessLog:
  filePath: "/path/to/access.log"
  format: json
  fields:
    defaultMode: keep
    names:
      ClientUsername: drop
    headers:
      defaultMode: keep
      names:
          User-Agent: redact
          Authorization: drop
          Content-Type: keep
```

```toml tab="File (TOML)"
# Limiting the Logs to Specific Fields
[accessLog]
  filePath = "/path/to/access.log"
  format = "json"

    [accessLog.fields.names]
      "ClientUsername" = "drop"

    [accessLog.fields.headers]
      defaultMode = "keep"

      [accessLog.fields.headers.names]
        "User-Agent" = "redact"
        "Authorization" = "drop"
        "Content-Type" = "keep"
```

```bash tab="CLI"
# Limiting the Logs to Specific Fields
--accesslog.filepath=/path/to/access.log
--accesslog.format=json
--accesslog.fields.defaultmode=keep
--accesslog.fields.names.ClientUsername=drop
--accesslog.fields.headers.defaultmode=keep
--accesslog.fields.headers.names.User-Agent=redact
--accesslog.fields.headers.names.Authorization=drop
--accesslog.fields.headers.names.Content-Type=keep
```

??? info "Available Fields"

    | Field                   | Description                                                                                                                                                         |
    |-------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------|
    | `StartUTC`              | The time at which request processing started.                                                                                                                       |
    | `StartLocal`            | The local time at which request processing started.                                                                                                                 |
    | `Duration`              | The total time taken (in nanoseconds) by processing the response, including the origin server's time but not the log writing time.                                  |
    | `RouterName`            | The name of the Traefik  router.                                                                                                                                    |
    | `ServiceName`           | The name of the Traefik backend.                                                                                                                                    |
    | `ServiceURL`            | The URL of the Traefik backend.                                                                                                                                     |
    | `ServiceAddr`           | The IP:port of the Traefik backend (extracted from `ServiceURL`)                                                                                                    |
    | `ClientAddr`            | The remote address in its original form (usually IP:port).                                                                                                          |
    | `ClientHost`            | The remote IP address from which the client request was received.                                                                                                   |
    | `ClientPort`            | The remote TCP port from which the client request was received.                                                                                                     |
    | `ClientUsername`        | The username provided in the URL, if present.                                                                                                                       |
    | `RequestAddr`           | The HTTP Host header (usually IP:port). This is treated as not a header by the Go API.                                                                              |
    | `RequestHost`           | The HTTP Host server name (not including port).                                                                                                                     |
    | `RequestPort`           | The TCP port from the HTTP Host.                                                                                                                                    |
    | `RequestMethod`         | The HTTP method.                                                                                                                                                    |
    | `RequestPath`           | The HTTP request URI, not including the scheme, host or port.                                                                                                       |
    | `RequestProtocol`       | The version of HTTP requested.                                                                                                                                      |
    | `RequestScheme`         | The HTTP scheme requested `http` or `https`.                                                                                                                        |
    | `RequestLine`           | `RequestMethod` + `RequestPath` + `RequestProtocol`                                                                                                                 |
    | `RequestContentSize`    | The number of bytes in the request entity (a.k.a. body) sent by the client.                                                                                         |
    | `OriginDuration`        | The time taken (in nanoseconds) by the origin server ('upstream') to return its response.                                                                           |
    | `OriginContentSize`     | The content length specified by the origin server, or 0 if unspecified.                                                                                             |
    | `OriginStatus`          | The HTTP status code returned by the origin server. If the request was handled by this Traefik instance (e.g. with a redirect), then this value will be absent.     |
    | `OriginStatusLine`      | `OriginStatus` + Status code explanation                                                                                                                            |
    | `DownstreamStatus`      | The HTTP status code returned to the client.                                                                                                                        |
    | `DownstreamStatusLine`  | `DownstreamStatus` + Status code explanation                                                                                                                        |
    | `DownstreamContentSize` | The number of bytes in the response entity returned to the client. This is in addition to the "Content-Length" header, which may be present in the origin response. |
    | `RequestCount`          | The number of requests received since the Traefik instance started.                                                                                                 |
    | `GzipRatio`             | The response body compression ratio achieved.                                                                                                                       |
    | `Overhead`              | The processing time overhead (in nanoseconds) caused by Traefik.                                                                                                    |
    | `RetryAttempts`         | The amount of attempts the request was retried.                                                                                                                     |
    | `TLSVersion`            | The TLS version used by the connection (e.g. `1.2`) (if connection is TLS).                                                                                         |
    | `TLSCipher`             | The TLS cipher used by the connection (e.g. `TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA`) (if connection is TLS)                                                           |

## Log Rotation

Traefik will close and reopen its log files, assuming they're configured, on receipt of a USR1 signal.
This allows the logs to be rotated and processed by an external program, such as `logrotate`.

!!! warning
    This does not work on Windows due to the lack of USR signals.

## Time Zones

Traefik will timestamp each log line in UTC time by default.

It is possible to configure the Traefik to timestamp in a specific timezone by ensuring the following configuration has been made in your environment:

1. Provide time zone data to `/etc/localtime` or `/usr/share/zoneinfo` (based on your distribution) or set the environment variable TZ to the desired timezone
2. Specify the field `StartLocal` by dropping the field named `StartUTC` (available on the default Common Log Format (CLF) as well as JSON)

Example utilizing Docker Compose:

```yaml
version: "3.7"

services:
  traefik:
    image: traefik:v2.2
    environment:
      - TZ=US/Alaska
    command:
      - --accesslog.fields.names.StartUTC=drop
      - --providers.docker
    ports:
      - 80:80
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
```
