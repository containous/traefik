package kubernetes

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/tls"
	"github.com/traefik/traefik/types"
	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestProvider_loadIngresses(t *testing.T) {
	testCases := []struct {
		desc     string
		provider Provider
		fixtures []string
		expected *types.Configuration
	}{
		{
			desc: "simple",
			fixtures: []string{
				filepath.Join("fixtures", "loadIngresses_ingresses.yml"),
				filepath.Join("fixtures", "loadIngresses_services.yml"),
				filepath.Join("fixtures", "loadIngresses_endpoints.yml"),
			},
			expected: buildConfiguration(
				backends(
					backend("foo/bar",
						lbMethod("wrr"),
						servers(
							server("http://10.10.0.1:8080", weight(1)),
							server("http://10.21.0.1:8080", weight(1))),
					),
					backend("foo/namedthing",
						lbMethod("wrr"),
						servers(
							server("https://example.com", weight(1)),
						),
					),
					backend("bar",
						lbMethod("wrr"),
						servers(
							server("https://10.15.0.1:8443", weight(1)),
							server("https://10.15.0.2:9443", weight(1)),
						),
					),
					backend("service5",
						lbMethod("wrr"),
						servers(
							server("http://example.com:8888", weight(1)),
						),
					),
					backend("service6",
						lbMethod("wrr"),
						servers(
							server("http://10.15.0.3:80", weight(1)),
						),
					),
					backend("*.service7",
						lbMethod("wrr"),
						servers(
							server("http://10.10.0.7:80", weight(1)),
						),
					),
					backend("service8",
						lbMethod("wrr"),
						servers(
							server("http://10.10.0.8:80", weight(1)),
						),
					),
				),
				frontends(
					frontend("foo/bar",
						passHostHeader(),
						routes(
							route("/bar", "PathPrefix:/bar"),
							route("foo", "Host:foo")),
					),
					frontend("foo/namedthing",
						passHostHeader(),
						routes(
							route("/namedthing", "PathPrefix:/namedthing"),
							route("foo", "Host:foo")),
					),
					frontend("bar",
						passHostHeader(),
						routes(route("bar", "Host:bar")),
					),
					frontend("service5",
						passHostHeader(),
						routes(route("service5", "Host:service5")),
					),
					frontend("service6",
						passHostHeader(),
						routes(route("service6", "Host:service6")),
					),
					frontend("*.service7",
						passHostHeader(),
						routes(route("*.service7", "HostRegexp:{subdomain:[A-Za-z0-9-_]+}.service7")),
					),
					frontend("service8",
						passHostHeader(),
						routes(route("/", "PathPrefix:/")),
					),
				),
			),
		},
		{
			desc: "loadGlobalIngressWithExternalName",
			fixtures: []string{
				filepath.Join("fixtures", "loadGlobalIngressWithExternalName_ingresses.yml"),
				filepath.Join("fixtures", "loadGlobalIngressWithExternalName_services.yml"),
			},
			expected: buildConfiguration(
				backends(
					backend("global-default-backend",
						lbMethod("wrr"),
						servers(
							server("http://some-external-name", weight(1)),
						),
					),
				),
				frontends(
					frontend("global-default-backend",
						frontendName("global-default-frontend"),
						passHostHeader(),
						routes(
							route("/", "PathPrefix:/"),
						),
					),
				),
			),
		},
		{
			desc: "loadGlobalIngressWithPortNumbers",
			fixtures: []string{
				filepath.Join("fixtures", "loadGlobalIngressWithPortNumbers_ingresses.yml"),
				filepath.Join("fixtures", "loadGlobalIngressWithPortNumbers_services.yml"),
				filepath.Join("fixtures", "loadGlobalIngressWithPortNumbers_endpoints.yml"),
			},
			expected: buildConfiguration(
				backends(
					backend("global-default-backend",
						lbMethod("wrr"),
						servers(
							server("http://10.10.0.1:8080", weight(1)),
						),
					),
				),
				frontends(
					frontend("global-default-backend",
						frontendName("global-default-frontend"),
						passHostHeader(),
						routes(
							route("/", "PathPrefix:/"),
						),
					),
				),
			),
		},
		{
			desc: "loadGlobalIngressWithMultiplePortNumbers",
			fixtures: []string{
				filepath.Join("fixtures", "loadGlobalIngressWithMultiplePortNumbers_ingresses.yml"),
				filepath.Join("fixtures", "loadGlobalIngressWithMultiplePortNumbers_services.yml"),
				filepath.Join("fixtures", "loadGlobalIngressWithMultiplePortNumbers_endpoints.yml"),
			},
			expected: buildConfiguration(
				backends(
					backend("global-default-backend",
						lbMethod("wrr"),
						servers(
							server("http://10.10.0.1:8080", weight(1)),
						),
					),
				),
				frontends(
					frontend("global-default-backend",
						frontendName("global-default-frontend"),
						passHostHeader(),
						routes(
							route("/", "PathPrefix:/"),
						),
					),
				),
			),
		},
		{
			desc: "loadGlobalIngressWithHttpsPortNames",
			fixtures: []string{
				filepath.Join("fixtures", "loadGlobalIngressWithHttpsPortNames_ingresses.yml"),
				filepath.Join("fixtures", "loadGlobalIngressWithHttpsPortNames_services.yml"),
				filepath.Join("fixtures", "loadGlobalIngressWithHttpsPortNames_endpoints.yml"),
			},
			expected: buildConfiguration(
				backends(
					backend("global-default-backend",
						lbMethod("wrr"),
						servers(
							server("https://10.10.0.1:8080", weight(1)),
						),
					),
				),
				frontends(
					frontend("global-default-backend",
						frontendName("global-default-frontend"),
						passHostHeader(),
						routes(
							route("/", "PathPrefix:/"),
						),
					),
				),
			),
		},
		{
			desc: "getPassHostHeader",
			fixtures: []string{
				filepath.Join("fixtures", "getPassHostHeader_ingresses.yml"),
				filepath.Join("fixtures", "getPassHostHeader_services.yml"),
			},
			provider: Provider{DisablePassHostHeaders: true},
			expected: buildConfiguration(
				backends(backend("foo/bar", lbMethod("wrr"), servers())),
				frontends(
					frontend("foo/bar",
						routes(
							route("/bar", "PathPrefix:/bar"),
							route("foo", "Host:foo")),
					),
				),
			),
		},
		{
			desc: "getPassTLSCert", // Deprecated
			fixtures: []string{
				filepath.Join("fixtures", "getPassTLSCert_ingresses.yml"),
				filepath.Join("fixtures", "getPassTLSCert_services.yml"),
			},
			provider: Provider{EnablePassTLSCert: true},
			expected: buildConfiguration(
				backends(backend("foo/bar", lbMethod("wrr"), servers())),
				frontends(frontend("foo/bar",
					passHostHeader(),
					passTLSCert(),
					routes(
						route("/bar", "PathPrefix:/bar"),
						route("foo", "Host:foo")),
				)),
			),
		},
		{
			desc: "onlyReferencesServicesFromOwnNamespace",
			fixtures: []string{
				filepath.Join("fixtures", "onlyReferencesServicesFromOwnNamespace_ingresses.yml"),
				filepath.Join("fixtures", "onlyReferencesServicesFromOwnNamespace_services.yml"),
			},
			expected: buildConfiguration(
				backends(backend("foo", lbMethod("wrr"), servers())),
				frontends(frontend("foo",
					passHostHeader(),
					routes(route("foo", "Host:foo")),
				)),
			),
		},
		{
			desc: "hostlessIngress",
			fixtures: []string{
				filepath.Join("fixtures", "hostlessIngress_ingresses.yml"),
				filepath.Join("fixtures", "hostlessIngress_services.yml"),
			},
			provider: Provider{DisablePassHostHeaders: true},
			expected: buildConfiguration(
				backends(backend("/bar", lbMethod("wrr"), servers())),
				frontends(frontend("/bar",
					routes(route("/bar", "PathPrefix:/bar")))),
			),
		},
		{
			desc: "serviceAnnotations",
			fixtures: []string{
				filepath.Join("fixtures", "serviceAnnotations_ingresses.yml"),
				filepath.Join("fixtures", "serviceAnnotations_services.yml"),
				filepath.Join("fixtures", "serviceAnnotations_endpoints.yml"),
			},
			expected: buildConfiguration(
				backends(
					backend("foo/bar",
						servers(
							server("http://10.10.0.1:8080", weight(1)),
							server("http://10.21.0.1:8080", weight(1))),
						lbMethod("drr"),
						circuitBreaker("NetworkErrorRatio() > 0.5"),
					),
					backend("flush",
						servers(),
						lbMethod("wrr"),
						responseForwarding("10ms"),
					),
					backend("bar",
						servers(
							server("http://10.15.0.1:8080", weight(1)),
							server("http://10.15.0.2:8080", weight(1))),
						lbMethod("wrr"), lbSticky(),
					),
					backend("baz",
						servers(
							server("http://10.14.0.1:8080", weight(1)),
							server("http://10.12.0.1:8080", weight(1))),
						lbMethod("wrr"),
						buffering(
							maxRequestBodyBytes(10485760),
							memRequestBodyBytes(2097153),
							maxResponseBodyBytes(10485761),
							memResponseBodyBytes(2097152),
							retrying("IsNetworkError() && Attempts() <= 2"),
						),
					),
					backend("max-conn",
						servers(
							server("http://10.4.0.1:8080", weight(1)),
							server("http://10.4.0.2:8080", weight(1))),
						maxConnExtractorFunc("client.ip"),
						maxConnAmount(6),
						lbMethod("wrr"),
					),
				),
				frontends(
					frontend("foo/bar",
						passHostHeader(),
						routes(
							route("/bar", "PathPrefix:/bar"),
							route("foo", "Host:foo")),
					),
					frontend("bar",
						passHostHeader(),
						routes(route("bar", "Host:bar"))),
					frontend("baz",
						passHostHeader(),
						routes(route("baz", "Host:baz"))),
					frontend("max-conn",
						passHostHeader(),
						routes(
							route("max-conn", "Host:max-conn"))),
					frontend("flush",
						passHostHeader(),
						routes(
							route("flush", "Host:flush"))),
				),
			),
		},
		{
			desc: "ingressAnnotations",
			fixtures: []string{
				filepath.Join("fixtures", "ingressAnnotations_ingresses.yml"),
				filepath.Join("fixtures", "ingressAnnotations_services.yml"),
				filepath.Join("fixtures", "ingressAnnotations_secrets.yml"),
			},
			expected: buildConfiguration(
				backends(
					backend("foo/bar",
						servers(
							server("http://example.com", weight(1)),
							server("http://example.com", weight(1))),
						lbMethod("wrr"),
					),
					backend("other/stuff",
						servers(
							server("http://example.com", weight(1)),
							server("http://example.com", weight(1))),
						lbMethod("wrr"),
					),
					backend("http-https_other/",
						servers(
							server("http://example.com", weight(1)),
							server("http://example.com", weight(1))),
						lbMethod("wrr"),
					),
					backend("other/sslstuff",
						servers(
							server("http://example.com", weight(1)),
							server("http://example.com", weight(1))),
						lbMethod("wrr"),
					),
					backend("basic/auth",
						servers(
							server("http://example.com", weight(1)),
							server("http://example.com", weight(1))),
						lbMethod("wrr"),
					),
					backend("redirect/https",
						servers(
							server("http://example.com", weight(1)),
							server("http://example.com", weight(1))),
						lbMethod("wrr"),
					),
					backend("test/whitelist-source-range",
						servers(
							server("http://example.com", weight(1)),
							server("http://example.com", weight(1))),
						lbMethod("wrr"),
					),
					backend("rewrite/api",
						servers(
							server("http://example.com", weight(1)),
							server("http://example.com", weight(1))),
						lbMethod("wrr"),
					),
					backend("rewritetargetrootpath/",
						servers(
							server("http://example.com", weight(1)),
							server("http://example.com", weight(1))),
						lbMethod("wrr"),
					),
					backend("error-pages/errorpages",
						servers(
							server("http://example.com", weight(1)),
							server("http://example.com", weight(1))),
						lbMethod("wrr"),
					),
					backend("rate-limit/ratelimit",
						servers(
							server("http://example.com", weight(1)),
							server("http://example.com", weight(1))),
						lbMethod("wrr"),
					),
					backend("custom-headers/customheaders",
						servers(
							server("http://example.com", weight(1)),
							server("http://example.com", weight(1))),
						lbMethod("wrr"),
					),
					backend("root/",
						servers(
							server("http://example.com", weight(1))),
						lbMethod("wrr"),
					),
					backend("root/root1",
						servers(
							server("http://example.com", weight(1))),
						lbMethod("wrr"),
					),
					backend("root2/",
						servers(),
						lbMethod("wrr"),
					),
					backend("root3",
						servers(
							server("http://example.com", weight(1))),
						lbMethod("wrr"),
					),
					backend("protocol/valid",
						servers(
							server("h2c://example.com", weight(1)),
							server("h2c://example.com", weight(1))),
						lbMethod("wrr"),
					),
					backend("protocol/notvalid",
						servers(),
						lbMethod("wrr"),
					),
					backend("protocol/missmatch",
						servers(
							server("http://example.com", weight(1)),
							server("http://example.com", weight(1))),
						lbMethod("wrr"),
					),
					backend("protocol/noAnnotation",
						servers(
							server("https://example.com", weight(1)),
							server("https://example.com", weight(1))),
						lbMethod("wrr"),
					),
				),
				frontends(
					frontend("foo/bar",
						routes(
							route("/bar", "PathPrefix:/bar"),
							route("foo", "Host:foo")),
					),
					frontend("other/stuff",
						passHostHeader(),
						routes(
							route("/stuff", "PathPrefix:/stuff"),
							route("other", "Host:other")),
					),
					frontend("http-https_other/",
						passHostHeader(),
						entryPoints("http", "https"),
						routes(
							route("/", "PathPrefix:/"),
							route("other", "Host:other")),
					),
					frontend("other/sslstuff",
						passHostHeader(),
						passTLSClientCert(),
						passTLSCert(),
						routes(
							route("/sslstuff", "PathPrefix:/sslstuff"),
							route("other", "Host:other")),
					),
					frontend("basic/auth",
						passHostHeader(),
						basicAuthDeprecated("myUser:myEncodedPW"),
						routes(
							route("/auth", "PathPrefix:/auth"),
							route("basic", "Host:basic")),
					),
					frontend("redirect/https",
						passHostHeader(),
						redirectEntryPoint("https"),
						routes(
							route("/https", "PathPrefix:/https"),
							route("redirect", "Host:redirect")),
					),
					frontend("test/whitelist-source-range",
						passHostHeader(),
						whiteList(true, "1.1.1.1/24", "1234:abcd::42/32"),
						routes(
							route("/whitelist-source-range", "PathPrefix:/whitelist-source-range"),
							route("test", "Host:test")),
					),
					frontend("rewritetargetrootpath/",
						passHostHeader(),
						routes(
							route("/", "PathPrefix:/;ReplacePathRegex: ^(.*) /app$1"),
							route("rewritetargetrootpath", "Host:rewritetargetrootpath")),
					),
					frontend("rewrite/api",
						passHostHeader(),
						routes(
							route("/api", "PathPrefix:/api;ReplacePathRegex: ^/api(.*) $1"),
							route("rewrite", "Host:rewrite")),
					),
					frontend("error-pages/errorpages",
						passHostHeader(),
						errorPage("foo", errorQuery("/bar"), errorStatus("123", "456"), errorBackend("bar")),
						routes(
							route("/errorpages", "PathPrefix:/errorpages"),
							route("error-pages", "Host:error-pages")),
					),
					frontend("rate-limit/ratelimit",
						passHostHeader(),
						rateLimit(rateExtractorFunc("client.ip"),
							rateSet("foo", limitPeriod(6*time.Second), limitAverage(12), limitBurst(18)),
							rateSet("bar", limitPeriod(3*time.Second), limitAverage(6), limitBurst(9))),
						routes(
							route("/ratelimit", "PathPrefix:/ratelimit"),
							route("rate-limit", "Host:rate-limit")),
					),
					frontend("custom-headers/customheaders",
						passHostHeader(),
						headers(&types.Headers{
							CustomRequestHeaders: map[string]string{
								"Access-Control-Allow-Methods": "POST,GET,OPTIONS",
								"Content-Type":                 "application/json; charset=utf-8",
							},
							CustomResponseHeaders: map[string]string{
								"Access-Control-Allow-Methods": "POST,GET,OPTIONS",
								"Content-Type":                 "application/json; charset=utf-8",
							},
							SSLProxyHeaders: map[string]string{
								"Access-Control-Allow-Methods": "POST,GET,OPTIONS",
								"Content-Type":                 "application/json; charset=utf-8",
							},
							AllowedHosts:            []string{"foo", "fii", "fuu"},
							HostsProxyHeaders:       []string{"foo", "fii", "fuu"},
							STSSeconds:              666,
							SSLForceHost:            true,
							SSLRedirect:             true,
							SSLTemporaryRedirect:    true,
							STSIncludeSubdomains:    true,
							STSPreload:              true,
							ForceSTSHeader:          true,
							FrameDeny:               true,
							ContentTypeNosniff:      true,
							BrowserXSSFilter:        true,
							IsDevelopment:           true,
							CustomFrameOptionsValue: "foo",
							SSLHost:                 "foo",
							ContentSecurityPolicy:   "foo",
							PublicKey:               "foo",
							ReferrerPolicy:          "foo",
							CustomBrowserXSSValue:   "foo",
						}),
						routes(
							route("/customheaders", "PathPrefix:/customheaders"),
							route("custom-headers", "Host:custom-headers")),
					),
					frontend("root/",
						passHostHeader(),
						redirectRegex("root/$", "root/root"),
						routes(
							route("/", "PathPrefix:/"),
							route("root", "Host:root"),
						),
					),
					frontend("root2/",
						passHostHeader(),
						redirectRegex("root2/$", "root2/root2"),
						routes(
							route("/", "PathPrefix:/;ReplacePathRegex: ^(.*) /abc$1"),
							route("root2", "Host:root2"),
						),
					),
					frontend("root/root1",
						passHostHeader(),
						routes(
							route("/root1", "PathPrefix:/root1"),
							route("root", "Host:root"),
						),
					),
					frontend("root3",
						passHostHeader(),
						redirectRegex("root3/$", "root3/root"),
						routes(
							route("root3", "Host:root3"),
						),
					),
					frontend("protocol/valid",
						passHostHeader(),
						routes(
							route("/valid", "PathPrefix:/valid"),
							route("protocol", "Host:protocol"),
						),
					),
					frontend("protocol/notvalid",
						passHostHeader(),
						routes(
							route("/notvalid", "PathPrefix:/notvalid"),
							route("protocol", "Host:protocol"),
						),
					),
					frontend("protocol/missmatch",
						passHostHeader(),
						routes(
							route("/missmatch", "PathPrefix:/missmatch"),
							route("protocol", "Host:protocol"),
						),
					),
					frontend("protocol/noAnnotation",
						passHostHeader(),
						routes(
							route("/noAnnotation", "PathPrefix:/noAnnotation"),
							route("protocol", "Host:protocol"),
						),
					),
				),
			),
		},
		{
			desc: "priorityHeaderValue",
			fixtures: []string{
				filepath.Join("fixtures", "priorityHeaderValue_ingresses.yml"),
				filepath.Join("fixtures", "priorityHeaderValue_services.yml"),
			},
			expected: buildConfiguration(
				backends(
					backend("1337-foo/bar",
						servers(server("http://example.com", weight(1))),
						lbMethod("wrr"),
					),
				),
				frontends(
					frontend("1337-foo/bar",
						passHostHeader(),
						priority(1337),
						routes(
							route("/bar", "PathPrefix:/bar"),
							route("foo", "Host:foo")),
					),
				),
			),
		},
		{
			desc: "invalidPassTLSCertValue",
			fixtures: []string{
				filepath.Join("fixtures", "invalidPassTLSCertValue_ingresses.yml"),
				filepath.Join("fixtures", "invalidPassTLSCertValue_services.yml"),
			},
			expected: buildConfiguration(
				backends(
					backend("foo/bar",
						servers(server("http://example.com", weight(1))),
						lbMethod("wrr"),
					),
				),
				frontends(
					frontend("foo/bar",
						passHostHeader(),
						routes(
							route("/bar", "PathPrefix:/bar"),
							route("foo", "Host:foo")),
					),
				),
			),
		},
		{
			desc: "invalidPassHostHeaderValue",
			fixtures: []string{
				filepath.Join("fixtures", "invalidPassHostHeaderValue_ingresses.yml"),
				filepath.Join("fixtures", "invalidPassHostHeaderValue_services.yml"),
			},
			expected: buildConfiguration(
				backends(
					backend("foo/bar",
						servers(server("http://example.com", weight(1))),
						lbMethod("wrr"),
					),
				),
				frontends(
					frontend("foo/bar",
						passHostHeader(),
						routes(
							route("/bar", "PathPrefix:/bar"),
							route("foo", "Host:foo")),
					),
				),
			),
		},
		{
			desc: "missingResources",
			fixtures: []string{
				filepath.Join("fixtures", "missingResources_ingresses.yml"),
				filepath.Join("fixtures", "missingResources_services.yml"),
				filepath.Join("fixtures", "missingResources_endpoints.yml"),
			},
			expected: buildConfiguration(
				backends(
					backend("fully_working",
						servers(server("http://10.10.0.1:8080", weight(1))),
						lbMethod("wrr"),
					),
					backend("missing_service",
						servers(),
						lbMethod("wrr"),
					),
					backend("missing_endpoints",
						servers(),
						lbMethod("wrr"),
					),
					backend("missing_endpoint_subsets",
						servers(),
						lbMethod("wrr"),
					),
				),
				frontends(
					frontend("fully_working",
						passHostHeader(),
						routes(route("fully_working", "Host:fully_working")),
					),
					frontend("missing_endpoints",
						passHostHeader(),
						routes(route("missing_endpoints", "Host:missing_endpoints")),
					),
					frontend("missing_endpoint_subsets",
						passHostHeader(),
						routes(route("missing_endpoint_subsets", "Host:missing_endpoint_subsets")),
					),
				),
			),
		},
		{
			desc: "ForwardAuth",
			fixtures: []string{
				filepath.Join("fixtures", "loadIngressesForwardAuth_ingresses.yml"),
				filepath.Join("fixtures", "loadIngressesForwardAuth_services.yml"),
				filepath.Join("fixtures", "loadIngressesForwardAuth_endpoints.yml"),
			},
			expected: buildConfiguration(
				backends(
					backend("foo/bar",
						lbMethod("wrr"),
						servers(
							server("http://10.10.0.1:8080", weight(1))),
					),
				),
				frontends(
					frontend("foo/bar",
						passHostHeader(),
						auth(forwardAuth("https://auth.host",
							fwdTrustForwardHeader(),
							fwdAuthResponseHeaders("X-Auth", "X-Test", "X-Secret"))),
						routes(
							route("/bar", "PathPrefix:/bar"),
							route("foo", "Host:foo")),
					),
				),
			),
		},
		{
			desc: "ForwardAuthMissingURL",
			fixtures: []string{
				filepath.Join("fixtures", "loadIngressesForwardAuthMissingURL_ingresses.yml"),
				filepath.Join("fixtures", "loadIngressesForwardAuthMissingURL_services.yml"),
				filepath.Join("fixtures", "loadIngressesForwardAuthMissingURL_endpoints.yml"),
			},
			expected: buildConfiguration(
				backends(
					backend("foo/bar",
						lbMethod("wrr"),
						servers(),
					),
				),
				frontends(),
			),
		},
		{
			desc: "ForwardAuthWithTLSSecret",
			fixtures: []string{
				filepath.Join("fixtures", "loadIngressesForwardAuthWithTLSSecret_ingresses.yml"),
				filepath.Join("fixtures", "loadIngressesForwardAuthWithTLSSecret_services.yml"),
				filepath.Join("fixtures", "loadIngressesForwardAuthWithTLSSecret_endpoints.yml"),
				filepath.Join("fixtures", "loadIngressesForwardAuthWithTLSSecret_secrets.yml"),
			},
			expected: buildConfiguration(
				backends(
					backend("foo/bar",
						lbMethod("wrr"),
						servers(
							server("http://10.10.0.1:8080", weight(1))),
					),
				),
				frontends(
					frontend("foo/bar",
						passHostHeader(),
						auth(
							forwardAuth("https://auth.host",
								fwdAuthTLS(
									"-----BEGIN CERTIFICATE-----\n-----END CERTIFICATE-----",
									"-----BEGIN PRIVATE KEY-----\n-----END PRIVATE KEY-----",
									true))),
						routes(
							route("/bar", "PathPrefix:/bar"),
							route("foo", "Host:foo")),
					),
				),
			),
		},
		{
			desc: "tLSSecretLoad",
			fixtures: []string{
				filepath.Join("fixtures", "tLSSecretLoad_ingresses.yml"),
				filepath.Join("fixtures", "tLSSecretLoad_services.yml"),
				filepath.Join("fixtures", "tLSSecretLoad_secrets.yml"),
			},
			expected: buildConfiguration(
				backends(
					backend("ep1-ep2_example.com",
						servers(),
						lbMethod("wrr"),
					),
					backend("ep1-ep2_example.org",
						servers(),
						lbMethod("wrr"),
					),
				),
				frontends(
					frontend("ep1-ep2_example.com",
						entryPoints("ep1", "ep2"),
						passHostHeader(),
						routes(
							route("example.com", "Host:example.com"),
						),
					),
					frontend("ep1-ep2_example.org",
						entryPoints("ep1", "ep2"),
						passHostHeader(),
						routes(
							route("example.org", "Host:example.org"),
						),
					),
				),
				tlsesSection(
					tlsSection(
						tlsEntryPoints("ep1", "ep2"),
						certificate(
							"-----BEGIN CERTIFICATE-----\n-----END CERTIFICATE-----",
							"-----BEGIN PRIVATE KEY-----\n-----END PRIVATE KEY-----"),
					),
				),
			),
		},
		{
			desc: "multiPortServices",
			fixtures: []string{
				filepath.Join("fixtures", "multiPortServices_ingresses.yml"),
				filepath.Join("fixtures", "multiPortServices_services.yml"),
				filepath.Join("fixtures", "multiPortServices_endpoints.yml"),
			},
			expected: buildConfiguration(
				backends(
					backend("/cheddar",
						lbMethod("wrr"),
						servers(
							server("http://10.10.0.1:8080", weight(1)),
							server("http://10.10.0.2:8080", weight(1)),
						),
					),
					backend("/stilton",
						lbMethod("wrr"),
						servers(
							server("http://10.20.0.1:8081", weight(1)),
							server("http://10.20.0.2:8081", weight(1)),
						),
					),
				),
				frontends(
					frontend("/cheddar",
						passHostHeader(),
						routes(route("/cheddar", "PathPrefix:/cheddar")),
					),
					frontend("/stilton",
						passHostHeader(),
						routes(route("/stilton", "PathPrefix:/stilton")),
					),
				),
			),
		},
		{
			desc: "percentageWeightServiceAnnotation",
			fixtures: []string{
				filepath.Join("fixtures", "percentageWeightServiceAnnotation_ingresses.yml"),
				filepath.Join("fixtures", "percentageWeightServiceAnnotation_services.yml"),
				filepath.Join("fixtures", "percentageWeightServiceAnnotation_endpoints.yml"),
			},
			expected: buildConfiguration(
				backends(
					backend("host1/foo",
						servers(
							server("http://10.10.0.1:8080", weight(int(newPercentageValueFromFloat64(0.05)))),
							server("http://10.10.0.2:8080", weight(int(newPercentageValueFromFloat64(0.05)))),
							server("http://10.10.0.3:7070", weight(int(newPercentageValueFromFloat64(0.35)))),
							server("http://10.10.0.4:7070", weight(int(newPercentageValueFromFloat64(0.35)))),
							server("http://example.com:9090", weight(int(newPercentageValueFromFloat64(0.2)))),
						),
						lbMethod("wrr"),
					),
					backend("host1/bar",
						servers(
							server("http://10.10.0.3:7070", weight(int(newPercentageValueFromFloat64(0.5)))),
							server("http://10.10.0.4:7070", weight(int(newPercentageValueFromFloat64(0.5)))),
						),
						lbMethod("wrr"),
					),
				),
				frontends(
					frontend("host1/bar",
						passHostHeader(),
						routes(
							route("/bar", "PathPrefix:/bar"),
							route("host1", "Host:host1")),
					),
					frontend("host1/foo",
						passHostHeader(),
						routes(
							route("/foo", "PathPrefix:/foo"),
							route("host1", "Host:host1")),
					),
				),
			),
		},
		{
			desc: "templateBreakingIngresssValues",
			fixtures: []string{
				filepath.Join("fixtures", "templateBreakingIngresssValues_ingresses.yml"),
			},
			expected: buildConfiguration(
				backends(),
				frontends(),
			),
		},
		{
			desc: "divergingIngressDefinitions",
			fixtures: []string{
				filepath.Join("fixtures", "divergingIngressDefinitions_ingresses.yml"),
				filepath.Join("fixtures", "divergingIngressDefinitions_services.yml"),
				filepath.Join("fixtures", "divergingIngressDefinitions_endpoints.yml"),
			},
			expected: buildConfiguration(
				backends(
					backend("host-a",
						servers(
							server("http://10.10.0.1:80", weight(1)),
							server("http://10.10.0.2:80", weight(1)),
						),
						lbMethod("wrr"),
					),
				),
				frontends(
					frontend("host-a",
						passHostHeader(),
						routes(
							route("host-a", "Host:host-a")),
					),
				),
			),
		},
		{
			desc:     "Empty IngressClass annotation",
			provider: Provider{},
			fixtures: []string{
				filepath.Join("fixtures", "ingressClassAnnotation_ingresses.yml"),
				filepath.Join("fixtures", "ingressClassAnnotation_services.yml"),
				filepath.Join("fixtures", "ingressClassAnnotation_endpoints.yml"),
			},
			expected: buildConfiguration(
				backends(
					backend("other/stuff",
						servers(
							server("http://example.com", weight(1)),
							server("http://example.com", weight(1))),
						lbMethod("wrr"),
					),
					backend("other/",
						servers(
							server("http://example.com", weight(1)),
							server("http://example.com", weight(1))),
						lbMethod("wrr"),
					),
					backend("other/sslstuff",
						servers(
							server("http://example.com", weight(1)),
							server("http://example.com", weight(1))),
						lbMethod("wrr"),
					),
				),
				frontends(
					frontend("other/stuff",
						passHostHeader(),
						routes(
							route("/stuff", "PathPrefix:/stuff"),
							route("other", "Host:other")),
					),
					frontend("other/",
						passHostHeader(),
						routes(
							route("/", "PathPrefix:/"),
							route("other", "Host:other")),
					),
					frontend("other/sslstuff",
						passHostHeader(),
						routes(
							route("/sslstuff", "PathPrefix:/sslstuff"),
							route("other", "Host:other")),
					),
				),
			),
		},
		{
			desc:     "Provided IngressClass annotation",
			provider: Provider{IngressClass: traefikDefaultRealm + "-other"},
			fixtures: []string{
				filepath.Join("fixtures", "ingressClassAnnotation_ingresses.yml"),
				filepath.Join("fixtures", "ingressClassAnnotation_services.yml"),
				filepath.Join("fixtures", "ingressClassAnnotation_endpoints.yml"),
			},
			expected: buildConfiguration(
				backends(
					backend("foo/bar",
						servers(
							server("http://example.com", weight(1))),
						lbMethod("wrr"),
					),
				),
				frontends(
					frontend("foo/bar",
						passHostHeader(),
						routes(
							route("/bar", "PathPrefix:/bar"),
							route("foo", "Host:foo")),
					),
				),
			),
		},
		{
			desc:     "Provided IngressClass annotation",
			provider: Provider{IngressClass: "custom"},
			fixtures: []string{
				filepath.Join("fixtures", "ingressClassAnnotation_ingresses.yml"),
				filepath.Join("fixtures", "ingressClassAnnotation_services.yml"),
				filepath.Join("fixtures", "ingressClassAnnotation_endpoints.yml"),
			},
			expected: buildConfiguration(
				backends(
					backend("foo/bar",
						servers(
							server("http://10.10.0.1:80", weight(1))),
						lbMethod("wrr"),
					),
				),
				frontends(
					frontend("foo/bar",
						passHostHeader(),
						routes(
							route("/bar", "PathPrefix:/bar"),
							route("foo", "Host:foo")),
					),
				),
			),
		},
		{
			desc:     "BasicAuth",
			provider: Provider{},
			fixtures: []string{
				filepath.Join("fixtures", "loadIngressesBasicAuth_ingresses.yml"),
				filepath.Join("fixtures", "loadIngressesBasicAuth_services.yml"),
				filepath.Join("fixtures", "loadIngressesBasicAuth_secrets.yml"),
			},
			expected: buildConfiguration(
				backends(
					backend("basic/auth",
						servers(
							server("http://example.com", weight(1))),
						lbMethod("wrr"),
					),
				),
				frontends(
					frontend("basic/auth",
						auth(basicAuth(baUsers("myUser:myEncodedPW"), baRemoveHeaders())),
						passHostHeader(),
						routes(
							route("/auth", "PathPrefix:/auth"),
							route("basic", "Host:basic")),
					),
				),
			),
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			client := newClientMock(test.fixtures...)
			client.watchChan = make(chan interface{})

			actual, err := test.provider.loadIngresses(client)
			require.NoError(t, err, "error loading ingresses")

			// f, err := os.Create(filepath.Join("temp", test.desc+".toml"))
			// require.NoError(t, err, "error creating file")
			// err = toml.NewEncoder(f).Encode(test.expected)
			// require.NoError(t, err, "error writing TOML")

			assert.Equal(t, test.expected, actual)
		})
	}
}

