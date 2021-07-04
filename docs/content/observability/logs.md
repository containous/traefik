# Logs

Reading What's Happening
{: .subtitle }

By default, logs are written to stdout, in text format.

## Configuration

### General

Traefik logs concern everything that happens to Traefik itself (startup, configuration, events, shutdown, and so on).

#### `filePath`

By default, the logs are written to the standard output.
You can configure a file path instead using the `filePath` option.

```yaml tab="File (YAML)"
# Writing Logs to a File
log:
  filePath: "/path/to/traefik.log"
```

```toml tab="File (TOML)"
# Writing Logs to a File
[log]
  filePath = "/path/to/traefik.log"
```

```bash tab="CLI"
# Writing Logs to a File
--log.filePath=/path/to/traefik.log
```

#### `format`

By default, the logs use a text format (`common`), but you can also ask for the `json` format in the `format` option.

```yaml tab="File (YAML)"
# Writing Logs to a File, in JSON
log:
  filePath: "/path/to/log-file.log"
  format: json
```

```toml tab="File (TOML)"
# Writing Logs to a File, in JSON
[log]
  filePath = "/path/to/log-file.log"
  format = "json"
```

```bash tab="CLI"
# Writing Logs to a File, in JSON
--log.filePath=/path/to/traefik.log
--log.format=json
```

#### `level`

By default, the `level` is set to `ERROR`. Alternative logging levels are `DEBUG`, `PANIC`, `FATAL`, `ERROR`, `WARN`, and `INFO`.

```yaml tab="File (YAML)"
log:
  level: DEBUG
```

```toml tab="File (TOML)"
[log]
  level = "DEBUG"
```

```bash tab="CLI"
--log.level=DEBUG
```

## Log Rotation

Traefik will close and reopen its log files, assuming they're configured, on receipt of a USR1 signal.
This allows the logs to be rotated and processed by an external program, such as `logrotate`.

!!! warning
    This does not work on Windows due to the lack of USR signals.
