# Traefik & AWS ECS

A Story of Labels & Elastic Containers
{: .subtitle }

Attach labels to your ECS containers and let Traefik do the rest!

## Configuration Examples

??? example "Configuring ECS provider"

    Enabling the ECS provider:

    ```yaml tab="File (YAML)"
    providers:
      ecs: {}
    ```

    ```toml tab="File (TOML)"
    [providers.ecs]
    ```

    ```bash tab="CLI"
    --providers.ecs=true
    ```

## Policy

Traefik needs the following policy to read ECS information:

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "TraefikECSReadAccess",
            "Effect": "Allow",
            "Action": [
                "ecs:ListClusters",
                "ecs:DescribeClusters",
                "ecs:ListTasks",
                "ecs:DescribeTasks",
                "ecs:DescribeContainerInstances",
                "ecs:DescribeTaskDefinition",
                "ec2:DescribeInstances"
            ],
            "Resource": [
                "*"
            ]
        }
    ]
}
```

## Provider Configuration

### `autoDiscoverClusters`

_Optional, Default=false_

Search for services in cluster list.

- If set to `true` service discovery is disabled on configured clusters, but enabled for all other clusters.
- If set to `false` service discovery is enabled on configured clusters only.

```yaml tab="File (YAML)"
providers:
  ecs:
    autoDiscoverClusters: true
    # ...
```

```toml tab="File (TOML)"
[providers.ecs]
  autoDiscoverClusters = true
  # ...
```

```bash tab="CLI"
--providers.ecs.autoDiscoverClusters=true
# ...
```

### `clusters`

_Optional, Default=["default"]_

Search for services in cluster list.

```yaml tab="File (YAML)"
providers:
  ecs:
    clusters:
      - default
    # ...
```

```toml tab="File (TOML)"
[providers.ecs]
  clusters = ["default"]
  # ...
```

```bash tab="CLI"
--providers.ecs.clusters=default
# ...
```

### `exposedByDefault`

_Optional, Default=true_

Expose ECS services by default in Traefik.

If set to `false`, services that do not have a `traefik.enable=true` label are ignored from the resulting routing configuration.

```yaml tab="File (YAML)"
providers:
  ecs:
    exposedByDefault: false
    # ...
```

```toml tab="File (TOML)"
[providers.ecs]
  exposedByDefault = false
  # ...
```

```bash tab="CLI"
--providers.ecs.exposedByDefault=false
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
  ecs:
    defaultRule: "Host(`{{ .Name }}.{{ index .Labels \"customLabel\"}}`)"
    # ...
```

```toml tab="File (TOML)"
[providers.ecs]
  defaultRule = "Host(`{{ .Name }}.{{ index .Labels \"customLabel\"}}`)"
  # ...
```

```bash tab="CLI"
--providers.ecs.defaultRule=Host(`{{ .Name }}.{{ index .Labels \"customLabel\"}}`)
# ...
```

### `refreshSeconds`

_Optional, Default=15_

Polling interval (in seconds).

```yaml tab="File (YAML)"
providers:
  ecs:
    refreshSeconds: 15
    # ...
```

```toml tab="File (TOML)"
[providers.ecs]
  refreshSeconds = 15
  # ...
```

```bash tab="CLI"
--providers.ecs.refreshSeconds=15
# ...
```

### Credentials

_Optional_

If `region` is not provided, it is resolved from the EC2 metadata endpoint for EC2 tasks.
In a FARGATE context it is resolved from the `AWS_REGION` environment variable.

If `accessKeyID` and `secretAccessKey` are not provided, credentials are resolved in the following order:

- Using the environment variables `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, and `AWS_SESSION_TOKEN`.
- Using shared credentials, determined by `AWS_PROFILE` and `AWS_SHARED_CREDENTIALS_FILE`, defaults to `default` and `~/.aws/credentials`.
- Using EC2 instance role or ECS task role

```yaml tab="File (YAML)"
providers:
  ecs:
    region: us-east-1
    accessKeyID: "abc"
    secretAccessKey: "123"
    # ...
```

```toml tab="File (TOML)"
[providers.ecs]
  region = "us-east-1"
  accessKeyID = "abc"
  secretAccessKey = "123"
```

```bash tab="CLI"
--providers.ecs.region="us-east-1"
--providers.ecs.accessKeyID="abc"
--providers.ecs.secretAccessKey="123"
# ...
```
