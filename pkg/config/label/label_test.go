package label

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ptypes "github.com/traefik/paerser/types"
	"github.com/traefik/traefik/v2/pkg/config/dynamic"
)

func TestDecodeConfiguration(t *testing.T) {
	labels := map[string]string{
		"traefik.http.middlewares.Middleware0.addprefix.prefix":                                    "foobar",
		"traefik.http.middlewares.Middleware1.basicauth.headerfield":                               "foobar",
		"traefik.http.middlewares.Middleware1.basicauth.realm":                                     "foobar",
		"traefik.http.middlewares.Middleware1.basicauth.removeheader":                              "true",
		"traefik.http.middlewares.Middleware1.basicauth.users":                                     "foobar, fiibar",
		"traefik.http.middlewares.Middleware1.basicauth.usersfile":                                 "foobar",
		"traefik.http.middlewares.Middleware2.buffering.maxrequestbodybytes":                       "42",
		"traefik.http.middlewares.Middleware2.buffering.maxresponsebodybytes":                      "42",
		"traefik.http.middlewares.Middleware2.buffering.memrequestbodybytes":                       "42",
		"traefik.http.middlewares.Middleware2.buffering.memresponsebodybytes":                      "42",
		"traefik.http.middlewares.Middleware2.buffering.retryexpression":                           "foobar",
		"traefik.http.middlewares.Middleware3.chain.middlewares":                                   "foobar, fiibar",
		"traefik.http.middlewares.Middleware4.circuitbreaker.expression":                           "foobar",
		"traefik.http.middlewares.Middleware5.digestauth.headerfield":                              "foobar",
		"traefik.http.middlewares.Middleware5.digestauth.realm":                                    "foobar",
		"traefik.http.middlewares.Middleware5.digestauth.removeheader":                             "true",
		"traefik.http.middlewares.Middleware5.digestauth.users":                                    "foobar, fiibar",
		"traefik.http.middlewares.Middleware5.digestauth.usersfile":                                "foobar",
		"traefik.http.middlewares.Middleware6.errors.query":                                        "foobar",
		"traefik.http.middlewares.Middleware6.errors.service":                                      "foobar",
		"traefik.http.middlewares.Middleware6.errors.status":                                       "foobar, fiibar",
		"traefik.http.middlewares.Middleware7.forwardauth.address":                                 "foobar",
		"traefik.http.middlewares.Middleware7.forwardauth.authresponseheaders":                     "foobar, fiibar",
		"traefik.http.middlewares.Middleware7.forwardauth.authrequestheaders":                      "foobar, fiibar",
		"traefik.http.middlewares.Middleware7.forwardauth.tls.ca":                                  "foobar",
		"traefik.http.middlewares.Middleware7.forwardauth.tls.caoptional":                          "true",
		"traefik.http.middlewares.Middleware7.forwardauth.tls.cert":                                "foobar",
		"traefik.http.middlewares.Middleware7.forwardauth.tls.insecureskipverify":                  "true",
		"traefik.http.middlewares.Middleware7.forwardauth.tls.key":                                 "foobar",
		"traefik.http.middlewares.Middleware7.forwardauth.trustforwardheader":                      "true",
		"traefik.http.middlewares.Middleware8.headers.accesscontrolallowcredentials":               "true",
		"traefik.http.middlewares.Middleware8.headers.allowedhosts":                                "foobar, fiibar",
		"traefik.http.middlewares.Middleware8.headers.accesscontrolallowheaders":                   "X-foobar, X-fiibar",
		"traefik.http.middlewares.Middleware8.headers.accesscontrolallowmethods":                   "GET, PUT",
		"traefik.http.middlewares.Middleware8.headers.accesscontrolalloworiginList":                "foobar, fiibar",
		"traefik.http.middlewares.Middleware8.headers.accesscontrolalloworiginListRegex":           "foobar, fiibar",
		"traefik.http.middlewares.Middleware8.headers.accesscontrolexposeheaders":                  "X-foobar, X-fiibar",
		"traefik.http.middlewares.Middleware8.headers.accesscontrolmaxage":                         "200",
		"traefik.http.middlewares.Middleware8.headers.addvaryheader":                               "true",
		"traefik.http.middlewares.Middleware8.headers.browserxssfilter":                            "true",
		"traefik.http.middlewares.Middleware8.headers.contentsecuritypolicy":                       "foobar",
		"traefik.http.middlewares.Middleware8.headers.contenttypenosniff":                          "true",
		"traefik.http.middlewares.Middleware8.headers.custombrowserxssvalue":                       "foobar",
		"traefik.http.middlewares.Middleware8.headers.customframeoptionsvalue":                     "foobar",
		"traefik.http.middlewares.Middleware8.headers.customrequestheaders.name0":                  "foobar",
		"traefik.http.middlewares.Middleware8.headers.customrequestheaders.name1":                  "foobar",
		"traefik.http.middlewares.Middleware8.headers.customresponseheaders.name0":                 "foobar",
		"traefik.http.middlewares.Middleware8.headers.customresponseheaders.name1":                 "foobar",
		"traefik.http.middlewares.Middleware8.headers.forcestsheader":                              "true",
		"traefik.http.middlewares.Middleware8.headers.framedeny":                                   "true",
		"traefik.http.middlewares.Middleware8.headers.hostsproxyheaders":                           "foobar, fiibar",
		"traefik.http.middlewares.Middleware8.headers.isdevelopment":                               "true",
		"traefik.http.middlewares.Middleware8.headers.publickey":                                   "foobar",
		"traefik.http.middlewares.Middleware8.headers.referrerpolicy":                              "foobar",
		"traefik.http.middlewares.Middleware8.headers.featurepolicy":                               "foobar",
		"traefik.http.middlewares.Middleware8.headers.permissionspolicy":                           "foobar",
		"traefik.http.middlewares.Middleware8.headers.sslforcehost":                                "true",
		"traefik.http.middlewares.Middleware8.headers.sslhost":                                     "foobar",
		"traefik.http.middlewares.Middleware8.headers.sslproxyheaders.name0":                       "foobar",
		"traefik.http.middlewares.Middleware8.headers.sslproxyheaders.name1":                       "foobar",
		"traefik.http.middlewares.Middleware8.headers.sslredirect":                                 "true",
		"traefik.http.middlewares.Middleware8.headers.ssltemporaryredirect":                        "true",
		"traefik.http.middlewares.Middleware8.headers.stsincludesubdomains":                        "true",
		"traefik.http.middlewares.Middleware8.headers.stspreload":                                  "true",
		"traefik.http.middlewares.Middleware8.headers.stsseconds":                                  "42",
		"traefik.http.middlewares.Middleware9.ipwhitelist.ipstrategy.depth":                        "42",
		"traefik.http.middlewares.Middleware9.ipwhitelist.ipstrategy.excludedips":                  "foobar, fiibar",
		"traefik.http.middlewares.Middleware9.ipwhitelist.sourcerange":                             "foobar, fiibar",
		"traefik.http.middlewares.Middleware10.inflightreq.amount":                                 "42",
		"traefik.http.middlewares.Middleware10.inflightreq.sourcecriterion.ipstrategy.depth":       "42",
		"traefik.http.middlewares.Middleware10.inflightreq.sourcecriterion.ipstrategy.excludedips": "foobar, fiibar",
		"traefik.http.middlewares.Middleware10.inflightreq.sourcecriterion.requestheadername":      "foobar",
		"traefik.http.middlewares.Middleware10.inflightreq.sourcecriterion.requesthost":            "true",
		"traefik.http.middlewares.Middleware11.passtlsclientcert.info.notafter":                    "true",
		"traefik.http.middlewares.Middleware11.passtlsclientcert.info.notbefore":                   "true",
		"traefik.http.middlewares.Middleware11.passtlsclientcert.info.sans":                        "true",
		"traefik.http.middlewares.Middleware11.passTLSClientCert.info.serialNumber":                "true",
		"traefik.http.middlewares.Middleware11.passtlsclientcert.info.subject.commonname":          "true",
		"traefik.http.middlewares.Middleware11.passtlsclientcert.info.subject.country":             "true",
		"traefik.http.middlewares.Middleware11.passtlsclientcert.info.subject.domaincomponent":     "true",
		"traefik.http.middlewares.Middleware11.passtlsclientcert.info.subject.locality":            "true",
		"traefik.http.middlewares.Middleware11.passtlsclientcert.info.subject.organization":        "true",
		"traefik.http.middlewares.Middleware11.passtlsclientcert.info.subject.province":            "true",
		"traefik.http.middlewares.Middleware11.passtlsclientcert.info.subject.serialnumber":        "true",
		"traefik.http.middlewares.Middleware11.passtlsclientcert.info.issuer.commonname":           "true",
		"traefik.http.middlewares.Middleware11.passtlsclientcert.info.issuer.country":              "true",
		"traefik.http.middlewares.Middleware11.passtlsclientcert.info.issuer.domaincomponent":      "true",
		"traefik.http.middlewares.Middleware11.passtlsclientcert.info.issuer.locality":             "true",
		"traefik.http.middlewares.Middleware11.passtlsclientcert.info.issuer.organization":         "true",
		"traefik.http.middlewares.Middleware11.passtlsclientcert.info.issuer.province":             "true",
		"traefik.http.middlewares.Middleware11.passtlsclientcert.info.issuer.serialnumber":         "true",
		"traefik.http.middlewares.Middleware11.passtlsclientcert.pem":                              "true",
		"traefik.http.middlewares.Middleware12.ratelimit.average":                                  "42",
		"traefik.http.middlewares.Middleware12.ratelimit.period":                                   "1s",
		"traefik.http.middlewares.Middleware12.ratelimit.burst":                                    "42",
		"traefik.http.middlewares.Middleware12.ratelimit.sourcecriterion.requestheadername":        "foobar",
		"traefik.http.middlewares.Middleware12.ratelimit.sourcecriterion.requesthost":              "true",
		"traefik.http.middlewares.Middleware12.ratelimit.sourcecriterion.ipstrategy.depth":         "42",
		"traefik.http.middlewares.Middleware12.ratelimit.sourcecriterion.ipstrategy.excludedips":   "foobar, foobar",
		"traefik.http.middlewares.Middleware13.redirectregex.permanent":                            "true",
		"traefik.http.middlewares.Middleware13.redirectregex.regex":                                "foobar",
		"traefik.http.middlewares.Middleware13.redirectregex.replacement":                          "foobar",
		"traefik.http.middlewares.Middleware13b.redirectscheme.scheme":                             "https",
		"traefik.http.middlewares.Middleware13b.redirectscheme.port":                               "80",
		"traefik.http.middlewares.Middleware13b.redirectscheme.permanent":                          "true",
		"traefik.http.middlewares.Middleware14.replacepath.path":                                   "foobar",
		"traefik.http.middlewares.Middleware15.replacepathregex.regex":                             "foobar",
		"traefik.http.middlewares.Middleware15.replacepathregex.replacement":                       "foobar",
		"traefik.http.middlewares.Middleware16.retry.attempts":                                     "42",
		"traefik.http.middlewares.Middleware16.retry.initialinterval":                              "1s",
		"traefik.http.middlewares.Middleware17.stripprefix.prefixes":                               "foobar, fiibar",
		"traefik.http.middlewares.Middleware18.stripprefixregex.regex":                             "foobar, fiibar",
		"traefik.http.middlewares.Middleware19.compress.minimumbodysizebytes":                      "42",
		"traefik.http.middlewares.Middleware20.plugin.tomato.aaa":                                  "foo1",
		"traefik.http.middlewares.Middleware20.plugin.tomato.bbb":                                  "foo2",
		"traefik.http.routers.Router0.entrypoints":                                                 "foobar, fiibar",
		"traefik.http.routers.Router0.middlewares":                                                 "foobar, fiibar",
		"traefik.http.routers.Router0.priority":                                                    "42",
		"traefik.http.routers.Router0.rule":                                                        "foobar",
		"traefik.http.routers.Router0.tls":                                                         "true",
		"traefik.http.routers.Router0.service":                                                     "foobar",
		"traefik.http.routers.Router1.entrypoints":                                                 "foobar, fiibar",
		"traefik.http.routers.Router1.middlewares":                                                 "foobar, fiibar",
		"traefik.http.routers.Router1.priority":                                                    "42",
		"traefik.http.routers.Router1.rule":                                                        "foobar",
		"traefik.http.routers.Router1.service":                                                     "foobar",

		"traefik.http.services.Service0.loadbalancer.healthcheck.headers.name0":        "foobar",
		"traefik.http.services.Service0.loadbalancer.healthcheck.headers.name1":        "foobar",
		"traefik.http.services.Service0.loadbalancer.healthcheck.hostname":             "foobar",
		"traefik.http.services.Service0.loadbalancer.healthcheck.interval":             "foobar",
		"traefik.http.services.Service0.loadbalancer.healthcheck.path":                 "foobar",
		"traefik.http.services.Service0.loadbalancer.healthcheck.port":                 "42",
		"traefik.http.services.Service0.loadbalancer.healthcheck.scheme":               "foobar",
		"traefik.http.services.Service0.loadbalancer.healthcheck.timeout":              "foobar",
		"traefik.http.services.Service0.loadbalancer.healthcheck.followredirects":      "true",
		"traefik.http.services.Service0.loadbalancer.passhostheader":                   "true",
		"traefik.http.services.Service0.loadbalancer.responseforwarding.flushinterval": "foobar",
		"traefik.http.services.Service0.loadbalancer.server.scheme":                    "foobar",
		"traefik.http.services.Service0.loadbalancer.server.port":                      "8080",
		"traefik.http.services.Service0.loadbalancer.sticky.cookie.name":               "foobar",
		"traefik.http.services.Service0.loadbalancer.sticky.cookie.secure":             "true",
		"traefik.http.services.Service1.loadbalancer.healthcheck.headers.name0":        "foobar",
		"traefik.http.services.Service1.loadbalancer.healthcheck.headers.name1":        "foobar",
		"traefik.http.services.Service1.loadbalancer.healthcheck.hostname":             "foobar",
		"traefik.http.services.Service1.loadbalancer.healthcheck.interval":             "foobar",
		"traefik.http.services.Service1.loadbalancer.healthcheck.path":                 "foobar",
		"traefik.http.services.Service1.loadbalancer.healthcheck.port":                 "42",
		"traefik.http.services.Service1.loadbalancer.healthcheck.scheme":               "foobar",
		"traefik.http.services.Service1.loadbalancer.healthcheck.timeout":              "foobar",
		"traefik.http.services.Service1.loadbalancer.healthcheck.followredirects":      "true",
		"traefik.http.services.Service1.loadbalancer.passhostheader":                   "true",
		"traefik.http.services.Service1.loadbalancer.responseforwarding.flushinterval": "foobar",
		"traefik.http.services.Service1.loadbalancer.server.scheme":                    "foobar",
		"traefik.http.services.Service1.loadbalancer.server.port":                      "8080",
		"traefik.http.services.Service1.loadbalancer.sticky":                           "false",
		"traefik.http.services.Service1.loadbalancer.sticky.cookie.name":               "fui",

		"traefik.tcp.middlewares.Middleware0.ipwhitelist.sourcerange":      "foobar, fiibar",
		"traefik.tcp.routers.Router0.rule":                                 "foobar",
		"traefik.tcp.routers.Router0.entrypoints":                          "foobar, fiibar",
		"traefik.tcp.routers.Router0.service":                              "foobar",
		"traefik.tcp.routers.Router0.tls.passthrough":                      "false",
		"traefik.tcp.routers.Router0.tls.options":                          "foo",
		"traefik.tcp.routers.Router1.rule":                                 "foobar",
		"traefik.tcp.routers.Router1.entrypoints":                          "foobar, fiibar",
		"traefik.tcp.routers.Router1.service":                              "foobar",
		"traefik.tcp.routers.Router1.tls.options":                          "foo",
		"traefik.tcp.routers.Router1.tls.passthrough":                      "false",
		"traefik.tcp.services.Service0.loadbalancer.server.Port":           "42",
		"traefik.tcp.services.Service0.loadbalancer.TerminationDelay":      "42",
		"traefik.tcp.services.Service0.loadbalancer.proxyProtocol.version": "42",
		"traefik.tcp.services.Service1.loadbalancer.server.Port":           "42",
		"traefik.tcp.services.Service1.loadbalancer.TerminationDelay":      "42",
		"traefik.tcp.services.Service1.loadbalancer.proxyProtocol":         "true",

		"traefik.udp.routers.Router0.entrypoints":                "foobar, fiibar",
		"traefik.udp.routers.Router0.service":                    "foobar",
		"traefik.udp.routers.Router1.entrypoints":                "foobar, fiibar",
		"traefik.udp.routers.Router1.service":                    "foobar",
		"traefik.udp.services.Service0.loadbalancer.server.Port": "42",
		"traefik.udp.services.Service1.loadbalancer.server.Port": "42",
	}

	configuration, err := DecodeConfiguration(labels)
	require.NoError(t, err)

	expected := &dynamic.Configuration{
		TCP: &dynamic.TCPConfiguration{
			Routers: map[string]*dynamic.TCPRouter{
				"Router0": {
					EntryPoints: []string{
						"foobar",
						"fiibar",
					},
					Service: "foobar",
					Rule:    "foobar",
					TLS: &dynamic.RouterTCPTLSConfig{
						Passthrough: false,
						Options:     "foo",
					},
				},
				"Router1": {
					EntryPoints: []string{
						"foobar",
						"fiibar",
					},
					Service: "foobar",
					Rule:    "foobar",
					TLS: &dynamic.RouterTCPTLSConfig{
						Passthrough: false,
						Options:     "foo",
					},
				},
			},
			Middlewares: map[string]*dynamic.TCPMiddleware{
				"Middleware0": {
					IPWhiteList: &dynamic.TCPIPWhiteList{
						SourceRange: []string{"foobar", "fiibar"},
					},
				},
			},
			Services: map[string]*dynamic.TCPService{
				"Service0": {
					LoadBalancer: &dynamic.TCPServersLoadBalancer{
						Servers: []dynamic.TCPServer{
							{
								Port: "42",
							},
						},
						TerminationDelay: func(i int) *int { return &i }(42),
						ProxyProtocol:    &dynamic.ProxyProtocol{Version: 42},
					},
				},
				"Service1": {
					LoadBalancer: &dynamic.TCPServersLoadBalancer{
						Servers: []dynamic.TCPServer{
							{
								Port: "42",
							},
						},
						TerminationDelay: func(i int) *int { return &i }(42),
						ProxyProtocol:    &dynamic.ProxyProtocol{Version: 2},
					},
				},
			},
		},
		UDP: &dynamic.UDPConfiguration{
			Routers: map[string]*dynamic.UDPRouter{
				"Router0": {
					EntryPoints: []string{
						"foobar",
						"fiibar",
					},
					Service: "foobar",
				},
				"Router1": {
					EntryPoints: []string{
						"foobar",
						"fiibar",
					},
					Service: "foobar",
				},
			},
			Services: map[string]*dynamic.UDPService{
				"Service0": {
					LoadBalancer: &dynamic.UDPServersLoadBalancer{
						Servers: []dynamic.UDPServer{
							{
								Port: "42",
							},
						},
					},
				},
				"Service1": {
					LoadBalancer: &dynamic.UDPServersLoadBalancer{
						Servers: []dynamic.UDPServer{
							{
								Port: "42",
							},
						},
					},
				},
			},
		},
		HTTP: &dynamic.HTTPConfiguration{
			Routers: map[string]*dynamic.Router{
				"Router0": {
					EntryPoints: []string{
						"foobar",
						"fiibar",
					},
					Middlewares: []string{
						"foobar",
						"fiibar",
					},
					Service:  "foobar",
					Rule:     "foobar",
					Priority: 42,
					TLS:      &dynamic.RouterTLSConfig{},
				},
				"Router1": {
					EntryPoints: []string{
						"foobar",
						"fiibar",
					},
					Middlewares: []string{
						"foobar",
						"fiibar",
					},
					Service:  "foobar",
					Rule:     "foobar",
					Priority: 42,
				},
			},
			Middlewares: map[string]*dynamic.Middleware{
				"Middleware0": {
					AddPrefix: &dynamic.AddPrefix{
						Prefix: "foobar",
					},
				},
				"Middleware1": {
					BasicAuth: &dynamic.BasicAuth{
						Users: []string{
							"foobar",
							"fiibar",
						},
						UsersFile:    "foobar",
						Realm:        "foobar",
						RemoveHeader: true,
						HeaderField:  "foobar",
					},
				},
				"Middleware10": {
					InFlightReq: &dynamic.InFlightReq{
						Amount: 42,
						SourceCriterion: &dynamic.SourceCriterion{
							IPStrategy: &dynamic.IPStrategy{
								Depth:       42,
								ExcludedIPs: []string{"foobar", "fiibar"},
							},
							RequestHeaderName: "foobar",
							RequestHost:       true,
						},
					},
				},
				"Middleware11": {
					PassTLSClientCert: &dynamic.PassTLSClientCert{
						PEM: true,
						Info: &dynamic.TLSClientCertificateInfo{
							NotAfter:     true,
							NotBefore:    true,
							SerialNumber: true,
							Subject: &dynamic.TLSCLientCertificateDNInfo{
								Country:         true,
								Province:        true,
								Locality:        true,
								Organization:    true,
								CommonName:      true,
								SerialNumber:    true,
								DomainComponent: true,
							},
							Issuer: &dynamic.TLSCLientCertificateDNInfo{
								Country:         true,
								Province:        true,
								Locality:        true,
								Organization:    true,
								CommonName:      true,
								SerialNumber:    true,
								DomainComponent: true,
							},
							Sans: true,
						},
					},
				},
				"Middleware12": {
					RateLimit: &dynamic.RateLimit{
						Average: 42,
						Burst:   42,
						Period:  ptypes.Duration(time.Second),
						SourceCriterion: &dynamic.SourceCriterion{
							IPStrategy: &dynamic.IPStrategy{
								Depth:       42,
								ExcludedIPs: []string{"foobar", "foobar"},
							},
							RequestHeaderName: "foobar",
							RequestHost:       true,
						},
					},
				},
				"Middleware13": {
					RedirectRegex: &dynamic.RedirectRegex{
						Regex:       "foobar",
						Replacement: "foobar",
						Permanent:   true,
					},
				},
				"Middleware13b": {
					RedirectScheme: &dynamic.RedirectScheme{
						Scheme:    "https",
						Port:      "80",
						Permanent: true,
					},
				},
				"Middleware14": {
					ReplacePath: &dynamic.ReplacePath{
						Path: "foobar",
					},
				},
				"Middleware15": {
					ReplacePathRegex: &dynamic.ReplacePathRegex{
						Regex:       "foobar",
						Replacement: "foobar",
					},
				},
				"Middleware16": {
					Retry: &dynamic.Retry{
						Attempts:        42,
						InitialInterval: ptypes.Duration(time.Second),
					},
				},
				"Middleware17": {
					StripPrefix: &dynamic.StripPrefix{
						Prefixes: []string{
							"foobar",
							"fiibar",
						},
						ForceSlash: true,
					},
				},
				"Middleware18": {
					StripPrefixRegex: &dynamic.StripPrefixRegex{
						Regex: []string{
							"foobar",
							"fiibar",
						},
					},
				},
				"Middleware19": {
					Compress: &dynamic.Compress{
						MinimumBodySizeBytes: 42,
					},
				},
				"Middleware2": {
					Buffering: &dynamic.Buffering{
						MaxRequestBodyBytes:  42,
						MemRequestBodyBytes:  42,
						MaxResponseBodyBytes: 42,
						MemResponseBodyBytes: 42,
						RetryExpression:      "foobar",
					},
				},
				"Middleware3": {
					Chain: &dynamic.Chain{
						Middlewares: []string{
							"foobar",
							"fiibar",
						},
					},
				},
				"Middleware4": {
					CircuitBreaker: &dynamic.CircuitBreaker{
						Expression: "foobar",
					},
				},
				"Middleware5": {
					DigestAuth: &dynamic.DigestAuth{
						Users: []string{
							"foobar",
							"fiibar",
						},
						UsersFile:    "foobar",
						RemoveHeader: true,
						Realm:        "foobar",
						HeaderField:  "foobar",
					},
				},
				"Middleware6": {
					Errors: &dynamic.ErrorPage{
						Status: []string{
							"foobar",
							"fiibar",
						},
						Service: "foobar",
						Query:   "foobar",
					},
				},
				"Middleware7": {
					ForwardAuth: &dynamic.ForwardAuth{
						Address: "foobar",
						TLS: &dynamic.ClientTLS{
							CA:                 "foobar",
							CAOptional:         true,
							Cert:               "foobar",
							Key:                "foobar",
							InsecureSkipVerify: true,
						},
						TrustForwardHeader: true,
						AuthResponseHeaders: []string{
							"foobar",
							"fiibar",
						},
						AuthRequestHeaders: []string{
							"foobar",
							"fiibar",
						},
					},
				},
				"Middleware8": {
					Headers: &dynamic.Headers{
						CustomRequestHeaders: map[string]string{
							"name0": "foobar",
							"name1": "foobar",
						},
						CustomResponseHeaders: map[string]string{
							"name0": "foobar",
							"name1": "foobar",
						},
						AccessControlAllowCredentials: true,
						AccessControlAllowHeaders: []string{
							"X-foobar",
							"X-fiibar",
						},
						AccessControlAllowMethods: []string{
							"GET",
							"PUT",
						},
						AccessControlAllowOriginList: []string{
							"foobar",
							"fiibar",
						},
						AccessControlAllowOriginListRegex: []string{
							"foobar",
							"fiibar",
						},
						AccessControlExposeHeaders: []string{
							"X-foobar",
							"X-fiibar",
						},
						AccessControlMaxAge: 200,
						AddVaryHeader:       true,
						AllowedHosts: []string{
							"foobar",
							"fiibar",
						},
						HostsProxyHeaders: []string{
							"foobar",
							"fiibar",
						},
						SSLRedirect:          true,
						SSLTemporaryRedirect: true,
						SSLHost:              "foobar",
						SSLProxyHeaders: map[string]string{
							"name0": "foobar",
							"name1": "foobar",
						},
						SSLForceHost:            true,
						STSSeconds:              42,
						STSIncludeSubdomains:    true,
						STSPreload:              true,
						ForceSTSHeader:          true,
						FrameDeny:               true,
						CustomFrameOptionsValue: "foobar",
						ContentTypeNosniff:      true,
						BrowserXSSFilter:        true,
						CustomBrowserXSSValue:   "foobar",
						ContentSecurityPolicy:   "foobar",
						PublicKey:               "foobar",
						ReferrerPolicy:          "foobar",
						FeaturePolicy:           "foobar",
						PermissionsPolicy:       "foobar",
						IsDevelopment:           true,
					},
				},
				"Middleware9": {
					IPWhiteList: &dynamic.IPWhiteList{
						SourceRange: []string{
							"foobar",
							"fiibar",
						},
						IPStrategy: &dynamic.IPStrategy{
							Depth: 42,
							ExcludedIPs: []string{
								"foobar",
								"fiibar",
							},
						},
					},
				},
				"Middleware20": {
					Plugin: map[string]dynamic.PluginConf{
						"tomato": {
							"aaa": "foo1",
							"bbb": "foo2",
						},
					},
				},
			},
			Services: map[string]*dynamic.Service{
				"Service0": {
					LoadBalancer: &dynamic.ServersLoadBalancer{
						Sticky: &dynamic.Sticky{
							Cookie: &dynamic.Cookie{
								Name:     "foobar",
								Secure:   true,
								HTTPOnly: false,
							},
						},
						Servers: []dynamic.Server{
							{
								Scheme: "foobar",
								Port:   "8080",
							},
						},
						HealthCheck: &dynamic.ServerHealthCheck{
							Scheme:   "foobar",
							Path:     "foobar",
							Port:     42,
							Interval: "foobar",
							Timeout:  "foobar",
							Hostname: "foobar",
							Headers: map[string]string{
								"name0": "foobar",
								"name1": "foobar",
							},
							FollowRedirects: func(v bool) *bool { return &v }(true),
						},
						PassHostHeader: func(v bool) *bool { return &v }(true),
						ResponseForwarding: &dynamic.ResponseForwarding{
							FlushInterval: "foobar",
						},
					},
				},
				"Service1": {
					LoadBalancer: &dynamic.ServersLoadBalancer{
						Servers: []dynamic.Server{
							{
								Scheme: "foobar",
								Port:   "8080",
							},
						},
						HealthCheck: &dynamic.ServerHealthCheck{
							Scheme:   "foobar",
							Path:     "foobar",
							Port:     42,
							Interval: "foobar",
							Timeout:  "foobar",
							Hostname: "foobar",
							Headers: map[string]string{
								"name0": "foobar",
								"name1": "foobar",
							},
							FollowRedirects: func(v bool) *bool { return &v }(true),
						},
						PassHostHeader: func(v bool) *bool { return &v }(true),
						ResponseForwarding: &dynamic.ResponseForwarding{
							FlushInterval: "foobar",
						},
					},
				},
			},
		},
	}

	assert.Equal(t, expected, configuration)
}