func Test_addGlobalBackend(t *testing.T) {
	testCases := []struct {
		desc     string
		client   clientMock
		ingress  *extensionsv1beta1.Ingress
		config   *types.Configuration
		expected string
	}{
		{
			desc:   "Duplicate Frontend",
			client: clientMock{},
			ingress: buildIngress(
				iNamespace("testing"),
				iSpecBackends(iSpecBackend(iIngressBackend("service1", intstr.FromInt(80)))),
			),
			config: buildConfiguration(
				frontends(
					frontend("global-default-backend",
						frontendName("global-default-frontend"),
						passHostHeader(),
						routes(
							route("/", "PathPrefix:/"),
						),
					),
				),
			),
			expected: "duplicate frontend: global-default-frontend",
		},
		{
			desc:   "Duplicate Backend",
			client: clientMock{},
			ingress: buildIngress(
				iNamespace("testing"),
				iSpecBackends(iSpecBackend(iIngressBackend("service1", intstr.FromInt(80)))),
			),
			config: buildConfiguration(
				backends(
					backend("global-default-backend",
						lbMethod("wrr"),
						servers(
							server("http://10.10.0.1:8080", weight(1)),
						),
					),
				)),
			expected: "duplicate backend: global-default-backend",
		},
		{
			desc:   "ServiceMissing",
			client: clientMock{},
			ingress: buildIngress(
				iNamespace("testing"),
				iSpecBackends(iSpecBackend(iIngressBackend("service1", intstr.FromInt(80)))),
			),
			config: buildConfiguration(
				frontends(),
				backends(),
			),
			expected: "service not found for testing/service1",
		},
		{
			desc: "ServiceAPIError",
			client: clientMock{
				apiServiceError: errors.New("failed kube api call"),
			},
			ingress: buildIngress(
				iNamespace("testing"),
				iSpecBackends(iSpecBackend(iIngressBackend("service1", intstr.FromInt(80)))),
			),
			config: buildConfiguration(
				frontends(),
				backends(),
			),
			expected: "error while retrieving service information from k8s API testing/service1: failed kube api call",
		},
		{
			desc: "EndpointMissing",
			client: clientMock{
				services: []*corev1.Service{
					buildService(
						sName("service"),
						sNamespace("testing"),
						sUID("1"),
						sSpec(
							clusterIP("10.0.0.1"),
							sPorts(sPort(80, "")),
						),
					),
				},
			},
			ingress: buildIngress(
				iNamespace("testing"),
				iSpecBackends(iSpecBackend(iIngressBackend("service", intstr.FromInt(80)))),
			),
			config: buildConfiguration(
				frontends(),
				backends(),
			),
			expected: "endpoints not found for testing/service",
		},
		{
			desc: "EndpointAPIError",
			client: clientMock{
				apiEndpointsError: errors.New("failed kube api call"),
				services: []*corev1.Service{
					buildService(
						sName("service"),
						sNamespace("testing"),
						sUID("1"),
						sSpec(
							clusterIP("10.0.0.1"),
							sPorts(sPort(80, "")),
						),
					),
				},
			},
			ingress: buildIngress(
				iNamespace("testing"),
				iSpecBackends(iSpecBackend(iIngressBackend("service", intstr.FromInt(80)))),
			),
			config: buildConfiguration(
				frontends(),
				backends(),
			),
			expected: "error retrieving endpoint information from k8s API testing/service: failed kube api call",
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			provider := Provider{}

			mock := test.client
			mock.watchChan = make(chan interface{})

			err := provider.addGlobalBackend(mock, test.ingress, test.config)
			assert.EqualError(t, err, test.expected)
		})
	}
}

