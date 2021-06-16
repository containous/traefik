# RedirectRegex

Redirecting the Client to a Different Location
{: .subtitle }

<!--
TODO: add schema
-->

The RedirectRegex redirects a request using regex matching and replacement.

## Configuration Examples

```yaml tab="Docker"
# Redirect with domain replacement
# Note: all dollar signs need to be doubled for escaping.
labels:
  - "traefik.http.middlewares.test-redirectregex.redirectregex.regex=^http://localhost/(.*)"
  - "traefik.http.middlewares.test-redirectregex.redirectregex.replacement=http://mydomain/$${1}"
```

```yaml tab="Kubernetes"
# Redirect with domain replacement
apiVersion: traefik.containo.us/v1alpha1
kind: Middleware
metadata:
  name: test-redirectregex
spec:
  redirectRegex:
    regex: ^http://localhost/(.*)
    replacement: http://mydomain/${1}
```

```yaml tab="Consul Catalog"
# Redirect with domain replacement
# Note: all dollar signs need to be doubled for escaping.
- "traefik.http.middlewares.test-redirectregex.redirectregex.regex=^http://localhost/(.*)"
- "traefik.http.middlewares.test-redirectregex.redirectregex.replacement=http://mydomain/$${1}"
```

```json tab="Marathon"
"labels": {
  "traefik.http.middlewares.test-redirectregex.redirectregex.regex": "^http://localhost/(.*)",
  "traefik.http.middlewares.test-redirectregex.redirectregex.replacement": "http://mydomain/${1}"
}
```

```yaml tab="Rancher"
# Redirect with domain replacement
# Note: all dollar signs need to be doubled for escaping.
labels:
  - "traefik.http.middlewares.test-redirectregex.redirectregex.regex=^http://localhost/(.*)"
  - "traefik.http.middlewares.test-redirectregex.redirectregex.replacement=http://mydomain/$${1}"
```

```toml tab="File (TOML)"
# Redirect with domain replacement
[http.middlewares]
  [http.middlewares.test-redirectregex.redirectRegex]
    regex = "^http://localhost/(.*)"
    replacement = "http://mydomain/${1}"
```

```yaml tab="File (YAML)"
# Redirect with domain replacement
http:
  middlewares:
    test-redirectregex:
      redirectRegex:
        regex: "^http://localhost/(.*)"
        replacement: "http://mydomain/${1}"
```

## Configuration Options

!!! tip

    Regular expressions and replacements can be tested using online tools such as [Go Playground](https://play.golang.org/p/mWU9p-wk2ru) or the [Regex101](https://regex101.com/r/58sIgx/2).

### `permanent`

Set the `permanent` option to `true` to apply a permanent redirection.

### `regex`

The `regex` option is the regular expression to match and capture elements from the request URL.
    
### `replacement`

The `replacement` option defines how to modify the URL to have the new target URL.

!!! warning

    Care should be taken when defining replacement expand variables: `$1x` is equivalent to `${1x}`, not `${1}x` (see [Regexp.Expand](https://golang.org/pkg/regexp/#Regexp.Expand)), so use `${1}` syntax.
