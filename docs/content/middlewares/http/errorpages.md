# ErrorPage

It Has Never Been Easier to Say That Something Went Wrong
{: .subtitle }

![ErrorPages](../../assets/img/middleware/errorpages.png)

The ErrorPage middleware returns a custom page in lieu of the default, according to configured ranges of HTTP Status codes.

!!! important
    The error page itself is _not_ hosted by Traefik.

## Configuration Examples

```yaml tab="Docker"
# Dynamic Custom Error Page for 5XX Status Code
labels:
  - "traefik.http.middlewares.test-errorpage.errors.status=500-599"
  - "traefik.http.middlewares.test-errorpage.errors.service=serviceError"
  - "traefik.http.middlewares.test-errorpage.errors.query=/{status}.html"
```

```yaml tab="Kubernetes"
apiVersion: traefik.containo.us/v1alpha1
kind: Middleware
metadata:
  name: test-errorpage
spec:
  errors:
    status:
      - "500-599"
    query: /{status}.html
    service:
      name: whoami
      port: 80
```

```yaml tab="Consul Catalog"
# Dynamic Custom Error Page for 5XX Status Code
- "traefik.http.middlewares.test-errorpage.errors.status=500-599"
- "traefik.http.middlewares.test-errorpage.errors.service=serviceError"
- "traefik.http.middlewares.test-errorpage.errors.query=/{status}.html"
```

```json tab="Marathon"
"labels": {
  "traefik.http.middlewares.test-errorpage.errors.status": "500-599",
  "traefik.http.middlewares.test-errorpage.errors.service": "serviceError",
  "traefik.http.middlewares.test-errorpage.errors.query": "/{status}.html"
}
```

```yaml tab="Rancher"
# Dynamic Custom Error Page for 5XX Status Code
labels:
  - "traefik.http.middlewares.test-errorpage.errors.status=500-599"
  - "traefik.http.middlewares.test-errorpage.errors.service=serviceError"
  - "traefik.http.middlewares.test-errorpage.errors.query=/{status}.html"
```

```yaml tab="File (YAML)"
# Custom Error Page for 5XX
http:
  middlewares:
    test-errorpage:
      errors:
        status:
          - "500-599"
        service: serviceError
        query: "/{status}.html"

  services:
    # ... definition of error-handler-service and my-service
```

```toml tab="File (TOML)"
# Custom Error Page for 5XX
[http.middlewares]
  [http.middlewares.test-errorpage.errors]
    status = ["500-599"]
    service = "serviceError"
    query = "/{status}.html"

[http.services]
  # ... definition of error-handler-service and my-service
```

!!! note ""

    In this example, the error page URL is based on the status code (`query=/{status}.html`).

## Configuration Options

### `status`

The `status` option defines which status or range of statuses should result in an error page.

The status code ranges are inclusive (`500-599` will trigger with every code between `500` and `599`, `500` and `599` included).

!!! note ""

    You can define either a status code as a number (`500`),
    as multiple comma-separated numbers (`500,502`),
    as ranges by separating two codes with a dash (`500-599`),
    or a combination of the two (`404,418,500-599`).

### `service`

The service that will serve the new requested error page.

!!! note ""

    In Kubernetes, you need to reference a Kubernetes Service instead of a Traefik service.

### `query`

The URL for the error page (hosted by `service`). You can use the `{status}` variable in the `query` option in order to insert the status code in the URL.