func TestProvider_newK8sClient_inCluster(t *testing.T) {
	p := Provider{}
	os.Setenv("KUBERNETES_SERVICE_HOST", "localhost")
	os.Setenv("KUBERNETES_SERVICE_PORT", "443")
	defer os.Clearenv()
	_, err := p.newK8sClient("")
	assert.EqualError(t, err, "failed to create in-cluster configuration: open /var/run/secrets/kubernetes.io/serviceaccount/token: no such file or directory")
}

func TestProvider_newK8sClient_inCluster_failLabelSel(t *testing.T) {
	p := Provider{}
	os.Setenv("KUBERNETES_SERVICE_HOST", "localhost")
	os.Setenv("KUBERNETES_SERVICE_PORT", "443")
	defer os.Clearenv()
	_, err := p.newK8sClient("%")
	assert.EqualError(t, err, "invalid ingress label selector: \"%\"")
}

func TestProvider_newK8sClient_outOfCluster(t *testing.T) {
	p := Provider{}
	p.Endpoint = "localhost"
	_, err := p.newK8sClient("")
	assert.NoError(t, err)
}

func TestRuleType(t *testing.T) {
	testCases := []struct {
		desc             string
		ingressRuleType  string
		frontendRuleType string
	}{
		{
			desc:             "rule type annotation missing",
			ingressRuleType:  "",
			frontendRuleType: ruleTypePathPrefix,
		},
		{
			desc:             "Path rule type annotation set",
			ingressRuleType:  "Path",
			frontendRuleType: "Path",
		},
		{
			desc:             "PathStrip rule type annotation set",
			ingressRuleType:  "PathStrip",
			frontendRuleType: "PathStrip",
		},
		{
			desc:             "PathPrefixStrip rule type annotation set",
			ingressRuleType:  "PathPrefixStrip",
			frontendRuleType: "PathPrefixStrip",
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			ingress := buildIngress(iRules(iRule(
				iHost("host"),
				iPaths(
					onePath(iPath("/path"), iBackend("service", intstr.FromInt(80))),
				),
			)))

			ingress.Annotations = map[string]string{
				annotationKubernetesRuleType: test.ingressRuleType,
			}

			service := buildService(
				sName("service"),
				sUID("1"),
				sSpec(sPorts(sPort(801, "http"))),
			)

			watchChan := make(chan interface{})
			client := clientMock{
				ingresses: []*extensionsv1beta1.Ingress{ingress},
				services:  []*corev1.Service{service},
				watchChan: watchChan,
			}
			provider := Provider{DisablePassHostHeaders: true}

			actualConfig, err := provider.loadIngresses(client)
			require.NoError(t, err, "error loading ingresses")

			expected := buildFrontends(frontend("host/path",
				routes(
					route("/path", fmt.Sprintf("%s:/path", test.frontendRuleType)),
					route("host", "Host:host")),
			))

			assert.Equal(t, expected, actualConfig.Frontends)
		})
	}
}

