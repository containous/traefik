package anonymize

import (
	"flag"
	"os"
	"strings"
	"testing"
	"time"

	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ptypes "github.com/traefik/paerser/types"
	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/config/static"
	"github.com/traefik/traefik/v2/pkg/ping"
	"github.com/traefik/traefik/v2/pkg/plugins"
	"github.com/traefik/traefik/v2/pkg/provider/acme"
	"github.com/traefik/traefik/v2/pkg/provider/consulcatalog"
	"github.com/traefik/traefik/v2/pkg/provider/docker"
	"github.com/traefik/traefik/v2/pkg/provider/ecs"
	"github.com/traefik/traefik/v2/pkg/provider/file"
	"github.com/traefik/traefik/v2/pkg/provider/http"
	"github.com/traefik/traefik/v2/pkg/provider/kubernetes/crd"
	"github.com/traefik/traefik/v2/pkg/provider/kubernetes/gateway"
	"github.com/traefik/traefik/v2/pkg/provider/kubernetes/ingress"
	"github.com/traefik/traefik/v2/pkg/provider/kv"
	"github.com/traefik/traefik/v2/pkg/provider/kv/consul"
	"github.com/traefik/traefik/v2/pkg/provider/kv/etcd"
	"github.com/traefik/traefik/v2/pkg/provider/kv/redis"
	"github.com/traefik/traefik/v2/pkg/provider/kv/zk"
	"github.com/traefik/traefik/v2/pkg/provider/marathon"
	"github.com/traefik/traefik/v2/pkg/provider/rancher"
	"github.com/traefik/traefik/v2/pkg/provider/rest"
	traefiktls "github.com/traefik/traefik/v2/pkg/tls"
	"github.com/traefik/traefik/v2/pkg/tracing/datadog"
	"github.com/traefik/traefik/v2/pkg/tracing/elastic"
	"github.com/traefik/traefik/v2/pkg/tracing/haystack"
	"github.com/traefik/traefik/v2/pkg/tracing/instana"
	"github.com/traefik/traefik/v2/pkg/tracing/jaeger"
	"github.com/traefik/traefik/v2/pkg/tracing/zipkin"
	"github.com/traefik/traefik/v2/pkg/types"
)

var updateExpected = flag.Bool("update_expected", false, "Update expected files in fixtures")