func TestEncodeConfiguration(t *testing.T) {
	configuration := &dynamic.Configuration{
		TCP: &dynamic.TCPConfiguration{
			Routers: map[string]*dynamic.TCPRouter{
				"Router0": {
					EntryPoints: []string{
						"foobar",
						"fiibar",
					},
					Service: "foobar",
					Rule:    "foobar",
					TLS: &dynamic.RouterTCPTLSConfig{
						Passthrough: false,
						Options:     "foo",
					},
				},
				"Router1": {
					EntryPoints: []string{
						"foobar",
						"fiibar",
					},
					Service: "foobar",
					Rule:    "foobar",
					TLS: &dynamic.RouterTCPTLSConfig{
						Passthrough: false,
						Options:     "foo",
					},
				},
			},
			Middlewares: map[string]*dynamic.TCPMiddleware{
				"Middleware0": {
					IPWhiteList: &dynamic.TCPIPWhiteList{
						SourceRange: []string{"foobar", "fiibar"},
					},
				},
			},
			Services: map[string]*dynamic.TCPService{
				"Service0": {
					LoadBalancer: &dynamic.TCPServersLoadBalancer{
						Servers: []dynamic.TCPServer{
							{
								Port: "42",
							},
						},
						TerminationDelay: func(i int) *int { return &i }(42),
					},
				},
				"Service1": {
					LoadBalancer: &dynamic.TCPServersLoadBalancer{
						Servers: []dynamic.TCPServer{
							{
								Port: "42",
							},
						},
						TerminationDelay: func(i int) *int { return &i }(42),
					},
				},
			},
		},
		UDP: &dynamic.UDPConfiguration{
			Routers: map[string]*dynamic.UDPRouter{
				"Router0": {
					EntryPoints: []string{
						"foobar",
						"fiibar",
					},
					Service: "foobar",
				},
				"Router1": {
					EntryPoints: []string{
						"foobar",
						"fiibar",
					},
					Service: "foobar",
				},
			},
			Services: map[string]*dynamic.UDPService{
				"Service0": {
					LoadBalancer: &dynamic.UDPServersLoadBalancer{
						Servers: []dynamic.UDPServer{
							{
								Port: "42",
							},
						},
					},
				},
				"Service1": {
					LoadBalancer: &dynamic.UDPServersLoadBalancer{
						Servers: []dynamic.UDPServer{
							{
								Port: "42",
							},
						},
					},
				},
			},
		},
		HTTP: &dynamic.HTTPConfiguration{
			Routers: map[string]*dynamic.Router{
				"Router0": {
					EntryPoints: []string{
						"foobar",
						"fiibar",
					},
					Middlewares: []string{
						"foobar",
						"fiibar",
					},
					Service:  "foobar",
					Rule:     "foobar",
					Priority: 42,
					TLS:      &dynamic.RouterTLSConfig{},
				},
				"Router1": {
					EntryPoints: []string{
						"foobar",
						"fiibar",
					},
					Middlewares: []string{
						"foobar",
						"fiibar",
					},
					Service:  "foobar",
					Rule:     "foobar",
					Priority: 42,
				},
			},
			Middlewares: map[string]*dynamic.Middleware{
				"Middleware0": {
					AddPrefix: &dynamic.AddPrefix{
						Prefix: "foobar",
					},
				},
				"Middleware1": {
					BasicAuth: &dynamic.BasicAuth{
						Users: []string{
							"foobar",
							"fiibar",
						},
						UsersFile:    "foobar",
						Realm:        "foobar",
						RemoveHeader: true,
						HeaderField:  "foobar",
					},
				},
				"Middleware10": {
					InFlightReq: &dynamic.InFlightReq{
						Amount: 42,
						SourceCriterion: &dynamic.SourceCriterion{
							IPStrategy: &dynamic.IPStrategy{
								Depth:       42,
								ExcludedIPs: []string{"foobar", "fiibar"},
							},
							RequestHeaderName: "foobar",
							RequestHost:       true,
						},
					},
				},
				"Middleware11": {
					PassTLSClientCert: &dynamic.PassTLSClientCert{
						PEM: true,
						Info: &dynamic.TLSClientCertificateInfo{
							NotAfter:     true,
							NotBefore:    true,
							SerialNumber: true,
							Subject: &dynamic.TLSCLientCertificateDNInfo{
								Country:         true,
								Province:        true,
								Locality:        true,
								Organization:    true,
								CommonName:      true,
								SerialNumber:    true,
								DomainComponent: true,
							},
							Issuer: &dynamic.TLSCLientCertificateDNInfo{
								Country:         true,
								Province:        true,
								Locality:        true,
								Organization:    true,
								CommonName:      true,
								SerialNumber:    true,
								DomainComponent: true,
							}, Sans: true,
						},
					},
				},
				"Middleware12": {
					RateLimit: &dynamic.RateLimit{
						Average: 42,
						Burst:   42,
						Period:  ptypes.Duration(time.Second),
						SourceCriterion: &dynamic.SourceCriterion{
							IPStrategy: &dynamic.IPStrategy{
								Depth:       42,
								ExcludedIPs: []string{"foobar", "foobar"},
							},
							RequestHeaderName: "foobar",
							RequestHost:       true,
						},
					},
				},
				"Middleware13": {
					RedirectRegex: &dynamic.RedirectRegex{
						Regex:       "foobar",
						Replacement: "foobar",
						Permanent:   true,
					},
				},
				"Middleware13b": {
					RedirectScheme: &dynamic.RedirectScheme{
						Scheme:    "https",
						Port:      "80",
						Permanent: true,
					},
				},
				"Middleware14": {
					ReplacePath: &dynamic.ReplacePath{
						Path: "foobar",
					},
				},
				"Middleware15": {
					ReplacePathRegex: &dynamic.ReplacePathRegex{
						Regex:       "foobar",
						Replacement: "foobar",
					},
				},
				"Middleware16": {
					Retry: &dynamic.Retry{
						Attempts:        42,
						InitialInterval: ptypes.Duration(time.Second),
					},
				},
				"Middleware17": {
					StripPrefix: &dynamic.StripPrefix{
						Prefixes: []string{
							"foobar",
							"fiibar",
						},
						ForceSlash: true,
					},
				},
				"Middleware18": {
					StripPrefixRegex: &dynamic.StripPrefixRegex{
						Regex: []string{
							"foobar",
							"fiibar",
						},
					},
				},
				"Middleware19": {
					Compress: &dynamic.Compress{
						MinimumBodySizeBytes: 42,
					},
				},
				"Middleware2": {
					Buffering: &dynamic.Buffering{
						MaxRequestBodyBytes:  42,
						MemRequestBodyBytes:  42,
						MaxResponseBodyBytes: 42,
						MemResponseBodyBytes: 42,
						RetryExpression:      "foobar",
					},
				},
				"Middleware20": {
					Plugin: map[string]dynamic.PluginConf{
						"tomato": {
							"aaa": "foo1",
							"bbb": "foo2",
						},
					},
				},
				"Middleware3": {
					Chain: &dynamic.Chain{
						Middlewares: []string{
							"foobar",
							"fiibar",
						},
					},
				},
				"Middleware4": {
					CircuitBreaker: &dynamic.CircuitBreaker{
						Expression: "foobar",
					},
				},
				"Middleware5": {
					DigestAuth: &dynamic.DigestAuth{
						Users: []string{
							"foobar",
							"fiibar",
						},
						UsersFile:    "foobar",
						RemoveHeader: true,
						Realm:        "foobar",
						HeaderField:  "foobar",
					},
				},
				"Middleware6": {
					Errors: &dynamic.ErrorPage{
						Status: []string{
							"foobar",
							"fiibar",
						},
						Service: "foobar",
						Query:   "foobar",
					},
				},
				"Middleware7": {
					ForwardAuth: &dynamic.ForwardAuth{
						Address: "foobar",
						TLS: &dynamic.ClientTLS{
							CA:                 "foobar",
							CAOptional:         true,
							Cert:               "foobar",
							Key:                "foobar",
							InsecureSkipVerify: true,
						},
						TrustForwardHeader: true,
						AuthResponseHeaders: []string{
							"foobar",
							"fiibar",
						},
						AuthRequestHeaders: []string{
							"foobar",
							"fiibar",
						},
					},
				},
				"Middleware8": {
					Headers: &dynamic.Headers{
						CustomRequestHeaders: map[string]string{
							"name0": "foobar",
							"name1": "foobar",
						},
						CustomResponseHeaders: map[string]string{
							"name0": "foobar",
							"name1": "foobar",
						},
						AccessControlAllowCredentials: true,
						AccessControlAllowHeaders: []string{
							"X-foobar",
							"X-fiibar",
						},
						AccessControlAllowMethods: []string{
							"GET",
							"PUT",
						},
						AccessControlAllowOriginList: []string{
							"foobar",
							"fiibar",
						},
						AccessControlAllowOriginListRegex: []string{
							"foobar",
							"fiibar",
						},
						AccessControlExposeHeaders: []string{
							"X-foobar",
							"X-fiibar",
						},
						AccessControlMaxAge: 200,
						AddVaryHeader:       true,
						AllowedHosts: []string{
							"foobar",
							"fiibar",
						},
						HostsProxyHeaders: []string{
							"foobar",
							"fiibar",
						},
						SSLRedirect:          true,
						SSLTemporaryRedirect: true,
						SSLHost:              "foobar",
						SSLProxyHeaders: map[string]string{
							"name0": "foobar",
							"name1": "foobar",
						},
						SSLForceHost:            true,
						STSSeconds:              42,
						STSIncludeSubdomains:    true,
						STSPreload:              true,
						ForceSTSHeader:          true,
						FrameDeny:               true,
						CustomFrameOptionsValue: "foobar",
						ContentTypeNosniff:      true,
						BrowserXSSFilter:        true,
						CustomBrowserXSSValue:   "foobar",
						ContentSecurityPolicy:   "foobar",
						PublicKey:               "foobar",
						ReferrerPolicy:          "foobar",
						FeaturePolicy:           "foobar",
						PermissionsPolicy:       "foobar",
						IsDevelopment:           true,
					},
				},
				"Middleware9": {
					IPWhiteList: &dynamic.IPWhiteList{
						SourceRange: []string{
							"foobar",
							"fiibar",
						},
						IPStrategy: &dynamic.IPStrategy{
							Depth: 42,
							ExcludedIPs: []string{
								"foobar",
								"fiibar",
							},
						},
					},
				},
			},
			Services: map[string]*dynamic.Service{
				"Service0": {
					LoadBalancer: &dynamic.ServersLoadBalancer{
						Sticky: &dynamic.Sticky{
							Cookie: &dynamic.Cookie{
								Name:     "foobar",
								HTTPOnly: true,
							},
						},
						Servers: []dynamic.Server{
							{
								Scheme: "foobar",
								Port:   "8080",
							},
						},
						HealthCheck: &dynamic.ServerHealthCheck{
							Scheme:   "foobar",
							Path:     "foobar",
							Port:     42,
							Interval: "foobar",
							Timeout:  "foobar",
							Hostname: "foobar",
							Headers: map[string]string{
								"name0": "foobar",
								"name1": "foobar",
							},
						},
						PassHostHeader: func(v bool) *bool { return &v }(true),
						ResponseForwarding: &dynamic.ResponseForwarding{
							FlushInterval: "foobar",
						},
					},
				},
				"Service1": {
					LoadBalancer: &dynamic.ServersLoadBalancer{
						Servers: []dynamic.Server{
							{
								Scheme: "foobar",
								Port:   "8080",
							},
						},
						HealthCheck: &dynamic.ServerHealthCheck{
							Scheme:   "foobar",
							Path:     "foobar",
							Port:     42,
							Interval: "foobar",
							Timeout:  "foobar",
							Hostname: "foobar",
							Headers: map[string]string{
								"name0": "foobar",
								"name1": "foobar",
							},
						},
						PassHostHeader: func(v bool) *bool { return &v }(true),
						ResponseForwarding: &dynamic.ResponseForwarding{
							FlushInterval: "foobar",
						},
					},
				},
			},
		},
	}

	labels, err := EncodeConfiguration(configuration)
	require.NoError(t, err)

	expected := map[string]string{
		"traefik.HTTP.Middlewares.Middleware0.AddPrefix.Prefix":                                    "foobar",
		"traefik.HTTP.Middlewares.Middleware1.BasicAuth.HeaderField":                               "foobar",
		"traefik.HTTP.Middlewares.Middleware1.BasicAuth.Realm":                                     "foobar",
		"traefik.HTTP.Middlewares.Middleware1.BasicAuth.RemoveHeader":                              "true",
		"traefik.HTTP.Middlewares.Middleware1.BasicAuth.Users":                                     "foobar, fiibar",
		"traefik.HTTP.Middlewares.Middleware1.BasicAuth.UsersFile":                                 "foobar",
		"traefik.HTTP.Middlewares.Middleware2.Buffering.MaxRequestBodyBytes":                       "42",
		"traefik.HTTP.Middlewares.Middleware2.Buffering.MaxResponseBodyBytes":                      "42",
		"traefik.HTTP.Middlewares.Middleware2.Buffering.MemRequestBodyBytes":                       "42",
		"traefik.HTTP.Middlewares.Middleware2.Buffering.MemResponseBodyBytes":                      "42",
		"traefik.HTTP.Middlewares.Middleware2.Buffering.RetryExpression":                           "foobar",
		"traefik.HTTP.Middlewares.Middleware3.Chain.Middlewares":                                   "foobar, fiibar",
		"traefik.HTTP.Middlewares.Middleware4.CircuitBreaker.Expression":                           "foobar",
		"traefik.HTTP.Middlewares.Middleware5.DigestAuth.HeaderField":                              "foobar",
		"traefik.HTTP.Middlewares.Middleware5.DigestAuth.Realm":                                    "foobar",
		"traefik.HTTP.Middlewares.Middleware5.DigestAuth.RemoveHeader":                             "true",
		"traefik.HTTP.Middlewares.Middleware5.DigestAuth.Users":                                    "foobar, fiibar",
		"traefik.HTTP.Middlewares.Middleware5.DigestAuth.UsersFile":                                "foobar",
		"traefik.HTTP.Middlewares.Middleware6.Errors.Query":                                        "foobar",
		"traefik.HTTP.Middlewares.Middleware6.Errors.Service":                                      "foobar",
		"traefik.HTTP.Middlewares.Middleware6.Errors.Status":                                       "foobar, fiibar",
		"traefik.HTTP.Middlewares.Middleware7.ForwardAuth.Address":                                 "foobar",
		"traefik.HTTP.Middlewares.Middleware7.ForwardAuth.AuthResponseHeaders":                     "foobar, fiibar",
		"traefik.HTTP.Middlewares.Middleware7.ForwardAuth.AuthRequestHeaders":                      "foobar, fiibar",
		"traefik.HTTP.Middlewares.Middleware7.ForwardAuth.TLS.CA":                                  "foobar",
		"traefik.HTTP.Middlewares.Middleware7.ForwardAuth.TLS.CAOptional":                          "true",
		"traefik.HTTP.Middlewares.Middleware7.ForwardAuth.TLS.Cert":                                "foobar",
		"traefik.HTTP.Middlewares.Middleware7.ForwardAuth.TLS.InsecureSkipVerify":                  "true",
		"traefik.HTTP.Middlewares.Middleware7.ForwardAuth.TLS.Key":                                 "foobar",
		"traefik.HTTP.Middlewares.Middleware7.ForwardAuth.TrustForwardHeader":                      "true",
		"traefik.HTTP.Middlewares.Middleware8.Headers.AccessControlAllowCredentials":               "true",
		"traefik.HTTP.Middlewares.Middleware8.Headers.AccessControlAllowHeaders":                   "X-foobar, X-fiibar",
		"traefik.HTTP.Middlewares.Middleware8.Headers.AccessControlAllowMethods":                   "GET, PUT",
		"traefik.HTTP.Middlewares.Middleware8.Headers.AccessControlAllowOriginList":                "foobar, fiibar",
		"traefik.HTTP.Middlewares.Middleware8.Headers.AccessControlAllowOriginListRegex":           "foobar, fiibar",
		"traefik.HTTP.Middlewares.Middleware8.Headers.AccessControlExposeHeaders":                  "X-foobar, X-fiibar",
		"traefik.HTTP.Middlewares.Middleware8.Headers.AccessControlMaxAge":                         "200",
		"traefik.HTTP.Middlewares.Middleware8.Headers.AddVaryHeader":                               "true",
		"traefik.HTTP.Middlewares.Middleware8.Headers.AllowedHosts":                                "foobar, fiibar",
		"traefik.HTTP.Middlewares.Middleware8.Headers.BrowserXSSFilter":                            "true",
		"traefik.HTTP.Middlewares.Middleware8.Headers.ContentSecurityPolicy":                       "foobar",
		"traefik.HTTP.Middlewares.Middleware8.Headers.ContentTypeNosniff":                          "true",
		"traefik.HTTP.Middlewares.Middleware8.Headers.CustomBrowserXSSValue":                       "foobar",
		"traefik.HTTP.Middlewares.Middleware8.Headers.CustomFrameOptionsValue":                     "foobar",
		"traefik.HTTP.Middlewares.Middleware8.Headers.CustomRequestHeaders.name0":                  "foobar",
		"traefik.HTTP.Middlewares.Middleware8.Headers.CustomRequestHeaders.name1":                  "foobar",
		"traefik.HTTP.Middlewares.Middleware8.Headers.CustomResponseHeaders.name0":                 "foobar",
		"traefik.HTTP.Middlewares.Middleware8.Headers.CustomResponseHeaders.name1":                 "foobar",
		"traefik.HTTP.Middlewares.Middleware8.Headers.ForceSTSHeader":                              "true",
		"traefik.HTTP.Middlewares.Middleware8.Headers.FrameDeny":                                   "true",
		"traefik.HTTP.Middlewares.Middleware8.Headers.HostsProxyHeaders":                           "foobar, fiibar",
		"traefik.HTTP.Middlewares.Middleware8.Headers.IsDevelopment":                               "true",
		"traefik.HTTP.Middlewares.Middleware8.Headers.PublicKey":                                   "foobar",
		"traefik.HTTP.Middlewares.Middleware8.Headers.ReferrerPolicy":                              "foobar",
		"traefik.HTTP.Middlewares.Middleware8.Headers.FeaturePolicy":                               "foobar",
		"traefik.HTTP.Middlewares.Middleware8.Headers.PermissionsPolicy":                           "foobar",
		"traefik.HTTP.Middlewares.Middleware8.Headers.SSLForceHost":                                "true",
		"traefik.HTTP.Middlewares.Middleware8.Headers.SSLHost":                                     "foobar",
		"traefik.HTTP.Middlewares.Middleware8.Headers.SSLProxyHeaders.name0":                       "foobar",
		"traefik.HTTP.Middlewares.Middleware8.Headers.SSLProxyHeaders.name1":                       "foobar",
		"traefik.HTTP.Middlewares.Middleware8.Headers.SSLRedirect":                                 "true",
		"traefik.HTTP.Middlewares.Middleware8.Headers.SSLTemporaryRedirect":                        "true",
		"traefik.HTTP.Middlewares.Middleware8.Headers.STSIncludeSubdomains":                        "true",
		"traefik.HTTP.Middlewares.Middleware8.Headers.STSPreload":                                  "true",
		"traefik.HTTP.Middlewares.Middleware8.Headers.STSSeconds":                                  "42",
		"traefik.HTTP.Middlewares.Middleware9.IPWhiteList.IPStrategy.Depth":                        "42",
		"traefik.HTTP.Middlewares.Middleware9.IPWhiteList.IPStrategy.ExcludedIPs":                  "foobar, fiibar",
		"traefik.HTTP.Middlewares.Middleware9.IPWhiteList.SourceRange":                             "foobar, fiibar",
		"traefik.HTTP.Middlewares.Middleware10.InFlightReq.Amount":                                 "42",
		"traefik.HTTP.Middlewares.Middleware10.InFlightReq.SourceCriterion.IPStrategy.Depth":       "42",
		"traefik.HTTP.Middlewares.Middleware10.InFlightReq.SourceCriterion.IPStrategy.ExcludedIPs": "foobar, fiibar",
		"traefik.HTTP.Middlewares.Middleware10.InFlightReq.SourceCriterion.RequestHeaderName":      "foobar",
		"traefik.HTTP.Middlewares.Middleware10.InFlightReq.SourceCriterion.RequestHost":            "true",
		"traefik.HTTP.Middlewares.Middleware11.PassTLSClientCert.Info.NotAfter":                    "true",
		"traefik.HTTP.Middlewares.Middleware11.PassTLSClientCert.Info.NotBefore":                   "true",
		"traefik.HTTP.Middlewares.Middleware11.PassTLSClientCert.Info.Sans":                        "true",
		"traefik.HTTP.Middlewares.Middleware11.PassTLSClientCert.Info.SerialNumber":                "true",
		"traefik.HTTP.Middlewares.Middleware11.PassTLSClientCert.Info.Subject.Country":             "true",
		"traefik.HTTP.Middlewares.Middleware11.PassTLSClientCert.Info.Subject.Province":            "true",
		"traefik.HTTP.Middlewares.Middleware11.PassTLSClientCert.Info.Subject.Locality":            "true",
		"traefik.HTTP.Middlewares.Middleware11.PassTLSClientCert.Info.Subject.Organization":        "true",
		"traefik.HTTP.Middlewares.Middleware11.PassTLSClientCert.Info.Subject.CommonName":          "true",
		"traefik.HTTP.Middlewares.Middleware11.PassTLSClientCert.Info.Subject.SerialNumber":        "true",
		"traefik.HTTP.Middlewares.Middleware11.PassTLSClientCert.Info.Subject.DomainComponent":     "true",
		"traefik.HTTP.Middlewares.Middleware11.PassTLSClientCert.Info.Issuer.Country":              "true",
		"traefik.HTTP.Middlewares.Middleware11.PassTLSClientCert.Info.Issuer.Province":             "true",
		"traefik.HTTP.Middlewares.Middleware11.PassTLSClientCert.Info.Issuer.Locality":             "true",
		"traefik.HTTP.Middlewares.Middleware11.PassTLSClientCert.Info.Issuer.Organization":         "true",
		"traefik.HTTP.Middlewares.Middleware11.PassTLSClientCert.Info.Issuer.CommonName":           "true",
		"traefik.HTTP.Middlewares.Middleware11.PassTLSClientCert.Info.Issuer.SerialNumber":         "true",
		"traefik.HTTP.Middlewares.Middleware11.PassTLSClientCert.Info.Issuer.DomainComponent":      "true",
		"traefik.HTTP.Middlewares.Middleware11.PassTLSClientCert.PEM":                              "true",
		"traefik.HTTP.Middlewares.Middleware12.RateLimit.Average":                                  "42",
		"traefik.HTTP.Middlewares.Middleware12.RateLimit.Period":                                   "1000000000",
		"traefik.HTTP.Middlewares.Middleware12.RateLimit.Burst":                                    "42",
		"traefik.HTTP.Middlewares.Middleware12.RateLimit.SourceCriterion.RequestHeaderName":        "foobar",
		"traefik.HTTP.Middlewares.Middleware12.RateLimit.SourceCriterion.RequestHost":              "true",
		"traefik.HTTP.Middlewares.Middleware12.RateLimit.SourceCriterion.IPStrategy.Depth":         "42",
		"traefik.HTTP.Middlewares.Middleware12.RateLimit.SourceCriterion.IPStrategy.ExcludedIPs":   "foobar, foobar",
		"traefik.HTTP.Middlewares.Middleware13.RedirectRegex.Regex":                                "foobar",
		"traefik.HTTP.Middlewares.Middleware13.RedirectRegex.Replacement":                          "foobar",
		"traefik.HTTP.Middlewares.Middleware13.RedirectRegex.Permanent":                            "true",
		"traefik.HTTP.Middlewares.Middleware13b.RedirectScheme.Scheme":                             "https",
		"traefik.HTTP.Middlewares.Middleware13b.RedirectScheme.Port":                               "80",
		"traefik.HTTP.Middlewares.Middleware13b.RedirectScheme.Permanent":                          "true",
		"traefik.HTTP.Middlewares.Middleware14.ReplacePath.Path":                                   "foobar",
		"traefik.HTTP.Middlewares.Middleware15.ReplacePathRegex.Regex":                             "foobar",
		"traefik.HTTP.Middlewares.Middleware15.ReplacePathRegex.Replacement":                       "foobar",
		"traefik.HTTP.Middlewares.Middleware16.Retry.Attempts":                                     "42",
		"traefik.HTTP.Middlewares.Middleware16.Retry.InitialInterval":                              "1000000000",
		"traefik.HTTP.Middlewares.Middleware17.StripPrefix.Prefixes":                               "foobar, fiibar",
		"traefik.HTTP.Middlewares.Middleware17.StripPrefix.ForceSlash":                             "true",
		"traefik.HTTP.Middlewares.Middleware18.StripPrefixRegex.Regex":                             "foobar, fiibar",
		"traefik.HTTP.Middlewares.Middleware19.Compress.MinimumBodySizeBytes":                      "42",
		"traefik.HTTP.Middlewares.Middleware20.Plugin.tomato.aaa":                                  "foo1",
		"traefik.HTTP.Middlewares.Middleware20.Plugin.tomato.bbb":                                  "foo2",

		"traefik.HTTP.Routers.Router0.EntryPoints": "foobar, fiibar",
		"traefik.HTTP.Routers.Router0.Middlewares": "foobar, fiibar",
		"traefik.HTTP.Routers.Router0.Priority":    "42",
		"traefik.HTTP.Routers.Router0.Rule":        "foobar",
		"traefik.HTTP.Routers.Router0.Service":     "foobar",
		"traefik.HTTP.Routers.Router0.TLS":         "true",
		"traefik.HTTP.Routers.Router1.EntryPoints": "foobar, fiibar",
		"traefik.HTTP.Routers.Router1.Middlewares": "foobar, fiibar",
		"traefik.HTTP.Routers.Router1.Priority":    "42",
		"traefik.HTTP.Routers.Router1.Rule":        "foobar",
		"traefik.HTTP.Routers.Router1.Service":     "foobar",

		"traefik.HTTP.Services.Service0.LoadBalancer.HealthCheck.Headers.name1":        "foobar",
		"traefik.HTTP.Services.Service0.LoadBalancer.HealthCheck.Hostname":             "foobar",
		"traefik.HTTP.Services.Service0.LoadBalancer.HealthCheck.Interval":             "foobar",
		"traefik.HTTP.Services.Service0.LoadBalancer.HealthCheck.Path":                 "foobar",
		"traefik.HTTP.Services.Service0.LoadBalancer.HealthCheck.Port":                 "42",
		"traefik.HTTP.Services.Service0.LoadBalancer.HealthCheck.Scheme":               "foobar",
		"traefik.HTTP.Services.Service0.LoadBalancer.HealthCheck.Timeout":              "foobar",
		"traefik.HTTP.Services.Service0.LoadBalancer.PassHostHeader":                   "true",
		"traefik.HTTP.Services.Service0.LoadBalancer.ResponseForwarding.FlushInterval": "foobar",
		"traefik.HTTP.Services.Service0.LoadBalancer.server.Port":                      "8080",
		"traefik.HTTP.Services.Service0.LoadBalancer.server.Scheme":                    "foobar",
		"traefik.HTTP.Services.Service0.LoadBalancer.Sticky.Cookie.Name":               "foobar",
		"traefik.HTTP.Services.Service0.LoadBalancer.Sticky.Cookie.HTTPOnly":           "true",
		"traefik.HTTP.Services.Service0.LoadBalancer.Sticky.Cookie.Secure":             "false",
		"traefik.HTTP.Services.Service1.LoadBalancer.HealthCheck.Headers.name0":        "foobar",
		"traefik.HTTP.Services.Service1.LoadBalancer.HealthCheck.Headers.name1":        "foobar",
		"traefik.HTTP.Services.Service1.LoadBalancer.HealthCheck.Hostname":             "foobar",
		"traefik.HTTP.Services.Service1.LoadBalancer.HealthCheck.Interval":             "foobar",
		"traefik.HTTP.Services.Service1.LoadBalancer.HealthCheck.Path":                 "foobar",
		"traefik.HTTP.Services.Service1.LoadBalancer.HealthCheck.Port":                 "42",
		"traefik.HTTP.Services.Service1.LoadBalancer.HealthCheck.Scheme":               "foobar",
		"traefik.HTTP.Services.Service1.LoadBalancer.HealthCheck.Timeout":              "foobar",
		"traefik.HTTP.Services.Service1.LoadBalancer.PassHostHeader":                   "true",
		"traefik.HTTP.Services.Service1.LoadBalancer.ResponseForwarding.FlushInterval": "foobar",
		"traefik.HTTP.Services.Service1.LoadBalancer.server.Port":                      "8080",
		"traefik.HTTP.Services.Service1.LoadBalancer.server.Scheme":                    "foobar",
		"traefik.HTTP.Services.Service0.LoadBalancer.HealthCheck.Headers.name0":        "foobar",

		"traefik.TCP.Middlewares.Middleware0.IPWhiteList.SourceRange": "foobar, fiibar",
		"traefik.TCP.Routers.Router0.Rule":                            "foobar",
		"traefik.TCP.Routers.Router0.EntryPoints":                     "foobar, fiibar",
		"traefik.TCP.Routers.Router0.Service":                         "foobar",
		"traefik.TCP.Routers.Router0.TLS.Passthrough":                 "false",
		"traefik.TCP.Routers.Router0.TLS.Options":                     "foo",
		"traefik.TCP.Routers.Router1.Rule":                            "foobar",
		"traefik.TCP.Routers.Router1.EntryPoints":                     "foobar, fiibar",
		"traefik.TCP.Routers.Router1.Service":                         "foobar",
		"traefik.TCP.Routers.Router1.TLS.Passthrough":                 "false",
		"traefik.TCP.Routers.Router1.TLS.Options":                     "foo",
		"traefik.TCP.Services.Service0.LoadBalancer.server.Port":      "42",
		"traefik.TCP.Services.Service0.LoadBalancer.TerminationDelay": "42",
		"traefik.TCP.Services.Service1.LoadBalancer.server.Port":      "42",
		"traefik.TCP.Services.Service1.LoadBalancer.TerminationDelay": "42",

		"traefik.UDP.Routers.Router0.EntryPoints":                "foobar, fiibar",
		"traefik.UDP.Routers.Router0.Service":                    "foobar",
		"traefik.UDP.Routers.Router1.EntryPoints":                "foobar, fiibar",
		"traefik.UDP.Routers.Router1.Service":                    "foobar",
		"traefik.UDP.Services.Service0.LoadBalancer.server.Port": "42",
		"traefik.UDP.Services.Service1.LoadBalancer.server.Port": "42",
	}

	for key, val := range expected {
		if _, ok := labels[key]; !ok {
			fmt.Println("missing in labels:", key, val)
		}
	}

	for key, val := range labels {
		if _, ok := expected[key]; !ok {
			fmt.Println("missing in expected:", key, val)
		}
	}
	assert.Equal(t, expected, labels)
}