func TestRuleFails(t *testing.T) {
	testCases := []struct {
		desc                      string
		ruletypeAnnotation        string
		requestModifierAnnotation string
	}{
		{
			desc:               "Rule-type using unknown rule",
			ruletypeAnnotation: "Foo: /bar",
		},
		{
			desc:               "Rule type full of spaces",
			ruletypeAnnotation: "  ",
		},
		{
			desc:               "Rule type missing both parts of rule",
			ruletypeAnnotation: "  :  ",
		},
		{
			desc:                      "Rule type combined with replacepath modifier",
			ruletypeAnnotation:        "ReplacePath",
			requestModifierAnnotation: "ReplacePath:/foo",
		},
		{
			desc:                      "Rule type combined with replacepathregex modifier",
			ruletypeAnnotation:        "ReplacePath",
			requestModifierAnnotation: "ReplacePathRegex:/foo /bar",
		},
		{
			desc:               "Empty Rule",
			ruletypeAnnotation: " : /bar",
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			ingress := buildIngress(iRules(iRule(
				iHost("host"),
				iPaths(
					onePath(iPath("/path"), iBackend("service", intstr.FromInt(80))),
				),
			)))

			ingress.Annotations = map[string]string{
				annotationKubernetesRuleType:        test.ruletypeAnnotation,
				annotationKubernetesRequestModifier: test.requestModifierAnnotation,
			}

			_, err := getRuleForPath(extensionsv1beta1.HTTPIngressPath{Path: "/path"}, ingress)
			assert.Error(t, err)
		})
	}
}