func TestDo_dynamicConfiguration(t *testing.T) {
	config := &dynamic.Configuration{}
	config.HTTP = &dynamic.HTTPConfiguration{
		Routers: map[string]*dynamic.Router{
			"foo": {
				EntryPoints: []string{"foo"},
				Middlewares: []string{"foo"},
				Service:     "foo",
				Rule:        "foo",
				Priority:    42,
				TLS: &dynamic.RouterTLSConfig{
					Options:      "foo",
					CertResolver: "foo",
					Domains: []types.Domain{
						{
							Main: "foo",
							SANs: []string{"foo"},
						},
					},
				},
			},
		},
		Services: map[string]*dynamic.Service{
			"foo": {
				LoadBalancer: &dynamic.ServersLoadBalancer{
					Sticky: &dynamic.Sticky{
						Cookie: &dynamic.Cookie{
							Name:     "foo",
							Secure:   true,
							HTTPOnly: true,
							SameSite: "foo",
						},
					},
					HealthCheck: &dynamic.ServerHealthCheck{
						Scheme:          "foo",
						Path:            "foo",
						Port:            42,
						Interval:        "foo",
						Timeout:         "foo",
						Hostname:        "foo",
						FollowRedirects: boolPtr(true),
						Headers: map[string]string{
							"foo": "bar",
						},
					},
					PassHostHeader: boolPtr(true),
					ResponseForwarding: &dynamic.ResponseForwarding{
						FlushInterval: "foo",
					},
					ServersTransport: "foo",
					Servers: []dynamic.Server{
						{
							URL: "http://127.0.0.1:8080",
						},
					},
				},
			},
			"bar": {
				Weighted: &dynamic.WeightedRoundRobin{
					Services: []dynamic.WRRService{
						{
							Name:   "foo",
							Weight: intPtr(42),
						},
					},
					Sticky: &dynamic.Sticky{
						Cookie: &dynamic.Cookie{
							Name:     "foo",
							Secure:   true,
							HTTPOnly: true,
							SameSite: "foo",
						},
					},
				},
			},
			"baz": {
				Mirroring: &dynamic.Mirroring{
					Service:     "foo",
					MaxBodySize: int64Ptr(42),
					Mirrors: []dynamic.MirrorService{
						{
							Name:    "foo",
							Percent: 42,
						},
					},
				},
			},
		},
		ServersTransports: map[string]*dynamic.ServersTransport{
			"foo": {
				ServerName:         "foo",
				InsecureSkipVerify: true,
				RootCAs:            []traefiktls.FileOrContent{"rootca.pem"},
				Certificates: []traefiktls.Certificate{
					{
						CertFile: "cert.pem",
						KeyFile:  "key.pem",
					},
				},
				MaxIdleConnsPerHost: 42,
				ForwardingTimeouts: &dynamic.ForwardingTimeouts{
					DialTimeout:           42,
					ResponseHeaderTimeout: 42,
					IdleConnTimeout:       42,
				},
			},
		},
		Models: map[string]*dynamic.Model{
			"foo": {
				Middlewares: []string{"foo"},
				TLS: &dynamic.RouterTLSConfig{
					Options:      "foo",
					CertResolver: "foo",
					Domains: []types.Domain{
						{
							Main: "foo",
							SANs: []string{"foo"},
						},
					},
				},
			},
		},
		Middlewares: map[string]*dynamic.Middleware{
			"foo": {
				AddPrefix: &dynamic.AddPrefix{
					Prefix: "foo",
				},
				StripPrefix: &dynamic.StripPrefix{
					Prefixes:   []string{"foo"},
					ForceSlash: true,
				},
				StripPrefixRegex: &dynamic.StripPrefixRegex{
					Regex: []string{"foo"},
				},
				ReplacePath: &dynamic.ReplacePath{
					Path: "foo",
				},
				ReplacePathRegex: &dynamic.ReplacePathRegex{
					Regex:       "foo",
					Replacement: "foo",
				},
				Chain: &dynamic.Chain{
					Middlewares: []string{"foo"},
				},
				IPWhiteList: &dynamic.IPWhiteList{
					SourceRange: []string{"foo"},
					IPStrategy: &dynamic.IPStrategy{
						Depth:       42,
						ExcludedIPs: []string{"127.0.0.1"},
					},
				},
				Headers: &dynamic.Headers{
					CustomRequestHeaders:              map[string]string{"foo": "bar"},
					CustomResponseHeaders:             map[string]string{"foo": "bar"},
					AccessControlAllowCredentials:     true,
					AccessControlAllowHeaders:         []string{"foo"},
					AccessControlAllowMethods:         []string{"foo"},
					AccessControlAllowOriginList:      []string{"foo"},
					AccessControlAllowOriginListRegex: []string{"foo"},
					AccessControlExposeHeaders:        []string{"foo"},
					AccessControlMaxAge:               42,
					AddVaryHeader:                     true,
					AllowedHosts:                      []string{"foo"},
					HostsProxyHeaders:                 []string{"foo"},
					SSLRedirect:                       true,
					SSLTemporaryRedirect:              true,
					SSLHost:                           "foo",
					SSLProxyHeaders:                   map[string]string{"foo": "bar"},
					SSLForceHost:                      true,
					STSSeconds:                        42,
					STSIncludeSubdomains:              true,
					STSPreload:                        true,
					ForceSTSHeader:                    true,
					FrameDeny:                         true,
					CustomFrameOptionsValue:           "foo",
					ContentTypeNosniff:                true,
					BrowserXSSFilter:                  true,
					CustomBrowserXSSValue:             "foo",
					ContentSecurityPolicy:             "foo",
					PublicKey:                         "foo",
					ReferrerPolicy:                    "foo",
					FeaturePolicy:                     "foo",
					PermissionsPolicy:                 "foo",
					IsDevelopment:                     true,
				},
				Errors: &dynamic.ErrorPage{
					Status:  []string{"foo"},
					Service: "foo",
					Query:   "foo",
				},
				RateLimit: &dynamic.RateLimit{
					Average: 42,
					Period:  42,
					Burst:   42,
					SourceCriterion: &dynamic.SourceCriterion{
						IPStrategy: &dynamic.IPStrategy{
							Depth:       42,
							ExcludedIPs: []string{"foo"},
						},
						RequestHeaderName: "foo",
						RequestHost:       true,
					},
				},
				RedirectRegex: &dynamic.RedirectRegex{
					Regex:       "foo",
					Replacement: "foo",
					Permanent:   true,
				},
				RedirectScheme: &dynamic.RedirectScheme{
					Scheme:    "foo",
					Port:      "foo",
					Permanent: true,
				},
				BasicAuth: &dynamic.BasicAuth{
					Users:        []string{"foo"},
					UsersFile:    "foo",
					Realm:        "foo",
					RemoveHeader: true,
					HeaderField:  "foo",
				},
				DigestAuth: &dynamic.DigestAuth{
					Users:        []string{"foo"},
					UsersFile:    "foo",
					RemoveHeader: true,
					Realm:        "foo",
					HeaderField:  "foo",
				},
				ForwardAuth: &dynamic.ForwardAuth{
					Address: "127.0.0.1",
					TLS: &dynamic.ClientTLS{
						CA:                 "ca.pem",
						CAOptional:         true,
						Cert:               "cert.pem",
						Key:                "cert.pem",
						InsecureSkipVerify: true,
					},
					TrustForwardHeader:       true,
					AuthResponseHeaders:      []string{"foo"},
					AuthResponseHeadersRegex: "foo",
					AuthRequestHeaders:       []string{"foo"},
				},
				InFlightReq: &dynamic.InFlightReq{
					Amount: 42,
					SourceCriterion: &dynamic.SourceCriterion{
						IPStrategy: &dynamic.IPStrategy{
							Depth:       42,
							ExcludedIPs: []string{"foo"},
						},
						RequestHeaderName: "foo",
						RequestHost:       true,
					},
				},
				Buffering: &dynamic.Buffering{
					MaxRequestBodyBytes:  42,
					MemRequestBodyBytes:  42,
					MaxResponseBodyBytes: 42,
					MemResponseBodyBytes: 42,
					RetryExpression:      "foo",
				},
				CircuitBreaker: &dynamic.CircuitBreaker{
					Expression: "foo",
				},
				Compress: &dynamic.Compress{
					ExcludedContentTypes: []string{"foo"},
				},
				PassTLSClientCert: &dynamic.PassTLSClientCert{
					PEM: true,
					Info: &dynamic.TLSClientCertificateInfo{
						NotAfter:  true,
						NotBefore: true,
						Sans:      true,
						Subject: &dynamic.TLSCLientCertificateSubjectDNInfo{
							Country:            true,
							Province:           true,
							Locality:           true,
							Organization:       true,
							OrganizationalUnit: true,
							CommonName:         true,
							SerialNumber:       true,
							DomainComponent:    true,
						},
						Issuer: &dynamic.TLSCLientCertificateIssuerDNInfo{
							Country:         true,
							Province:        true,
							Locality:        true,
							Organization:    true,
							CommonName:      true,
							SerialNumber:    true,
							DomainComponent: true,
						},
						SerialNumber: true,
					},
				},
				Retry: &dynamic.Retry{
					Attempts:        42,
					InitialInterval: 42,
				},
				ContentType: &dynamic.ContentType{
					AutoDetect: true,
				},
				Plugin: map[string]dynamic.PluginConf{
					"foo": {
						"answer": struct{ Answer int }{
							Answer: 42,
						},
					},
				},
			},
		},
	}
	config.TCP = &dynamic.TCPConfiguration{
		Routers: map[string]*dynamic.TCPRouter{
			"foo": {
				EntryPoints: []string{"foo"},
				Service:     "foo",
				Rule:        "foo",
				TLS: &dynamic.RouterTCPTLSConfig{
					Passthrough:  true,
					Options:      "foo",
					CertResolver: "foo",
					Domains: []types.Domain{
						{
							Main: "foo",
							SANs: []string{"foo"},
						},
					},
				},
			},
		},
		Services: map[string]*dynamic.TCPService{
			"foo": {
				LoadBalancer: &dynamic.TCPServersLoadBalancer{
					TerminationDelay: intPtr(42),
					ProxyProtocol: &dynamic.ProxyProtocol{
						Version: 42,
					},
					Servers: []dynamic.TCPServer{
						{
							Address: "127.0.0.1:8080",
						},
					},
				},
			},
			"bar": {
				Weighted: &dynamic.TCPWeightedRoundRobin{
					Services: []dynamic.TCPWRRService{
						{
							Name:   "foo",
							Weight: intPtr(42),
						},
					},
				},
			},
		},
	}
	config.UDP = &dynamic.UDPConfiguration{
		Routers: map[string]*dynamic.UDPRouter{
			"foo": {
				EntryPoints: []string{"foo"},
				Service:     "foo",
			},
		},
		Services: map[string]*dynamic.UDPService{
			"foo": {
				LoadBalancer: &dynamic.UDPServersLoadBalancer{
					Servers: []dynamic.UDPServer{
						{
							Address: "127.0.0.1:8080",
						},
					},
				},
			},
			"bar": {
				Weighted: &dynamic.UDPWeightedRoundRobin{
					Services: []dynamic.UDPWRRService{
						{
							Name:   "foo",
							Weight: intPtr(42),
						},
					},
				},
			},
		},
	}
	config.TLS = &dynamic.TLSConfiguration{
		Options: map[string]traefiktls.Options{
			"foo": {
				MinVersion:       "foo",
				MaxVersion:       "foo",
				CipherSuites:     []string{"foo"},
				CurvePreferences: []string{"foo"},
				ClientAuth: traefiktls.ClientAuth{
					CAFiles:        []traefiktls.FileOrContent{"ca.pem"},
					ClientAuthType: "RequireAndVerifyClientCert",
				},
				SniStrict:                true,
				PreferServerCipherSuites: true,
			},
		},
		Certificates: []*traefiktls.CertAndStores{
			{
				Certificate: traefiktls.Certificate{
					CertFile: "cert.pem",
					KeyFile:  "key.pem",
				},
				Stores: []string{"foo"},
			},
		},
		Stores: map[string]traefiktls.Store{
			"foo": {
				DefaultCertificate: &traefiktls.Certificate{
					CertFile: "cert.pem",
					KeyFile:  "key.pem",
				},
			},
		},
	}

	expectedConfiguration, err := os.ReadFile("./testdata/anonymized-dynamic-config.json")
	require.NoError(t, err)

	cleanJSON, err := Do(config, true)
	require.NoError(t, err)

	if *updateExpected {
		require.NoError(t, os.WriteFile("testdata/anonymized-dynamic-config.json", []byte(cleanJSON), 0666))
	}

	expected := strings.TrimSuffix(string(expectedConfiguration), "\n")
	assert.Equal(t, expected, cleanJSON)
}

