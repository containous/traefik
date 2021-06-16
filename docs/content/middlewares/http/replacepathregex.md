# ReplacePathRegex

Updating the Path Before Forwarding the Request (Using a Regex)
{: .subtitle }

<!--
TODO: add schema
-->

The ReplaceRegex replaces the path of a URL using regex matching and replacement.

## Configuration Examples

```yaml tab="Docker"
# Replace path with regex
labels:
  - "traefik.http.middlewares.test-replacepathregex.replacepathregex.regex=^/foo/(.*)"
  - "traefik.http.middlewares.test-replacepathregex.replacepathregex.replacement=/bar/$$1"
```

```yaml tab="Kubernetes"
# Replace path with regex
apiVersion: traefik.containo.us/v1alpha1
kind: Middleware
metadata:
  name: test-replacepathregex
spec:
  replacePathRegex:
    regex: ^/foo/(.*)
    replacement: /bar/$1
```

```yaml tab="Consul Catalog"
# Replace path with regex
- "traefik.http.middlewares.test-replacepathregex.replacepathregex.regex=^/foo/(.*)"
- "traefik.http.middlewares.test-replacepathregex.replacepathregex.replacement=/bar/$1"
```

```json tab="Marathon"
"labels": {
  "traefik.http.middlewares.test-replacepathregex.replacepathregex.regex": "^/foo/(.*)",
  "traefik.http.middlewares.test-replacepathregex.replacepathregex.replacement": "/bar/$1"
}
```

```yaml tab="Rancher"
# Replace path with regex
labels:
  - "traefik.http.middlewares.test-replacepathregex.replacepathregex.regex=^/foo/(.*)"
  - "traefik.http.middlewares.test-replacepathregex.replacepathregex.replacement=/bar/$1"
```

```toml tab="File (TOML)"
# Replace path with regex
[http.middlewares]
  [http.middlewares.test-replacepathregex.replacePathRegex]
    regex = "^/foo/(.*)"
    replacement = "/bar/$1"
```

```yaml tab="File (YAML)"
# Replace path with regex
http:
  middlewares:
    test-replacepathregex:
      replacePathRegex:
        regex: "^/foo/(.*)"
        replacement: "/bar/$1"
```

## Configuration Options

### General

The ReplacePathRegex middleware will:

- replace the matching path with the specified one.
- store the original path in a `X-Replaced-Path` header.

!!! tip

    Regular expressions and replacements can be tested using online tools such as [Go Playground](https://play.golang.org/p/mWU9p-wk2ru) or [Regex101](https://regex101.com/r/58sIgx/2).

### `regex`

The `regex` option is the regular expression to match and capture the path from the request URL.
    
### `replacement`

The `replacement` option defines the replacement path format, which can include captured variables.

!!! warning

    Care should be taken when defining replacement expand variables: `$1x` is equivalent to `${1x}`, not `${1}x` (see [Regexp.Expand](https://golang.org/pkg/regexp/#Regexp.Expand)), so use `${1}` syntax.