func TestModifierType(t *testing.T) {
	testCases := []struct {
		desc                      string
		requestModifierAnnotation string
		expectedModifierRule      string
	}{
		{
			desc:                      "Request modifier annotation missing",
			requestModifierAnnotation: "",
			expectedModifierRule:      "",
		},
		{
			desc:                      "AddPrefix modifier annotation",
			requestModifierAnnotation: " AddPrefix: /foo",
			expectedModifierRule:      "AddPrefix:/foo",
		},
		{
			desc:                      "ReplacePath modifier annotation",
			requestModifierAnnotation: " ReplacePath: /foo",
			expectedModifierRule:      "ReplacePath:/foo",
		},
		{
			desc:                      "ReplacePathRegex modifier annotation",
			requestModifierAnnotation: " ReplacePathRegex: /foo /bar",
			expectedModifierRule:      "ReplacePathRegex:/foo /bar",
		},
		{
			desc:                      "AddPrefix modifier annotation",
			requestModifierAnnotation: "AddPrefix:/foo",
			expectedModifierRule:      "AddPrefix:/foo",
		},
		{
			desc:                      "ReplacePath modifier annotation",
			requestModifierAnnotation: "ReplacePath:/foo",
			expectedModifierRule:      "ReplacePath:/foo",
		},
		{
			desc:                      "ReplacePathRegex modifier annotation",
			requestModifierAnnotation: "ReplacePathRegex:/foo /bar",
			expectedModifierRule:      "ReplacePathRegex:/foo /bar",
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			ingress := buildIngress(iRules(iRule(
				iHost("host"),
				iPaths(
					onePath(iPath("/path"), iBackend("service", intstr.FromInt(80))),
				),
			)))

			ingress.Annotations = map[string]string{
				annotationKubernetesRequestModifier: test.requestModifierAnnotation,
			}

			service := buildService(
				sName("service"),
				sUID("1"),
				sSpec(sPorts(sPort(801, "http"))),
			)

			watchChan := make(chan interface{})
			client := clientMock{
				ingresses: []*extensionsv1beta1.Ingress{ingress},
				services:  []*corev1.Service{service},
				watchChan: watchChan,
			}

			provider := Provider{DisablePassHostHeaders: true}

			actualConfig, err := provider.loadIngresses(client)
			require.NoError(t, err, "error loading ingresses")

			expectedRules := []string{"PathPrefix:/path"}
			if len(test.expectedModifierRule) > 0 {
				expectedRules = append(expectedRules, test.expectedModifierRule)
			}

			expected := buildFrontends(frontend("host/path",
				routes(
					route("/path", strings.Join(expectedRules, ";")),
					route("host", "Host:host")),
			))

			assert.Equal(t, expected, actualConfig.Frontends)
		})
	}
}

