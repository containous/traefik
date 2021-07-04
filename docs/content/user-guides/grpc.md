# gRPC Examples

## With HTTP (h2c)

This section explains how to use Traefik as reverse proxy for gRPC application.

### Traefik Configuration

Static configuration:

```yaml tab="File (YAML)"
entryPoints:
  web:
    address: :80

providers:
  file:
    directory: /path/to/dynamic/config

api: {}
```

```toml tab="File (TOML)"
[entryPoints]
  [entryPoints.web]
    address = ":80"

[api]

[providers.file]
  directory = "/path/to/dynamic/config"
```

```yaml tab="CLI"
--entryPoints.web.address=:80
--providers.file.directory=/path/to/dynamic/config
--api.insecure=true
```

`/path/to/dynamic/config/dynamic_conf.{yml,toml}`:

```yaml tab="YAML"
## dynamic configuration ##

http:
  routers:
    routerTest:
      service: srv-grpc
      rule: Host(`frontend.local`)

  services:
    srv-grpc:
      loadBalancer:
        servers:
        - url: h2c://backend.local:8080
```

```toml tab="TOML"
## dynamic configuration ##

[http]

  [http.routers]
    [http.routers.routerTest]
      service = "srv-grpc"
      rule = "Host(`frontend.local`)"

  [http.services]
    [http.services.srv-grpc]
      [http.services.srv-grpc.loadBalancer]
        [[http.services.srv-grpc.loadBalancer.servers]]
          url = "h2c://backend.local:8080"
```

!!! warning
    For providers with labels, you will have to specify the `traefik.http.services.<my-service-name>.loadbalancer.server.scheme=h2c`

### Conclusion

We don't need specific configuration to use gRPC in Traefik, we just need to use `h2c` protocol, or use HTTPS communications to have HTTP2 with the backend.

## With HTTPS

This section explains how to use Traefik as reverse proxy for gRPC application with self-signed certificates.

![gRPC architecture](../assets/img/user-guides/grpc.svg)

### gRPC Server Certificate

In order to secure the gRPC server, we generate a self-signed certificate for service url:

```bash
openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout ./backend.key -out ./backend.cert
```

That will prompt for information, the important answer is:

```txt
Common Name (e.g. server FQDN or YOUR name) []: backend.local
```

### gRPC Client Certificate

Generate your self-signed certificate for router url:

```bash
openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout ./frontend.key -out ./frontend.cert
```

with

```txt
Common Name (e.g. server FQDN or YOUR name) []: frontend.local
```

### Traefik Configuration

At last, we configure our Traefik instance to use both self-signed certificates.

Static configuration:

```yaml tab="File (YAML)"
entryPoints:
  websecure:
    address: :4443

serversTransport:
  # For secure connection on backend.local
  rootCAs:
    - ./backend.cert

providers:
  file:
    directory: /path/to/dynamic/config

api: {}
```

```toml tab="File (TOML)"
[entryPoints]
  [entryPoints.websecure]
    address = ":4443"


[serversTransport]
  # For secure connection on backend.local
  rootCAs = [ "./backend.cert" ]

[api]

[provider.file]
  directory = "/path/to/dynamic/config"
```

```yaml tab="CLI"
--entryPoints.websecure.address=:4443
# For secure connection on backend.local
--serversTransport.rootCAs=./backend.cert
--providers.file.directory=/path/to/dynamic/config
--api.insecure=true
```

`/path/to/dynamic/config/dynamic_conf.{yml,toml}`:

```yaml tab="YAML"
## dynamic configuration ##

http:
  routers:
    routerTest:
      service: srv-grpc
      rule: Host(`frontend.local`)
  services:
    srv-grpc:
      loadBalancer:
        servers:
        # Access on backend with HTTPS
        - url: https://backend.local:8080
tls:
  # For secure connection on frontend.local
  certificates:
  - certfile: ./frontend.cert
    keyfile: ./frontend.key
```

```toml tab="TOML"
## dynamic configuration ##

[http]

  [http.routers]
    [http.routers.routerTest]
      service = "srv-grpc"
      rule = "Host(`frontend.local`)"

  [http.services]
    [http.services.srv-grpc]
      [http.services.srv-grpc.loadBalancer]
        [[http.services.srv-grpc.loadBalancer.servers]]
          # Access on backend with HTTPS
          url = "https://backend.local:8080"

[tls]

  # For secure connection on frontend.local
  [[tls.certificates]]
    certFile = "./frontend.cert"
    keyFile = "./frontend.key"
```

!!! warning
    With some services, the server URLs use the IP, so you may need to configure `insecureSkipVerify` instead of the `rootCAs` to activate HTTPS without hostname verification.

### A gRPC example in go (modify for https)

We use the gRPC greeter example in [grpc-go](https://github.com/grpc/grpc-go/tree/master/examples/helloworld)

!!! warning
    In order to use this gRPC example, we need to modify it to use HTTPS

So we modify the "gRPC server example" to use our own self-signed certificate:

```go
// ...

// Read cert and key file
backendCert, _ := ioutil.ReadFile("./backend.cert")
backendKey, _ := ioutil.ReadFile("./backend.key")

// Generate Certificate struct
cert, err := tls.X509KeyPair(backendCert, backendKey)
if err != nil {
  log.Fatalf("failed to parse certificate: %v", err)
}

// Create credentials
creds := credentials.NewServerTLSFromCert(&cert)

// Use Credentials in gRPC server options
serverOption := grpc.Creds(creds)
var s *grpc.Server = grpc.NewServer(serverOption)
defer s.Stop()

pb.RegisterGreeterServer(s, &server{})
err := s.Serve(lis)

// ...
```

Next we will modify gRPC Client to use our Traefik self-signed certificate:

```go
// ...

// Read cert file
frontendCert, _ := ioutil.ReadFile("./frontend.cert")

// Create CertPool
roots := x509.NewCertPool()
roots.AppendCertsFromPEM(frontendCert)

// Create credentials
credsClient := credentials.NewClientTLSFromCert(roots, "")

// Dial with specific Transport (with credentials)
conn, err := grpc.Dial("frontend.local:4443", grpc.WithTransportCredentials(credsClient))
if err != nil {
    log.Fatalf("did not connect: %v", err)
}

defer conn.Close()
client := pb.NewGreeterClient(conn)

name := "World"
r, err := client.SayHello(context.Background(), &pb.HelloRequest{Name: name})

// ...
```