func TestDo_staticConfiguration(t *testing.T) {
	config := &static.Configuration{}

	config.Global = &static.Global{
		CheckNewVersion:    true,
		SendAnonymousUsage: true,
	}

	config.ServersTransport = &static.ServersTransport{
		InsecureSkipVerify:  true,
		RootCAs:             []traefiktls.FileOrContent{"root.ca"},
		MaxIdleConnsPerHost: 42,
		ForwardingTimeouts: &static.ForwardingTimeouts{
			DialTimeout:           42,
			ResponseHeaderTimeout: 42,
			IdleConnTimeout:       42,
		},
	}

	config.EntryPoints = static.EntryPoints{
		"foobar": {
			Address: "foo Address",
			Transport: &static.EntryPointsTransport{
				LifeCycle: &static.LifeCycle{
					RequestAcceptGraceTimeout: ptypes.Duration(111 * time.Second),
					GraceTimeOut:              ptypes.Duration(111 * time.Second),
				},
				RespondingTimeouts: &static.RespondingTimeouts{
					ReadTimeout:  ptypes.Duration(111 * time.Second),
					WriteTimeout: ptypes.Duration(111 * time.Second),
					IdleTimeout:  ptypes.Duration(111 * time.Second),
				},
			},
			ProxyProtocol: &static.ProxyProtocol{
				Insecure:   true,
				TrustedIPs: []string{"127.0.0.1/32", "192.168.0.1"},
			},
			ForwardedHeaders: &static.ForwardedHeaders{
				Insecure:   true,
				TrustedIPs: []string{"127.0.0.1/32", "192.168.0.1"},
			},
			HTTP: static.HTTPConfig{
				Redirections: &static.Redirections{
					EntryPoint: &static.RedirectEntryPoint{
						To:        "foobar",
						Scheme:    "foobar",
						Permanent: true,
						Priority:  42,
					},
				},
				Middlewares: []string{"foobar", "foobar"},
				TLS: &static.TLSConfig{
					Options:      "foobar",
					CertResolver: "foobar",
					Domains: []types.Domain{
						{Main: "foobar", SANs: []string{"foobar", "foobar"}},
					},
				},
			},
		},
	}

	config.Providers = &static.Providers{
		ProvidersThrottleDuration: ptypes.Duration(111 * time.Second),
	}

	config.ServersTransport = &static.ServersTransport{
		InsecureSkipVerify:  true,
		RootCAs:             []traefiktls.FileOrContent{"RootCAs 1", "RootCAs 2", "RootCAs 3"},
		MaxIdleConnsPerHost: 111,
		ForwardingTimeouts: &static.ForwardingTimeouts{
			DialTimeout:           ptypes.Duration(111 * time.Second),
			ResponseHeaderTimeout: ptypes.Duration(111 * time.Second),
			IdleConnTimeout:       ptypes.Duration(111 * time.Second),
		},
	}

	config.Providers.File = &file.Provider{
		Directory:                 "file Directory",
		Watch:                     true,
		Filename:                  "file Filename",
		DebugLogGeneratedTemplate: true,
	}

	config.Providers.Docker = &docker.Provider{
		Constraints: `Label("foo", "bar")`,
		Watch:       true,
		Endpoint:    "MyEndPoint",
		DefaultRule: "PathPrefix(`/`)",
		TLS: &types.ClientTLS{
			CA:                 "myCa",
			CAOptional:         true,
			Cert:               "mycert.pem",
			Key:                "mycert.key",
			InsecureSkipVerify: true,
		},
		ExposedByDefault:        true,
		UseBindPortIP:           true,
		SwarmMode:               true,
		Network:                 "MyNetwork",
		SwarmModeRefreshSeconds: 42,
		HTTPClientTimeout:       42,
	}

	config.Providers.Marathon = &marathon.Provider{
		Constraints:      `Label("foo", "bar")`,
		Trace:            true,
		Watch:            true,
		Endpoint:         "foobar",
		DefaultRule:      "PathPrefix(`/`)",
		ExposedByDefault: true,
		DCOSToken:        "foobar",
		TLS: &types.ClientTLS{
			CA:                 "myCa",
			CAOptional:         true,
			Cert:               "mycert.pem",
			Key:                "mycert.key",
			InsecureSkipVerify: true,
		},
		DialerTimeout:         42,
		ResponseHeaderTimeout: 42,
		TLSHandshakeTimeout:   42,
		KeepAlive:             42,
		ForceTaskHostname:     true,
		Basic: &marathon.Basic{
			HTTPBasicAuthUser: "user",
			HTTPBasicPassword: "password",
		},
		RespectReadinessChecks: true,
	}

	config.Providers.KubernetesIngress = &ingress.Provider{
		Endpoint:         "MyEndpoint",
		Token:            "MyToken",
		CertAuthFilePath: "MyCertAuthPath",
		Namespaces:       []string{"a", "b"},
		LabelSelector:    "myLabelSelector",
		IngressClass:     "MyIngressClass",
		IngressEndpoint: &ingress.EndpointIngress{
			IP:               "IP",
			Hostname:         "Hostname",
			PublishedService: "PublishedService",
		},
		ThrottleDuration: ptypes.Duration(111 * time.Second),
	}

	config.Providers.KubernetesCRD = &crd.Provider{
		Endpoint:         "MyEndpoint",
		Token:            "MyToken",
		CertAuthFilePath: "MyCertAuthPath",
		Namespaces:       []string{"a", "b"},
		LabelSelector:    "myLabelSelector",
		IngressClass:     "MyIngressClass",
		ThrottleDuration: ptypes.Duration(111 * time.Second),
	}

	config.Providers.KubernetesGateway = &gateway.Provider{
		Endpoint:         "MyEndpoint",
		Token:            "MyToken",
		CertAuthFilePath: "MyCertAuthPath",
		Namespaces:       []string{"a", "b"},
		LabelSelector:    "myLabelSelector",
		ThrottleDuration: ptypes.Duration(111 * time.Second),
	}

	config.Providers.Rest = &rest.Provider{
		Insecure: true,
	}

	config.Providers.Rancher = &rancher.Provider{
		Constraints:               `Label("foo", "bar")`,
		Watch:                     true,
		DefaultRule:               "PathPrefix(`/`)",
		ExposedByDefault:          true,
		EnableServiceHealthFilter: true,
		RefreshSeconds:            42,
		IntervalPoll:              true,
		Prefix:                    "MyPrefix",
	}

	config.Providers.ConsulCatalog = &consulcatalog.Provider{
		Constraints: `Label("foo", "bar")`,
		Endpoint: &consulcatalog.EndpointConfig{
			Address:    "MyAddress",
			Scheme:     "MyScheme",
			DataCenter: "MyDatacenter",
			Token:      "MyToken",
			TLS: &types.ClientTLS{
				CA:                 "myCa",
				CAOptional:         true,
				Cert:               "mycert.pem",
				Key:                "mycert.key",
				InsecureSkipVerify: true,
			},
			HTTPAuth: &consulcatalog.EndpointHTTPAuthConfig{
				Username: "MyUsername",
				Password: "MyPassword",
			},
			EndpointWaitTime: 42,
		},
		Prefix:            "MyPrefix",
		RefreshInterval:   42,
		RequireConsistent: true,
		Stale:             true,
		Cache:             true,
		ExposedByDefault:  true,
		DefaultRule:       "PathPrefix(`/`)",
	}

	config.Providers.Ecs = &ecs.Provider{
		Constraints:          `Label("foo", "bar")`,
		ExposedByDefault:     true,
		RefreshSeconds:       42,
		DefaultRule:          "PathPrefix(`/`)",
		Clusters:             []string{"Cluster1", "Cluster2"},
		AutoDiscoverClusters: true,
		Region:               "Awsregion",
		AccessKeyID:          "AwsAccessKeyID",
		SecretAccessKey:      "AwsSecretAccessKey",
	}

	config.Providers.Consul = &consul.Provider{
		Provider: kv.Provider{
			RootKey:   "RootKey",
			Endpoints: nil,
			Username:  "username",
			Password:  "password",
			TLS: &types.ClientTLS{
				CA:                 "myCa",
				CAOptional:         true,
				Cert:               "mycert.pem",
				Key:                "mycert.key",
				InsecureSkipVerify: true,
			},
		},
	}

	config.Providers.Etcd = &etcd.Provider{
		Provider: kv.Provider{
			RootKey:   "RootKey",
			Endpoints: nil,
			Username:  "username",
			Password:  "password",
			TLS: &types.ClientTLS{
				CA:                 "myCa",
				CAOptional:         true,
				Cert:               "mycert.pem",
				Key:                "mycert.key",
				InsecureSkipVerify: true,
			},
		},
	}

	config.Providers.ZooKeeper = &zk.Provider{
		Provider: kv.Provider{
			RootKey:   "RootKey",
			Endpoints: nil,
			Username:  "username",
			Password:  "password",
			TLS: &types.ClientTLS{
				CA:                 "myCa",
				CAOptional:         true,
				Cert:               "mycert.pem",
				Key:                "mycert.key",
				InsecureSkipVerify: true,
			},
		},
	}

	config.Providers.Redis = &redis.Provider{
		Provider: kv.Provider{
			RootKey:   "RootKey",
			Endpoints: nil,
			Username:  "username",
			Password:  "password",
			TLS: &types.ClientTLS{
				CA:                 "myCa",
				CAOptional:         true,
				Cert:               "mycert.pem",
				Key:                "mycert.key",
				InsecureSkipVerify: true,
			},
		},
	}

	config.Providers.HTTP = &http.Provider{
		Endpoint:     "Myenpoint",
		PollInterval: 42,
		PollTimeout:  42,
		TLS: &types.ClientTLS{
			CA:                 "myCa",
			CAOptional:         true,
			Cert:               "mycert.pem",
			Key:                "mycert.key",
			InsecureSkipVerify: true,
		},
	}

	config.API = &static.API{
		Insecure:  true,
		Dashboard: true,
		Debug:     true,
		DashboardAssets: &assetfs.AssetFS{
			Asset: func(path string) ([]byte, error) {
				return nil, nil
			},
			AssetDir: func(path string) ([]string, error) {
				return nil, nil
			},
			AssetInfo: func(path string) (os.FileInfo, error) {
				return nil, nil
			},
			Prefix: "fii",
		},
	}

	config.Metrics = &types.Metrics{
		Prometheus: &types.Prometheus{
			Buckets:              []float64{0.1, 0.3, 1.2, 5},
			AddEntryPointsLabels: true,
			AddServicesLabels:    true,
			EntryPoint:           "MyEntryPoint",
			ManualRouting:        true,
		},
		Datadog: &types.Datadog{
			Address:              "localhost:8181",
			PushInterval:         42,
			AddEntryPointsLabels: true,
			AddServicesLabels:    true,
		},
		StatsD: &types.Statsd{
			Address:              "localhost:8182",
			PushInterval:         42,
			AddEntryPointsLabels: true,
			AddServicesLabels:    true,
			Prefix:               "MyPrefix",
		},
		InfluxDB: &types.InfluxDB{
			Address:              "localhost:8183",
			Protocol:             "http",
			PushInterval:         42,
			Database:             "myDB",
			RetentionPolicy:      "12",
			Username:             "a",
			Password:             "aaaa",
			AddEntryPointsLabels: true,
			AddServicesLabels:    true,
		},
	}

	config.Ping = &ping.Handler{
		EntryPoint:            "MyEntryPoint",
		ManualRouting:         true,
		TerminatingStatusCode: 42,
	}

	config.Log = &types.TraefikLog{
		Level:    "Level",
		FilePath: "/foo/path",
		Format:   "json",
	}

	config.AccessLog = &types.AccessLog{
		FilePath: "AccessLog FilePath",
		Format:   "AccessLog Format",
		Filters: &types.AccessLogFilters{
			StatusCodes:   []string{"200", "500"},
			RetryAttempts: true,
			MinDuration:   42,
		},
		Fields: &types.AccessLogFields{
			DefaultMode: "drop",
			Names: map[string]string{
				"RequestHost": "keep",
			},
			Headers: &types.FieldHeaders{
				DefaultMode: "drop",
				Names: map[string]string{
					"Referer": "keep",
				},
			},
		},
		BufferingSize: 42,
	}

	config.Tracing = &static.Tracing{
		ServiceName:   "myServiceName",
		SpanNameLimit: 42,
		Jaeger: &jaeger.Config{
			SamplingServerURL:      "foobar",
			SamplingType:           "foobar",
			SamplingParam:          42,
			LocalAgentHostPort:     "foobar",
			Gen128Bit:              true,
			Propagation:            "foobar",
			TraceContextHeaderName: "foobar",
			Collector: &jaeger.Collector{
				Endpoint: "foobar",
				User:     "foobar",
				Password: "foobar",
			},
			DisableAttemptReconnecting: true,
		},
		Zipkin: &zipkin.Config{
			HTTPEndpoint: "foobar",
			SameSpan:     true,
			ID128Bit:     true,
			SampleRate:   42,
		},
		Datadog: &datadog.Config{
			LocalAgentHostPort:         "foobar",
			GlobalTag:                  "foobar",
			Debug:                      true,
			PrioritySampling:           true,
			TraceIDHeaderName:          "foobar",
			ParentIDHeaderName:         "foobar",
			SamplingPriorityHeaderName: "foobar",
			BagagePrefixHeaderName:     "foobar",
		},
		Instana: &instana.Config{
			LocalAgentHost: "foobar",
			LocalAgentPort: 4242,
			LogLevel:       "foobar",
		},
		Haystack: &haystack.Config{
			LocalAgentHost:          "foobar",
			LocalAgentPort:          42,
			GlobalTag:               "foobar",
			TraceIDHeaderName:       "foobar",
			ParentIDHeaderName:      "foobar",
			SpanIDHeaderName:        "foobar",
			BaggagePrefixHeaderName: "foobar",
		},
		Elastic: &elastic.Config{
			ServerURL:          "foobar",
			SecretToken:        "foobar",
			ServiceEnvironment: "foobar",
		},
	}

	config.HostResolver = &types.HostResolverConfig{
		CnameFlattening: true,
		ResolvConfig:    "foobar",
		ResolvDepth:     42,
	}

	config.CertificatesResolvers = map[string]static.CertificateResolver{
		"CertificateResolver0": {
			ACME: &acme.Configuration{
				Email:          "acme Email",
				CAServer:       "CAServer",
				PreferredChain: "foobar",
				Storage:        "Storage",
				KeyType:        "MyKeyType",
				DNSChallenge: &acme.DNSChallenge{
					Provider:                "DNSProvider",
					DelayBeforeCheck:        42,
					Resolvers:               []string{"resolver1", "resolver2"},
					DisablePropagationCheck: true,
				},
				HTTPChallenge: &acme.HTTPChallenge{
					EntryPoint: "MyEntryPoint",
				},
				TLSChallenge: &acme.TLSChallenge{},
			},
		},
	}

	config.Pilot = &static.Pilot{
		Token: "token",
	}

	config.Experimental = &static.Experimental{
		Plugins: map[string]plugins.Descriptor{
			"Descriptor0": {
				ModuleName: "foobar",
				Version:    "foobar",
			},
			"Descriptor1": {
				ModuleName: "foobar",
				Version:    "foobar",
			},
		},
		LocalPlugins: map[string]plugins.LocalDescriptor{
			"Descriptor0": {
				ModuleName: "foobar",
			},
			"Descriptor1": {
				ModuleName: "foobar",
			},
		},
	}

	expectedConfiguration, err := os.ReadFile("./testdata/anonymized-static-config.json")
	require.NoError(t, err)

	cleanJSON, err := Do(config, true)
	require.NoError(t, err)

	if *updateExpected {
		require.NoError(t, os.WriteFile("testdata/anonymized-static-config.json", []byte(cleanJSON), 0666))
	}

	expected := strings.TrimSuffix(string(expectedConfiguration), "\n")
	assert.Equal(t, expected, cleanJSON)
}

func boolPtr(value bool) *bool {
	return &value
}

func intPtr(value int) *int {
	return &value
}

func int64Ptr(value int64) *int64 {
	return &value
}