func TestModifierFails(t *testing.T) {
	testCases := []struct {
		desc                      string
		requestModifierAnnotation string
	}{
		{
			desc:                      "Request modifier missing part of annotation",
			requestModifierAnnotation: "AddPrefix: ",
		},
		{
			desc:                      "Request modifier full of spaces annotation",
			requestModifierAnnotation: "    ",
		},
		{
			desc:                      "Request modifier missing both parts of annotation",
			requestModifierAnnotation: "  :  ",
		},
		{
			desc:                      "Request modifier using unknown rule",
			requestModifierAnnotation: "Foo: /bar",
		},
		{
			desc:                      "Missing Rule",
			requestModifierAnnotation: " : /bar",
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			ingress := buildIngress(iRules(iRule(
				iHost("host"),
				iPaths(
					onePath(iPath("/path"), iBackend("service", intstr.FromInt(80))),
				),
			)))

			ingress.Annotations = map[string]string{
				annotationKubernetesRequestModifier: test.requestModifierAnnotation,
			}

			_, err := getRuleForPath(extensionsv1beta1.HTTPIngressPath{Path: "/path"}, ingress)
			assert.Error(t, err)
		})
	}
}

func Test_getFrontendRedirect_InvalidRedirectAnnotation(t *testing.T) {
	ingresses := []*extensionsv1beta1.Ingress{
		buildIngress(iNamespace("awesome"),
			iAnnotation(annotationKubernetesRedirectRegex, `bad\.regex`),
			iAnnotation(annotationKubernetesRedirectReplacement, "test"),
			iRules(iRule(
				iHost("foo"),
				iPaths(onePath(iPath("/bar"), iBackend("service1", intstr.FromInt(80))))),
			),
		),
		buildIngress(iNamespace("awesome"),
			iAnnotation(annotationKubernetesRedirectRegex, `test`),
			iAnnotation(annotationKubernetesRedirectReplacement, `bad\.replacement`),
			iRules(iRule(
				iHost("foo"),
				iPaths(onePath(iPath("/bar"), iBackend("service1", intstr.FromInt(80))))),
			),
		),
	}

	for _, ingress := range ingresses {
		actual := getFrontendRedirect(ingress, "test", "/")
		var expected *types.Redirect

		assert.Equal(t, expected, actual)
	}
}

