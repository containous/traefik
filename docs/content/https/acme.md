# Let's Encrypt

Automatic HTTPS
{: .subtitle }

You can configure Traefik to use an ACME provider (like Let's Encrypt) for automatic certificate generation.

!!! warning "Let's Encrypt and Rate Limiting"
    Note that Let's Encrypt API has [rate limiting](https://letsencrypt.org/docs/rate-limits).

    Use Let's Encrypt staging server with the [`caServer`](#caserver) configuration option
    when experimenting to avoid hitting this limit too fast.

## Certificate Resolvers

Traefik requires you to define "Certificate Resolvers" in the [static configuration](../getting-started/configuration-overview.md#the-static-configuration), 
which are responsible for retrieving certificates from an ACME server.

Then, each ["router"](../routing/routers/index.md) is configured to enable TLS,
and is associated to a certificate resolver through the [`tls.certresolver` configuration option](../routing/routers/index.md#certresolver).

Certificates are requested for domain names retrieved from the router's [dynamic configuration](../getting-started/configuration-overview.md#the-dynamic-configuration).

You can read more about this retrieval mechanism in the following section: [ACME Domain Definition](#domain-definition).

!!! important "Defining a certificates resolver does not result in all routers automatically using it. Each router that is supposed to use the resolver must [reference](../routing/routers/index.md#certresolver) it."

??? note "Configuration Reference"
    
    There are many available options for ACME.
    For a quick glance at what's possible, browse the configuration reference:
    
    ```toml tab="File (TOML)"
    --8<-- "content/https/ref-acme.toml"
    ```
    
    ```yaml tab="File (YAML)"
    --8<-- "content/https/ref-acme.yaml"
    ```
    
    ```bash tab="CLI"
    --8<-- "content/https/ref-acme.txt"
    ```

## Domain Definition

Certificate resolvers request certificates for a set of the domain names 
inferred from routers, with the following logic:

- If the router has a [`tls.domains`](../routing/routers/index.md#domains) option set,
  then the certificate resolver uses the `main` (and optionally `sans`) option of `tls.domains` to know the domain names for this router.

- If no [`tls.domains`](../routing/routers/index.md#domains) option is set, 
  then the certificate resolver uses the [router's rule](../routing/routers/index.md#rule), 
  by checking the `Host()` matchers. 
  Please note that [multiple `Host()` matchers can be used](../routing/routers/index.md#certresolver)) for specifying multiple domain names for this router.

Please note that:

- When multiple domain names are inferred from a given router,
  only **one** certificate is requested with the first domain name as the main domain,
  and the other domains as ["SANs" (Subject Alternative Name)](https://en.wikipedia.org/wiki/Subject_Alternative_Name).

- As [ACME V2 supports "wildcard domains"](#wildcard-domains),
  any router can provide a [wildcard domain](https://en.wikipedia.org/wiki/Wildcard_certificate) name, as "main" domain or as "SAN" domain.

Please check the [configuration examples below](#configuration-examples) for more details.

## Configuration Examples

??? example "Enabling ACME"
    
    ```toml tab="File (TOML)"
    [entryPoints]
      [entryPoints.web]
        address = ":80"
    
      [entryPoints.websecure]
        address = ":443"
    
    [certificatesResolvers.myresolver.acme]
      email = "your-email@example.com"
      storage = "acme.json"
      [certificatesResolvers.myresolver.acme.httpChallenge]
        # used during the challenge
        entryPoint = "web"
    ```
    
    ```yaml tab="File (YAML)"
    entryPoints:
      web:
        address: ":80"
    
      websecure:
        address: ":443"
    
    certificatesResolvers:
      myresolver:
        acme:
          email: your-email@example.com
          storage: acme.json
          httpChallenge:
            # used during the challenge
            entryPoint: web
    ```
    
    ```bash tab="CLI"
    --entrypoints.web.address=:80
    --entrypoints.websecure.address=:443
    # ...
    --certificatesresolvers.myresolver.acme.email=your-email@example.com
    --certificatesresolvers.myresolver.acme.storage=acme.json
    # used during the challenge
    --certificatesresolvers.myresolver.acme.httpchallenge.entrypoint=web
    ```

!!! important "Defining a certificates resolver does not result in all routers automatically using it. Each router that is supposed to use the resolver must [reference](../routing/routers/index.md#certresolver) it."

??? example "Single Domain from Router's Rule Example"
    
    * A certificate for the domain `example.com` is requested:

    --8<-- "content/https/include-acme-single-domain-example.md"

??? example "Multiple Domains from Router's Rule Example"
 
    * A certificate for the domains `example.com` (main) and `blog.example.org`
      is requested:
    
    --8<-- "content/https/include-acme-multiple-domains-from-rule-example.md"
    
??? example "Multiple Domains from Router's `tls.domain` Example"

    * A certificate for the domains `example.com` (main) and `*.example.org` (SAN)
      is requested:
      
    --8<-- "content/https/include-acme-multiple-domains-example.md"

## Automatic Renewals

Traefik automatically tracks the expiry date of ACME certificates it generates.

If there are less than 30 days remaining before the certificate expires, Traefik will attempt to renew it automatically.

!!! info ""
    Certificates that are no longer used may still be renewed, as Traefik does not currently check if the certificate is being used before renewing.

## Using LetsEncrypt with Kubernetes

When using LetsEncrypt with kubernetes, there are some known caveats with both the [ingress](../providers/kubernetes-ingress.md) and [crd](../providers/kubernetes-crd.md) providers.

!!! info ""
    If you intend to run multiple instances of Traefik with LetsEncrypt, please ensure you read the sections on those provider pages.

## The Different ACME Challenges

!!! important "Defining a certificates resolver does not result in all routers automatically using it. Each router that is supposed to use the resolver must [reference](../routing/routers/index.md#certresolver) it."

### `tlsChallenge`

Use the `TLS-ALPN-01` challenge to generate and renew ACME certificates by provisioning a TLS certificate.

As described on the Let's Encrypt [community forum](https://community.letsencrypt.org/t/support-for-ports-other-than-80-and-443/3419/72),
when using the `TLS-ALPN-01` challenge, Traefik must be reachable by Let's Encrypt through port 443.

??? example "Configuring the `tlsChallenge`"

    ```toml tab="File (TOML)"
    [certificatesResolvers.myresolver.acme]
      # ...
      [certificatesResolvers.myresolver.acme.tlsChallenge]
    ```

    ```yaml tab="File (YAML)"
    certificatesResolvers:
      myresolver:
        acme:
          # ...
          tlsChallenge: {}
    ```
    
    ```bash tab="CLI"
    # ...
    --certificatesresolvers.myresolver.acme.tlschallenge=true
    ```

### `httpChallenge`

Use the `HTTP-01` challenge to generate and renew ACME certificates by provisioning an HTTP resource under a well-known URI.

As described on the Let's Encrypt [community forum](https://community.letsencrypt.org/t/support-for-ports-other-than-80-and-443/3419/72),
when using the `HTTP-01` challenge, `certificatesresolvers.myresolver.acme.httpchallenge.entrypoint` must be reachable by Let's Encrypt through port 80.

??? example "Using an EntryPoint Called web for the `httpChallenge`"

    ```toml tab="File (TOML)"
    [entryPoints]
      [entryPoints.web]
        address = ":80"
      
      [entryPoints.websecure]
        address = ":443"
    
    [certificatesResolvers.myresolver.acme]
      # ...
      [certificatesResolvers.myresolver.acme.httpChallenge]
        entryPoint = "web"
    ```

    ```yaml tab="File (YAML)"
    entryPoints:
      web:
        address: ":80"
    
      websecure:
        address: ":443"
    
    certificatesResolvers:
      myresolver:
        acme:
          # ...
          httpChallenge:
            entryPoint: web
    ```
    
    ```bash tab="CLI"
    --entrypoints.web.address=:80
    --entrypoints.websecure.address=:443
    # ...
    --certificatesresolvers.myresolver.acme.httpchallenge.entrypoint=web
    ```

!!! info ""
    Redirection is fully compatible with the `HTTP-01` challenge.

### `dnsChallenge`

Use the `DNS-01` challenge to generate and renew ACME certificates by provisioning a DNS record.

??? example "Configuring a `dnsChallenge` with the DigitalOcean Provider"

    ```toml tab="File (TOML)"
    [certificatesResolvers.myresolver.acme]
      # ...
      [certificatesResolvers.myresolver.acme.dnsChallenge]
        provider = "digitalocean"
        delayBeforeCheck = 0
    # ...
    ```
    
    ```yaml tab="File (YAML)"
    certificatesResolvers:
      myresolver:
        acme:
          # ...
          dnsChallenge:
            provider: digitalocean
            delayBeforeCheck: 0
        # ...
    ```
    
    ```bash tab="CLI"
    # ...
    --certificatesresolvers.myresolver.acme.dnschallenge.provider=digitalocean
    --certificatesresolvers.myresolver.acme.dnschallenge.delaybeforecheck=0
    # ...
    ```

    !!! important
        A `provider` is mandatory.

#### `providers`
 
Here is a list of supported `providers`, that can automate the DNS verification,
along with the required environment variables and their [wildcard & root domain support](#wildcard-domains).
Do not hesitate to complete it.

Many lego environment variables can be overridden by their respective `_FILE` counterpart, which should have a filepath to a file that contains the secret as its value.
For example, `CF_API_EMAIL_FILE=/run/secrets/traefik_cf-api-email` could be used to provide a Cloudflare API email address as a Docker secret named `traefik_cf-api-email`.

For complete details, refer to your provider's _Additional configuration_ link.

| Provider Name                                               | Provider Code  | Environment Variables                                                                                                                       |                                                                             |
|-------------------------------------------------------------|----------------|---------------------------------------------------------------------------------------------------------------------------------------------|-----------------------------------------------------------------------------|
| [ACME DNS](https://github.com/joohoi/acme-dns)              | `acme-dns`     | `ACME_DNS_API_BASE`, `ACME_DNS_STORAGE_PATH`                                                                                                | [Additional configuration](https://go-acme.github.io/lego/dns/acme-dns)     |
| [Alibaba Cloud](https://www.alibabacloud.com)               | `alidns`       | `ALICLOUD_ACCESS_KEY`, `ALICLOUD_SECRET_KEY`, `ALICLOUD_REGION_ID`                                                                          | [Additional configuration](https://go-acme.github.io/lego/dns/alidns)       |
| [ArvanCloud](https://www.arvancloud.com/en)                 | `arvancloud`   | `ARVANCLOUD_API_KEY`                                                                                                                        | [Additional configuration](https://go-acme.github.io/lego/dns/arvancloud)   |
| [Auroradns](https://www.pcextreme.com/dns-health-checks)    | `auroradns`    | `AURORA_USER_ID`, `AURORA_KEY`, `AURORA_ENDPOINT`                                                                                           | [Additional configuration](https://go-acme.github.io/lego/dns/auroradns)    |
| [Autodns](https://www.internetx.com/domains/autodns/)       | `autodns`      | `AUTODNS_API_USER`, `AUTODNS_API_PASSWORD`                                                                                                  | [Additional configuration](https://go-acme.github.io/lego/dns/autodns)      |
| [Azure](https://azure.microsoft.com/services/dns/)          | `azure`        | `AZURE_CLIENT_ID`, `AZURE_CLIENT_SECRET`, `AZURE_SUBSCRIPTION_ID`, `AZURE_TENANT_ID`, `AZURE_RESOURCE_GROUP`, `[AZURE_METADATA_ENDPOINT]`   | [Additional configuration](https://go-acme.github.io/lego/dns/azure)        |
| [Bindman](https://github.com/labbsr0x/bindman-dns-webhook)  | `bindman`      | `BINDMAN_MANAGER_ADDRESS`                                                                                                                   | [Additional configuration](https://go-acme.github.io/lego/dns/bindman)      |
| [Blue Cat](https://www.bluecatnetworks.com/)                | `bluecat`      | `BLUECAT_SERVER_URL`, `BLUECAT_USER_NAME`, `BLUECAT_PASSWORD`, `BLUECAT_CONFIG_NAME`, `BLUECAT_DNS_VIEW`                                    | [Additional configuration](https://go-acme.github.io/lego/dns/bluecat)      |
| [Checkdomain](https://www.checkdomain.de/)                  | `checkdomain`  | `CHECKDOMAIN_TOKEN`,                                                                                                                        | [Additional configuration](https://go-acme.github.io/lego/dns/checkdomain/) |
| [CloudDNS](https://vshosting.eu/)                           | `clouddns`     | `CLOUDDNS_CLIENT_ID`, `CLOUDDNS_EMAIL`, `CLOUDDNS_PASSWORD`                                                                                 | [Additional configuration](https://go-acme.github.io/lego/dns/clouddns)     |
| [ClouDNS](https://www.cloudns.net/)                         | `cloudns`      | `CLOUDNS_AUTH_ID`, `CLOUDNS_AUTH_PASSWORD`                                                                                                  | [Additional configuration](https://go-acme.github.io/lego/dns/cloudns)      |
| [Cloudflare](https://www.cloudflare.com)                    | `cloudflare`   | `CF_API_EMAIL`, `CF_API_KEY` [^5] or `CF_DNS_API_TOKEN`, `[CF_ZONE_API_TOKEN]`                                                              | [Additional configuration](https://go-acme.github.io/lego/dns/cloudflare)   |
| [CloudXNS](https://www.cloudxns.net)                        | `cloudxns`     | `CLOUDXNS_API_KEY`, `CLOUDXNS_SECRET_KEY`                                                                                                   | [Additional configuration](https://go-acme.github.io/lego/dns/cloudxns)     |
| [ConoHa](https://www.conoha.jp)                             | `conoha`       | `CONOHA_TENANT_ID`, `CONOHA_API_USERNAME`, `CONOHA_API_PASSWORD`                                                                            | [Additional configuration](https://go-acme.github.io/lego/dns/conoha)       |
| [Constellix](https://constellix.com)                        | `constellix`   | `CONSTELLIX_API_KEY`, `CONSTELLIX_SECRET_KEY`                                                                                               | [Additional configuration](https://go-acme.github.io/lego/dns/constellix)   |
| [deSEC](https://desec.io)                                   | `desec`        | `DESEC_TOKEN`                                                                                                                               | [Additional configuration](https://go-acme.github.io/lego/dns/desec)        |
| [DigitalOcean](https://www.digitalocean.com)                | `digitalocean` | `DO_AUTH_TOKEN`                                                                                                                             | [Additional configuration](https://go-acme.github.io/lego/dns/digitalocean) |
| [DNSimple](https://dnsimple.com)                            | `dnsimple`     | `DNSIMPLE_OAUTH_TOKEN`, `DNSIMPLE_BASE_URL`                                                                                                 | [Additional configuration](https://go-acme.github.io/lego/dns/dnsimple)     |
| [DNS Made Easy](https://dnsmadeeasy.com)                    | `dnsmadeeasy`  | `DNSMADEEASY_API_KEY`, `DNSMADEEASY_API_SECRET`, `DNSMADEEASY_SANDBOX`                                                                      | [Additional configuration](https://go-acme.github.io/lego/dns/dnsmadeeasy)  |
| [DNSPod](https://www.dnspod.com/)                           | `dnspod`       | `DNSPOD_API_KEY`                                                                                                                            | [Additional configuration](https://go-acme.github.io/lego/dns/dnspod)       |
| [Domain Offensive (do.de)](https://www.do.de/)              | `dode`         | `DODE_TOKEN`                                                                                                                                | [Additional configuration](https://go-acme.github.io/lego/dns/dode)         |
| [Domeneshop](https://domene.shop)                           | `domeneshop`   | `DOMENESHOP_API_TOKEN`, `DOMENESHOP_API_SECRET`                                                                                             | [Additional configuration](https://go-acme.github.io/lego/dns/domeneshop)   |
| [DreamHost](https://www.dreamhost.com/)                     | `dreamhost`    | `DREAMHOST_API_KEY`                                                                                                                         | [Additional configuration](https://go-acme.github.io/lego/dns/dreamhost)    |
| [Duck DNS](https://www.duckdns.org/)                        | `duckdns`      | `DUCKDNS_TOKEN`                                                                                                                             | [Additional configuration](https://go-acme.github.io/lego/dns/duckdns)      |
| [Dyn](https://dyn.com)                                      | `dyn`          | `DYN_CUSTOMER_NAME`, `DYN_USER_NAME`, `DYN_PASSWORD`                                                                                        | [Additional configuration](https://go-acme.github.io/lego/dns/dyn)          |
| [Dynu](https://www.dynu.com)                                | `dynu`         | `DYNU_API_KEY`                                                                                                                              | [Additional configuration](https://go-acme.github.io/lego/dns/dynu)         |
| [EasyDNS](https://easydns.com/)                             | `easydns`      | `EASYDNS_TOKEN`, `EASYDNS_KEY`                                                                                                              | [Additional configuration](https://go-acme.github.io/lego/dns/easydns)      |
| [EdgeDNS](https://www.akamai.com/)                          | `edgedns`      | `AKAMAI_CLIENT_TOKEN`,  `AKAMAI_CLIENT_SECRET`,  `AKAMAI_ACCESS_TOKEN`                                                                      | [Additional configuration](https://go-acme.github.io/lego/dns/edgedns)      |
| External Program                                            | `exec`         | `EXEC_PATH`                                                                                                                                 | [Additional configuration](https://go-acme.github.io/lego/dns/exec)         |
| [Exoscale](https://www.exoscale.com)                        | `exoscale`     | `EXOSCALE_API_KEY`, `EXOSCALE_API_SECRET`, `EXOSCALE_ENDPOINT`                                                                              | [Additional configuration](https://go-acme.github.io/lego/dns/exoscale)     |
| [Fast DNS](https://www.akamai.com/)                         | `fastdns`      | `AKAMAI_CLIENT_TOKEN`,  `AKAMAI_CLIENT_SECRET`,  `AKAMAI_ACCESS_TOKEN`                                                                      | [Additional configuration](https://go-acme.github.io/lego/dns/edgedns)      |
| [Gandi](https://www.gandi.net)                              | `gandi`        | `GANDI_API_KEY`                                                                                                                             | [Additional configuration](https://go-acme.github.io/lego/dns/gandi)        |
| [Gandi v5](http://doc.livedns.gandi.net)                    | `gandiv5`      | `GANDIV5_API_KEY`                                                                                                                           | [Additional configuration](https://go-acme.github.io/lego/dns/gandiv5)      |
| [Glesys](https://glesys.com/)                               | `glesys`       | `GLESYS_API_USER`, `GLESYS_API_KEY`, `GLESYS_DOMAIN`                                                                                        | [Additional configuration](https://go-acme.github.io/lego/dns/glesys)       |
| [GoDaddy](https://godaddy.com/)                             | `godaddy`      | `GODADDY_API_KEY`, `GODADDY_API_SECRET`                                                                                                     | [Additional configuration](https://go-acme.github.io/lego/dns/godaddy)      |
| [Google Cloud DNS](https://cloud.google.com/dns/docs/)      | `gcloud`       | `GCE_PROJECT`, Application Default Credentials [^2] [^3], [`GCE_SERVICE_ACCOUNT_FILE`]                                                      | [Additional configuration](https://go-acme.github.io/lego/dns/gcloud)       |
| [Hetzner](https://hetzner.com)                              | `hetzner`      | `HETZNER_API_KEY`                                                                                                                           | [Additional configuration](https://go-acme.github.io/lego/dns/hetzner)      |
| [hosting.de](https://www.hosting.de)                        | `hostingde`    | `HOSTINGDE_API_KEY`, `HOSTINGDE_ZONE_NAME`                                                                                                  | [Additional configuration](https://go-acme.github.io/lego/dns/hostingde)    |
| HTTP request                                                | `httpreq`      | `HTTPREQ_ENDPOINT`, `HTTPREQ_MODE`, `HTTPREQ_USERNAME`, `HTTPREQ_PASSWORD` [^1]                                                             | [Additional configuration](https://go-acme.github.io/lego/dns/httpreq)      |
| [HyperOne](https://www.hyperone.com)                        | `hyperone`     | `HYPERONE_PASSPORT_LOCATION`, `HYPERONE_LOCATION_ID`                                                                                        | [Additional configuration](https://go-acme.github.io/lego/dns/hyperone)     |
| [IIJ](https://www.iij.ad.jp/)                               | `iij`          | `IIJ_API_ACCESS_KEY`, `IIJ_API_SECRET_KEY`, `IIJ_DO_SERVICE_CODE`                                                                           | [Additional configuration](https://go-acme.github.io/lego/dns/iij)          |
| [Infomaniak](https://www.infomaniak.com)                    | `infomaniak`   | `INFOMANIAK_ACCESS_TOKEN`                                                                                                                   | [Additional configuration](https://go-acme.github.io/lego/dns/infomaniak)   |
| [INWX](https://www.inwx.de/en)                              | `inwx`         | `INWX_USERNAME`, `INWX_PASSWORD`                                                                                                            | [Additional configuration](https://go-acme.github.io/lego/dns/inwx)         |
| [ionos](https://ionos.com/)                                 | `ionos`        | `IONOS_API_KEY`                                                                                                                             | [Additional configuration](https://go-acme.github.io/lego/dns/ionos)        |
| [Joker.com](https://joker.com)                              | `joker`        | `JOKER_API_MODE` with `JOKER_API_KEY` or `JOKER_USERNAME`, `JOKER_PASSWORD`                                                                 | [Additional configuration](https://go-acme.github.io/lego/dns/joker)        |
| [Lightsail](https://aws.amazon.com/lightsail/)              | `lightsail`    | `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `DNS_ZONE`                                                                                    | [Additional configuration](https://go-acme.github.io/lego/dns/lightsail)    |
| [Linode v4](https://www.linode.com)                         | `linode`       | `LINODE_TOKEN`                                                                                                                              | [Additional configuration](https://go-acme.github.io/lego/dns/linode)       |
| [Liquid Web](https://www.liquidweb.com/)                    | `liquidweb`    | `LIQUID_WEB_PASSWORD`, `LIQUID_WEB_USERNAME`, `LIQUID_WEB_ZONE`                                                                             | [Additional configuration](https://go-acme.github.io/lego/dns/liquidweb)    |
| [Loopia](https://loopia.com/)                               | `loopia`       | `LOOPIA_API_PASSWORD`, `LOOPIA_API_USER`                                                                                                    | [Additional configuration](https://go-acme.github.io/lego/dns/loopia)       |
| [LuaDNS](https://luadns.com)                                | `luadns`       | `LUADNS_API_USERNAME`, `LUADNS_API_TOKEN`                                                                                                   | [Additional configuration](https://go-acme.github.io/lego/dns/luadns)       |
| manual                                                      | `manual`       | none, but you need to run Traefik interactively [^4], turn on debug log to see instructions and press <kbd>Enter</kbd>.                     |                                                                             |
| [MyDNS.jp](https://www.mydns.jp/)                           | `mydnsjp`      | `MYDNSJP_MASTER_ID`, `MYDNSJP_PASSWORD`                                                                                                     | [Additional configuration](https://go-acme.github.io/lego/dns/mydnsjp)      |
| [Mythic Beasts](https://www.mythic-beasts.com)              | `mythicbeasts` | `MYTHICBEASTS_USER_NAME`, `MYTHICBEASTS_PASSWORD`                                                                                           | [Additional configuration](https://go-acme.github.io/lego/dns/mythicbeasts) |
| [Namecheap](https://www.namecheap.com)                      | `namecheap`    | `NAMECHEAP_API_USER`, `NAMECHEAP_API_KEY`                                                                                                   | [Additional configuration](https://go-acme.github.io/lego/dns/namecheap)    |
| [name.com](https://www.name.com/)                           | `namedotcom`   | `NAMECOM_USERNAME`, `NAMECOM_API_TOKEN`, `NAMECOM_SERVER`                                                                                   | [Additional configuration](https://go-acme.github.io/lego/dns/namedotcom)   |
| [Namesilo](https://www.namesilo.com/)                       | `namesilo`     | `NAMESILO_API_KEY`                                                                                                                          | [Additional configuration](https://go-acme.github.io/lego/dns/namesilo)     |
| [Netcup](https://www.netcup.eu/)                            | `netcup`       | `NETCUP_CUSTOMER_NUMBER`, `NETCUP_API_KEY`, `NETCUP_API_PASSWORD`                                                                           | [Additional configuration](https://go-acme.github.io/lego/dns/netcup)       |
| [Netlify](https://www.netlify.com)                          | `netlify`      | `NETLIFY_TOKEN`                                                                                                                             | [Additional configuration](https://go-acme.github.io/lego/dns/netlify)      |
| [NIFCloud](https://cloud.nifty.com/service/dns.htm)         | `nifcloud`     | `NIFCLOUD_ACCESS_KEY_ID`, `NIFCLOUD_SECRET_ACCESS_KEY`                                                                                      | [Additional configuration](https://go-acme.github.io/lego/dns/nifcloud)     |
| [Njalla](https://njal.la)                                   | `njalla`       | `NJALLA_TOKEN`                                                                                                                              | [Additional configuration](https://go-acme.github.io/lego/dns/njalla)       |
| [NS1](https://ns1.com/)                                     | `ns1`          | `NS1_API_KEY`                                                                                                                               | [Additional configuration](https://go-acme.github.io/lego/dns/ns1)          |
| [Open Telekom Cloud](https://cloud.telekom.de)              | `otc`          | `OTC_DOMAIN_NAME`, `OTC_USER_NAME`, `OTC_PASSWORD`, `OTC_PROJECT_NAME`, `OTC_IDENTITY_ENDPOINT`                                             | [Additional configuration](https://go-acme.github.io/lego/dns/otc)          |
| [OVH](https://www.ovh.com)                                  | `ovh`          | `OVH_ENDPOINT`, `OVH_APPLICATION_KEY`, `OVH_APPLICATION_SECRET`, `OVH_CONSUMER_KEY`                                                         | [Additional configuration](https://go-acme.github.io/lego/dns/ovh)          |
| [Openstack Designate](https://docs.openstack.org/designate) | `designate`    | `OS_AUTH_URL`, `OS_USERNAME`, `OS_PASSWORD`, `OS_TENANT_NAME`, `OS_REGION_NAME`                                                             | [Additional configuration](https://go-acme.github.io/lego/dns/designate)    |
| [Oracle Cloud](https://cloud.oracle.com/home)               | `oraclecloud`  | `OCI_COMPARTMENT_OCID`, `OCI_PRIVKEY_FILE`, `OCI_PRIVKEY_PASS`, `OCI_PUBKEY_FINGERPRINT`, `OCI_REGION`, `OCI_TENANCY_OCID`, `OCI_USER_OCID` | [Additional configuration](https://go-acme.github.io/lego/dns/oraclecloud)  |
| [PowerDNS](https://www.powerdns.com)                        | `pdns`         | `PDNS_API_KEY`, `PDNS_API_URL`                                                                                                              | [Additional configuration](https://go-acme.github.io/lego/dns/pdns)         |
| [Rackspace](https://www.rackspace.com/cloud/dns)            | `rackspace`    | `RACKSPACE_USER`, `RACKSPACE_API_KEY`                                                                                                       | [Additional configuration](https://go-acme.github.io/lego/dns/rackspace)    |
| [reg.ru](https://www.reg.ru)                                | `regru`        | `REGRU_USERNAME`, `REGRU_PASSWORD`                                                                                                          | [Additional configuration](https://go-acme.github.io/lego/dns/regru)        |
| [RFC2136](https://tools.ietf.org/html/rfc2136)              | `rfc2136`      | `RFC2136_TSIG_KEY`, `RFC2136_TSIG_SECRET`, `RFC2136_TSIG_ALGORITHM`, `RFC2136_NAMESERVER`                                                   | [Additional configuration](https://go-acme.github.io/lego/dns/rfc2136)      |
| [Route 53](https://aws.amazon.com/route53/)                 | `route53`      | `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `[AWS_REGION]`, `[AWS_HOSTED_ZONE_ID]` or a configured user/instance IAM profile.             | [Additional configuration](https://go-acme.github.io/lego/dns/route53)      |
| [RimuHosting](https://rimuhosting.com)                      | `rimuhosting`  | `RIMUHOSTING_API_KEY`                                                                                                                       | [Additional configuration](https://go-acme.github.io/lego/dns/rimuhosting)  |
| [Sakura Cloud](https://cloud.sakura.ad.jp/)                 | `sakuracloud`  | `SAKURACLOUD_ACCESS_TOKEN`, `SAKURACLOUD_ACCESS_TOKEN_SECRET`                                                                               | [Additional configuration](https://go-acme.github.io/lego/dns/sakuracloud)  |
| [Scaleway](https://www.scaleway.com)                        | `scaleway`     | `SCALEWAY_API_TOKEN`                                                                                                                        | [Additional configuration](https://go-acme.github.io/lego/dns/scaleway)     |
| [Selectel](https://selectel.ru/en/)                         | `selectel`     | `SELECTEL_API_TOKEN`                                                                                                                        | [Additional configuration](https://go-acme.github.io/lego/dns/selectel)     |
| [Servercow](https://servercow.de)                           | `servercow`    | `SERVERCOW_USERNAME`, `SERVERCOW_PASSWORD`                                                                                                  | [Additional configuration](https://go-acme.github.io/lego/dns/servercow)    |
| [Stackpath](https://www.stackpath.com/)                     | `stackpath`    | `STACKPATH_CLIENT_ID`, `STACKPATH_CLIENT_SECRET`, `STACKPATH_STACK_ID`                                                                      | [Additional configuration](https://go-acme.github.io/lego/dns/stackpath)    |
| [TransIP](https://www.transip.nl/)                          | `transip`      | `TRANSIP_ACCOUNT_NAME`, `TRANSIP_PRIVATE_KEY_PATH`                                                                                          | [Additional configuration](https://go-acme.github.io/lego/dns/transip)      |
| [VegaDNS](https://github.com/shupp/VegaDNS-API)             | `vegadns`      | `SECRET_VEGADNS_KEY`, `SECRET_VEGADNS_SECRET`, `VEGADNS_URL`                                                                                | [Additional configuration](https://go-acme.github.io/lego/dns/vegadns)      |
| [Versio](https://www.versio.nl/domeinnamen)                 | `versio`       | `VERSIO_USERNAME`, `VERSIO_PASSWORD`                                                                                                        | [Additional configuration](https://go-acme.github.io/lego/dns/versio)       |
| [Vscale](https://vscale.io/)                                | `vscale`       | `VSCALE_API_TOKEN`                                                                                                                          | [Additional configuration](https://go-acme.github.io/lego/dns/vscale)       |
| [VULTR](https://www.vultr.com)                              | `vultr`        | `VULTR_API_KEY`                                                                                                                             | [Additional configuration](https://go-acme.github.io/lego/dns/vultr)        |
| [Yandex](https://yandex.com)                                | `yandex`       | `YANDEX_PDD_TOKEN`                                                                                                                          | [Additional configuration](https://go-acme.github.io/lego/dns/yandex)       |
| [Zone.ee](https://www.zone.ee)                              | `zoneee`       | `ZONEEE_API_USER`, `ZONEEE_API_KEY`                                                                                                         | [Additional configuration](https://go-acme.github.io/lego/dns/zoneee)       |
| [Zonomi](https://zonomi.com)                                | `zonomi`       | `ZONOMI_API_KEY`                                                                                                                            | [Additional configuration](https://go-acme.github.io/lego/dns/zonomi)       |

[^1]: more information about the HTTP message format can be found [here](https://go-acme.github.io/lego/dns/httpreq/)
[^2]: [providing_credentials_to_your_application](https://cloud.google.com/docs/authentication/production)
[^3]: [google/default.go](https://github.com/golang/oauth2/blob/36a7019397c4c86cf59eeab3bc0d188bac444277/google/default.go#L61-L76)
[^4]: `docker stack` remark: there is no way to support terminal attached to container when deploying with `docker stack`, so you might need to run container with `docker run -it` to generate certificates using `manual` provider.
[^5]: The `Global API Key` needs to be used, not the `Origin CA Key`.

!!! info "`delayBeforeCheck`"
    By default, the `provider` verifies the TXT record _before_ letting ACME verify.
    You can delay this operation by specifying a delay (in seconds) with `delayBeforeCheck` (value must be greater than zero).
    This option is useful when internal networks block external DNS queries.

#### `resolvers`

Use custom DNS servers to resolve the FQDN authority.

```toml tab="File (TOML)"
[certificatesResolvers.myresolver.acme]
  # ...
  [certificatesResolvers.myresolver.acme.dnsChallenge]
    # ...
    resolvers = ["1.1.1.1:53", "8.8.8.8:53"]
```

```yaml tab="File (YAML)"
certificatesResolvers:
  myresolver:
    acme:
      # ...
      dnsChallenge:
        # ...
        resolvers:
          - "1.1.1.1:53"
          - "8.8.8.8:53"
```

```bash tab="CLI"
# ...
--certificatesresolvers.myresolver.acme.dnschallenge.resolvers=1.1.1.1:53,8.8.8.8:53
```

#### Wildcard Domains

[ACME V2](https://community.letsencrypt.org/t/acme-v2-and-wildcard-certificate-support-is-live/55579) supports wildcard certificates.
As described in [Let's Encrypt's post](https://community.letsencrypt.org/t/staging-endpoint-for-acme-v2/49605) wildcard certificates can only be generated through a [`DNS-01` challenge](#dnschallenge).

## External Account Binding

- `kid`: Key identifier from External CA
- `hmacEncoded`: HMAC key from External CA, should be in Base64 URL Encoding without padding format

```toml tab="File (TOML)"
[certificatesResolvers.myresolver.acme]
  # ...
  [certificatesResolvers.myresolver.acme.eab]
    kid = "abc-keyID-xyz"
    hmacEncoded = "abc-hmac-xyz"
```

```yaml tab="File (YAML)"
certificatesResolvers:
  myresolver:
    acme:
      # ...
      eab:
        kid: abc-keyID-xyz
        hmacEncoded: abc-hmac-xyz
```

```bash tab="CLI"
# ...
--certificatesresolvers.myresolver.acme.eab.kid=abc-keyID-xyz
--certificatesresolvers.myresolver.acme.eab.hmacencoded=abc-hmac-xyz
```

## More Configuration

### `caServer`

_Required, Default="https://acme-v02.api.letsencrypt.org/directory"_

The CA server to use:

- Let's Encrypt production server: https://acme-v02.api.letsencrypt.org/directory
- Let's Encrypt staging server: https://acme-staging-v02.api.letsencrypt.org/directory

??? example "Using the Let's Encrypt staging server"

    ```toml tab="File (TOML)"
    [certificatesResolvers.myresolver.acme]
      # ...
      caServer = "https://acme-staging-v02.api.letsencrypt.org/directory"
      # ...
    ```
    
    ```yaml tab="File (YAML)"
    certificatesResolvers:
      myresolver:
        acme:
          # ...
          caServer: https://acme-staging-v02.api.letsencrypt.org/directory
          # ...
    ```

    ```bash tab="CLI"
    # ...
    --certificatesresolvers.myresolver.acme.caserver=https://acme-staging-v02.api.letsencrypt.org/directory
    # ...
    ```

### `storage`

_Required, Default="acme.json"_

The `storage` option sets the location where your ACME certificates are saved to.

```toml tab="File (TOML)"
[certificatesResolvers.myresolver.acme]
  # ...
  storage = "acme.json"
  # ...
```

```yaml tab="File (YAML)"
certificatesResolvers:
  myresolver:
    acme:
      # ...
      storage: acme.json
      # ...
```

```bash tab="CLI"
# ...
--certificatesresolvers.myresolver.acme.storage=acme.json
# ...
```

ACME certificates are stored in a JSON file that needs to have a `600` file mode.

In Docker you can mount either the JSON file, or the folder containing it:

```bash
docker run -v "/my/host/acme.json:/acme.json" traefik
```

```bash
docker run -v "/my/host/acme:/etc/traefik/acme" traefik
```

!!! warning
    For concurrency reasons, this file cannot be shared across multiple instances of Traefik.

### `preferredChain`

_Optional, Default=""_

Preferred chain to use.

If the CA offers multiple certificate chains, prefer the chain with an issuer matching this Subject Common Name.
If no match, the default offered chain will be used.

```toml tab="File (TOML)"
[certificatesResolvers.myresolver.acme]
  # ...
  preferredChain = "ISRG Root X1"
  # ...
```

```yaml tab="File (YAML)"
certificatesResolvers:
  myresolver:
    acme:
      # ...
      preferredChain: 'ISRG Root X1'
      # ...
```

```bash tab="CLI"
# ...
--certificatesresolvers.myresolver.acme.preferredChain="ISRG Root X1"
# ...
```

### `keyType`

_Optional, Default="RSA4096"_

KeyType used for generating certificate private key. Allow value 'EC256', 'EC384', 'RSA2048', 'RSA4096', 'RSA8192'.

```toml tab="File (TOML)"
[certificatesResolvers.myresolver.acme]
  # ...
  keyType = "RSA4096"
  # ...
```

```yaml tab="File (YAML)"
certificatesResolvers:
  myresolver:
    acme:
      # ...
      keyType: 'RSA4096'
      # ...
```

```bash tab="CLI"
# ...
--certificatesresolvers.myresolver.acme.keyType="RSA4096"
# ...
```

## Fallback

If Let's Encrypt is not reachable, the following certificates will apply:

  1. Previously generated ACME certificates (before downtime)
  1. Expired ACME certificates
  1. Provided certificates

!!! important
    For new (sub)domains which need Let's Encrypt authentication, the default Traefik certificate will be used until Traefik is restarted.
