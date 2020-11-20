module github.com/traefik/traefik/v2

go 1.15

require (
	github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78 // indirect
	github.com/Azure/go-autorest/autorest v0.9.6 // indirect
	github.com/BurntSushi/toml v0.3.1
	github.com/ExpediaDotCom/haystack-client-go v0.0.0-20190315171017-e7edbdf53a61
	github.com/Masterminds/semver v1.4.2 // indirect
	github.com/Masterminds/sprig v2.22.0+incompatible
	github.com/Microsoft/hcsshim v0.8.7 // indirect
	github.com/NYTimes/gziphandler v1.1.1
	github.com/Shopify/sarama v1.23.1 // indirect
	github.com/abbot/go-http-auth v0.0.0-00010101000000-000000000000
	github.com/abronan/valkeyrie v0.0.0-20200127174252-ef4277a138cd
	github.com/aws/aws-sdk-go v1.31.12
	github.com/c0va23/go-proxyprotocol v0.9.1
	github.com/cenkalti/backoff/v4 v4.0.2
	github.com/containerd/containerd v1.3.2 // indirect
	github.com/containous/alice v0.0.0-20181107144136-d83ebdd94cbd
	github.com/coreos/go-systemd v0.0.0-20191104093116-d3cd4ed1dbcf
	github.com/davecgh/go-spew v1.1.1
	github.com/docker/cli v0.0.0-20200221155518-740919cc7fc0
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/docker v0.7.3-0.20190327010347-be7ac8be2ae0
	github.com/docker/docker-credential-helpers v0.6.3 // indirect
	github.com/docker/go-connections v0.4.0
	github.com/docker/go-metrics v0.0.0-20181218153428-b84716841b82 // indirect
	github.com/docker/libcompose v0.0.0-20190805081528-eac9fe1b8b03 // indirect
	github.com/docker/libtrust v0.0.0-20160708172513-aabc10ec26b7 // indirect
	github.com/donovanhide/eventsource v0.0.0-20170630084216-b8f31a59085e // indirect
	github.com/eapache/channels v1.1.0
	github.com/elazarl/go-bindata-assetfs v1.0.0
	github.com/fatih/structs v1.1.0
	github.com/flynn/go-shlex v0.0.0-20150515145356-3f9db97f8568 // indirect
	github.com/gambol99/go-marathon v0.0.0-20180614232016-99a156b96fb2
	github.com/go-acme/lego/v4 v4.1.0
	github.com/go-check/check v0.0.0-00010101000000-000000000000
	github.com/go-kit/kit v0.10.1-0.20200915143503-439c4d2ed3ea
	github.com/golang/protobuf v1.4.3
	github.com/google/go-github/v28 v28.1.1
	github.com/gorilla/mux v1.7.4
	github.com/gorilla/websocket v1.4.2
	github.com/hashicorp/consul/api v1.3.0
	github.com/hashicorp/go-version v1.2.0
	github.com/influxdata/influxdb1-client v0.0.0-20191209144304-8bf82d3c094d
	github.com/instana/go-sensor v1.5.1
	github.com/libkermit/compose v0.0.0-20171122111507-c04e39c026ad
	github.com/libkermit/docker v0.0.0-20171122101128-e6674d32b807
	github.com/libkermit/docker-check v0.0.0-20171122104347-1113af38e591
	github.com/magiconair/properties v1.8.1 // indirect
	github.com/mailgun/ttlmap v0.0.0-20170619185759-c1c17f74874f
	github.com/miekg/dns v1.1.31
	github.com/mitchellh/copystructure v1.0.0
	github.com/mitchellh/hashstructure v1.0.0
	github.com/mitchellh/mapstructure v1.3.3
	github.com/morikuni/aec v0.0.0-20170113033406-39771216ff4c // indirect
	github.com/opencontainers/go-digest v1.0.0-rc1 // indirect
	github.com/opencontainers/image-spec v1.0.1 // indirect
	github.com/opencontainers/runc v1.0.0-rc10 // indirect
	github.com/opentracing/opentracing-go v1.1.0
	github.com/openzipkin-contrib/zipkin-go-opentracing v0.4.5
	github.com/openzipkin/zipkin-go v0.2.5
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/philhofer/fwd v1.0.0 // indirect
	github.com/pmezard/go-difflib v1.0.0
	github.com/prometheus/client_golang v1.8.0
	github.com/prometheus/client_model v0.2.0
	github.com/rancher/go-rancher-metadata v0.0.0-20200311180630-7f4c936a06ac
	github.com/sirupsen/logrus v1.6.0
	github.com/stretchr/testify v1.6.1
	github.com/stvp/go-udp-testing v0.0.0-20191102171040-06b61409b154
	github.com/tinylib/msgp v1.0.2 // indirect
	github.com/traefik/paerser v0.1.0
	github.com/traefik/yaegi v0.9.7
	github.com/uber/jaeger-client-go v2.25.0+incompatible
	github.com/uber/jaeger-lib v2.2.0+incompatible
	github.com/unrolled/render v1.0.2
	github.com/unrolled/secure v1.0.7
	github.com/vdemeester/shakers v0.1.0
	github.com/vulcand/oxy v1.1.0
	github.com/vulcand/predicate v1.1.0
	go.elastic.co/apm v1.7.0
	go.elastic.co/apm/module/apmot v1.7.0
	golang.org/x/mod v0.3.0
	golang.org/x/net v0.0.0-20200904194848-62affa334b73
	golang.org/x/time v0.0.0-20200630173020-3af7569d3a1e
	google.golang.org/grpc v1.33.1
	gopkg.in/DataDog/dd-trace-go.v1 v1.19.0
	gopkg.in/fsnotify.v1 v1.4.7
	gopkg.in/jcmturner/goidentity.v3 v3.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776
	k8s.io/api v0.19.2
	k8s.io/apimachinery v0.19.2
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	k8s.io/code-generator v0.19.2
	k8s.io/utils v0.0.0-20200729134348-d5654de09c73 // indirect
	knative.dev/networking v0.0.0-20201113013400-9912cdd6e95c
	mvdan.cc/xurls/v2 v2.1.0
)

// k8s.io/client-go v0.19.2
replace k8s.io/client-go => k8s.io/client-go v0.19.2
// google.golang.org/grpc v1.27.1
replace google.golang.org/grpc => google.golang.org/grpc v1.27.1
    
// Docker v19.03.6
replace github.com/docker/docker => github.com/docker/engine v1.4.2-0.20200204220554-5f6d6f3f2203

// Containous forks
replace (
	github.com/abbot/go-http-auth => github.com/containous/go-http-auth v0.4.1-0.20200324110947-a37a7636d23e
	github.com/go-check/check => github.com/containous/check v0.0.0-20170915194414-ca0bf163426a
	github.com/gorilla/mux => github.com/containous/mux v0.0.0-20181024131434-c33f32e26898
	github.com/mailgun/minheap => github.com/containous/minheap v0.0.0-20190809180810-6e71eb837595
	github.com/mailgun/multibuf => github.com/containous/multibuf v0.0.0-20190809014333-8b6c9a7e6bba
)