func TestProvider_loadIngresses_KubeAPIErrors(t *testing.T) {
	ingresses := []*extensionsv1beta1.Ingress{
		buildIngress(
			iNamespace("testing"),
			iRules(
				iRule(
					iHost("foo"),
					iPaths(onePath(iPath("/bar"), iBackend("service1", intstr.FromInt(80))))),
			),
		),
	}

	services := []*corev1.Service{
		buildService(
			sName("service1"),
			sNamespace("testing"),
			sUID("1"),
			sSpec(
				clusterIP("10.0.0.1"),
				sPorts(sPort(80, ""))),
		),
	}

	watchChan := make(chan interface{})
	apiErr := errors.New("failed kube api call")

	testCases := []struct {
		desc            string
		apiServiceErr   error
		apiEndpointsErr error
	}{
		{
			desc:          "failed service call",
			apiServiceErr: apiErr,
		},
		{
			desc:            "failed endpoints call",
			apiEndpointsErr: apiErr,
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			client := clientMock{
				ingresses:         ingresses,
				services:          services,
				watchChan:         watchChan,
				apiServiceError:   test.apiServiceErr,
				apiEndpointsError: test.apiEndpointsErr,
			}

			provider := Provider{}

			if _, err := provider.loadIngresses(client); err != nil {
				if client.apiServiceError != nil {
					assert.EqualError(t, err, "failed kube api call")
				}
				if client.apiEndpointsError != nil {
					assert.EqualError(t, err, "failed kube api call")
				}
			}
		})
	}
}

func TestProvider_loadIngresses_ForwardAuthWithTLSSecretFailures(t *testing.T) {
	testCases := []struct {
		desc       string
		secretName string
		certName   string
		certData   string
		keyName    string
		keyData    string
	}{
		{
			desc:       "empty certificate and key",
			secretName: "secret",
			certName:   "",
			certData:   "",
			keyName:    "",
			keyData:    "",
		},
		{
			desc:       "wrong secret name, empty certificate and key",
			secretName: "wrongSecret",
			certName:   "",
			certData:   "",
			keyName:    "",
			keyData:    "",
		},
		{
			desc:       "empty certificate data",
			secretName: "secret",
			certName:   "tls.crt",
			certData:   "",
			keyName:    "tls.key",
			keyData:    "-----BEGIN PRIVATE KEY-----\n-----END PRIVATE KEY-----",
		},
		{
			desc:       "empty key data",
			secretName: "secret",
			certName:   "tls.crt",
			certData:   "-----BEGIN CERTIFICATE-----\n-----END CERTIFICATE-----",
			keyName:    "tls.key",
			keyData:    "",
		},
		{
			desc:       "wrong cert name",
			secretName: "secret",
			certName:   "cert.crt",
			certData:   "-----BEGIN CERTIFICATE-----\n-----END CERTIFICATE----",
			keyName:    "tls.key",
			keyData:    "-----BEGIN PRIVATE KEY-----\n-----END PRIVATE KEY-----",
		},
		{
			desc:       "wrong key name",
			secretName: "secret",
			certName:   "tls.crt",
			certData:   "-----BEGIN CERTIFICATE-----\n-----END CERTIFICATE-----",
			keyName:    "cert.key",
			keyData:    "-----BEGIN PRIVATE KEY-----\n-----END PRIVATE KEY-----",
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			secrets := []*corev1.Secret{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      test.secretName,
					UID:       "1",
					Namespace: "testing",
				},
				Data: map[string][]byte{
					test.certName: []byte(test.certData),
					test.keyName:  []byte(test.keyData),
				},
			}}

			client := newClientMock(
				filepath.Join("fixtures", "loadIngressesForwardAuthWithTLSSecretFailures_services.yml"),
				filepath.Join("fixtures", "loadIngressesForwardAuthWithTLSSecretFailures_ingresses.yml"),
				filepath.Join("fixtures", "loadIngressesForwardAuthWithTLSSecretFailures_endpoints.yml"),
			)

			client.secrets = secrets
			client.watchChan = make(chan interface{})

			provider := Provider{}

			actual, err := provider.loadIngresses(client)
			require.NoError(t, err, "error loading ingresses")

			expected := buildConfiguration(
				backends(
					backend("foo/bar",
						lbMethod("wrr"),
						servers(),
					),
				),
				frontends(),
			)

			assert.Equal(t, expected, actual)
		})
	}
}

func TestGetTLS(t *testing.T) {
	testIngressWithoutHostname := buildIngress(
		iNamespace("testing"),
		iRules(
			iRule(iHost("ep1.example.com")),
			iRule(iHost("ep2.example.com")),
		),
		iTLSes(
			iTLS("test-secret"),
		),
	)

	testIngressWithoutSecret := buildIngress(
		iNamespace("testing"),
		iRules(
			iRule(iHost("ep1.example.com")),
		),
		iTLSes(
			iTLS("", "foo.com"),
		),
	)

	testCases := []struct {
		desc      string
		ingress   *extensionsv1beta1.Ingress
		client    Client
		result    map[string]*tls.Configuration
		errResult string
	}{
		{
			desc:    "api client returns error",
			ingress: testIngressWithoutHostname,
			client: clientMock{
				apiSecretError: errors.New("api secret error"),
			},
			errResult: "failed to fetch secret testing/test-secret: api secret error",
		},
		{
			desc:      "api client doesn't find secret",
			ingress:   testIngressWithoutHostname,
			client:    clientMock{},
			errResult: "secret testing/test-secret does not exist",
		},
		{
			desc:    "entry 'tls.crt' in secret missing",
			ingress: testIngressWithoutHostname,
			client: clientMock{
				secrets: []*corev1.Secret{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-secret",
							Namespace: "testing",
						},
						Data: map[string][]byte{
							"tls.key": []byte("tls-key"),
						},
					},
				},
			},
			errResult: "secret testing/test-secret is missing the following TLS data entries: tls.crt",
		},
		{
			desc:    "entry 'tls.key' in secret missing",
			ingress: testIngressWithoutHostname,
			client: clientMock{
				secrets: []*corev1.Secret{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-secret",
							Namespace: "testing",
						},
						Data: map[string][]byte{
							"tls.crt": []byte("tls-crt"),
						},
					},
				},
			},
			errResult: "secret testing/test-secret is missing the following TLS data entries: tls.key",
		},
		{
			desc:    "secret doesn't provide any of the required fields",
			ingress: testIngressWithoutHostname,
			client: clientMock{
				secrets: []*corev1.Secret{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-secret",
							Namespace: "testing",
						},
						Data: map[string][]byte{},
					},
				},
			},
			errResult: "secret testing/test-secret is missing the following TLS data entries: tls.crt, tls.key",
		},
		{
			desc: "add certificates to the configuration",
			ingress: buildIngress(
				iNamespace("testing"),
				iRules(
					iRule(iHost("ep1.example.com")),
					iRule(iHost("ep2.example.com")),
					iRule(iHost("ep3.example.com")),
				),
				iTLSes(
					iTLS("test-secret"),
					iTLS("test-secret2"),
				),
			),
			client: clientMock{
				secrets: []*corev1.Secret{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-secret2",
							Namespace: "testing",
						},
						Data: map[string][]byte{
							"tls.crt": []byte("-----BEGIN CERTIFICATE-----\n-----END CERTIFICATE-----\n"),
							"tls.key": []byte("-----BEGIN PRIVATE KEY-----\n-----END PRIVATE KEY-----\n"),
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-secret",
							Namespace: "testing",
						},
						Data: map[string][]byte{
							"tls.crt": []byte("-----BEGIN CERTIFICATE-----\n-----END CERTIFICATE-----\n"),
							"tls.key": []byte("-----BEGIN PRIVATE KEY-----\n-----END PRIVATE KEY-----\n"),
						},
					},
				},
			},
			result: map[string]*tls.Configuration{
				"testing/test-secret": {
					Certificate: &tls.Certificate{
						CertFile: tls.FileOrContent("-----BEGIN CERTIFICATE-----\n-----END CERTIFICATE-----\n"),
						KeyFile:  tls.FileOrContent("-----BEGIN PRIVATE KEY-----\n-----END PRIVATE KEY-----\n"),
					},
				},
				"testing/test-secret2": {
					Certificate: &tls.Certificate{
						CertFile: tls.FileOrContent("-----BEGIN CERTIFICATE-----\n-----END CERTIFICATE-----\n"),
						KeyFile:  tls.FileOrContent("-----BEGIN PRIVATE KEY-----\n-----END PRIVATE KEY-----\n"),
					},
				},
			},
		},
		{
			desc:    "return nil when no secret is defined",
			ingress: testIngressWithoutSecret,
			client:  clientMock{},
			result:  map[string]*tls.Configuration{},
		},
		{
			desc: "pass the endpoints defined in the annotation to the certificate",
			ingress: buildIngress(
				iNamespace("testing"),
				iAnnotation(annotationKubernetesFrontendEntryPoints, "https,api-secure"),
				iRules(iRule(iHost("example.com"))),
				iTLSes(iTLS("test-secret")),
			),
			client: clientMock{
				secrets: []*corev1.Secret{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-secret",
							Namespace: "testing",
						},
						Data: map[string][]byte{
							"tls.crt": []byte("-----BEGIN CERTIFICATE-----\n-----END CERTIFICATE-----\n"),
							"tls.key": []byte("-----BEGIN PRIVATE KEY-----\n-----END PRIVATE KEY-----\n"),
						},
					},
				},
			},
			result: map[string]*tls.Configuration{
				"testing/test-secret": {
					EntryPoints: []string{"api-secure", "https"},
					Certificate: &tls.Certificate{
						CertFile: tls.FileOrContent("-----BEGIN CERTIFICATE-----\n-----END CERTIFICATE-----\n"),
						KeyFile:  tls.FileOrContent("-----BEGIN PRIVATE KEY-----\n-----END PRIVATE KEY-----\n"),
					},
				},
			},
		},
		{
			desc: "load bad certificate",
			ingress: buildIngress(
				iNamespace("testing"),
				iAnnotation(annotationKubernetesFrontendEntryPoints, "https,api-secure"),
				iRules(iRule(iHost("example.com"))),
				iTLSes(iTLS("test-secret")),
			),
			client: clientMock{
				secrets: []*corev1.Secret{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-secret",
							Namespace: "testing",
						},
						Data: map[string][]byte{
							"tls.crt": []byte("invalid"),
							"tls.key": []byte("invalid"),
						},
					},
				},
			},
			errResult: "secret testing/test-secret does not contain PEM formatted TLS data entries: tls.crt,tls.key",
		},
		{
			desc: "load nested bad certificate",
			ingress: buildIngress(
				iNamespace("testing"),
				iAnnotation(annotationKubernetesFrontendEntryPoints, "https,api-secure"),
				iRules(iRule(iHost("example.com"))),
				iTLSes(iTLS("test-secret")),
			),
			client: clientMock{
				secrets: []*corev1.Secret{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-secret",
							Namespace: "testing",
						},
						Data: map[string][]byte{
							"tls.crt": []byte("-----BEGIN CERTIFICATE-----\n-----END CERTIFICATE-----\n-----BEGIN CERTIFICATE-----\n\x00\x00\x00-----END CERTIFICATE-----\n"),
							"tls.key": []byte("-----BEGIN PRIVATE KEY-----\n-----END PRIVATE KEY-----"),
						},
					},
				},
			},
			errResult: "secret testing/test-secret does not contain PEM formatted TLS data entries: tls.crt",
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			tlsConfigs := map[string]*tls.Configuration{}
			err := getTLS(test.ingress, test.client, tlsConfigs)

			if test.errResult != "" {
				assert.EqualError(t, err, test.errResult)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, test.result, tlsConfigs)
			}
		})
	}
}

func TestProvider_updateIngressStatus(t *testing.T) {
	testCases := []struct {
		desc                  string
		ingressEndpoint       *IngressEndpoint
		apiServiceError       error
		apiIngressStatusError error
		expectedError         bool
	}{
		{
			desc:          "without IngressEndpoint configuration",
			expectedError: false,
		},
		{
			desc:            "without any IngressEndpoint option",
			ingressEndpoint: &IngressEndpoint{},
			expectedError:   true,
		},
		{
			desc: "PublishedService - invalid format",
			ingressEndpoint: &IngressEndpoint{
				PublishedService: "foo",
			},
			expectedError: true,
		},
		{
			desc: "PublishedService - missing service",
			ingressEndpoint: &IngressEndpoint{
				PublishedService: "foo/bar",
			},
			expectedError: true,
		},
		{
			desc: "PublishedService - get service error",
			ingressEndpoint: &IngressEndpoint{
				PublishedService: "foo/bar",
			},
			apiServiceError: errors.New("error"),
			expectedError:   true,
		},
		{
			desc: "PublishedService - Skipping empty LoadBalancerIngress",
			ingressEndpoint: &IngressEndpoint{
				PublishedService: "testing/service-empty-status",
			},
			expectedError: false,
		},
		{
			desc: "PublishedService - with update error",
			ingressEndpoint: &IngressEndpoint{
				PublishedService: "testing/service",
			},
			apiIngressStatusError: errors.New("error"),
			expectedError:         true,
		},
		{
			desc: "PublishedService - right service",
			ingressEndpoint: &IngressEndpoint{
				PublishedService: "testing/service",
			},
			expectedError: false,
		},
		{
			desc: "IP - valid",
			ingressEndpoint: &IngressEndpoint{
				IP: "127.0.0.1",
			},
			expectedError: false,
		},
		{
			desc: "IP - with update error",
			ingressEndpoint: &IngressEndpoint{
				IP: "127.0.0.1",
			},
			apiIngressStatusError: errors.New("error"),
			expectedError:         true,
		},
		{
			desc: "hostname - valid",
			ingressEndpoint: &IngressEndpoint{
				Hostname: "foo",
			},
			expectedError: false,
		},
		{
			desc: "hostname - with update error",
			ingressEndpoint: &IngressEndpoint{
				Hostname: "foo",
			},
			apiIngressStatusError: errors.New("error"),
			expectedError:         true,
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			client := newClientMock(filepath.Join("fixtures", "providerUpdateIngressStatus_services.yml"))
			client.apiServiceError = test.apiServiceError
			client.apiIngressStatusError = test.apiIngressStatusError

			p := &Provider{
				IngressEndpoint: test.ingressEndpoint,
			}

			err := p.updateIngressStatus(&extensionsv1beta1.Ingress{}, client)

			if test.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
