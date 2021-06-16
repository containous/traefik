package gateway

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/provider"
	"github.com/traefik/traefik/v2/pkg/tls"
	"sigs.k8s.io/gateway-api/apis/v1alpha1"
)

var _ provider.Provider = (*Provider)(nil)

func Bool(v bool) *bool { return &v }

func TestLoadHTTPRoutes(t *testing.T) {
	testCases := []struct {
		desc         string
		ingressClass string
		paths        []string
		expected     *dynamic.Configuration
		entryPoints  map[string]Entrypoint
	}{
		{
			desc: "Empty",
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:     map[string]*dynamic.TCPRouter{},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services:    map[string]*dynamic.TCPService{},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers:     map[string]*dynamic.Router{},
					Middlewares: map[string]*dynamic.Middleware{},
					Services:    map[string]*dynamic.Service{},
				},
				TLS: &dynamic.TLSConfiguration{},
			},
		},
		{
			desc:  "Empty because missing entry point",
			paths: []string{"services.yml", "httproute/simple.yml"},
			entryPoints: map[string]Entrypoint{"web": {
				Address: ":443",
			}},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:     map[string]*dynamic.TCPRouter{},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services:    map[string]*dynamic.TCPService{},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers:     map[string]*dynamic.Router{},
					Middlewares: map[string]*dynamic.Middleware{},
					Services:    map[string]*dynamic.Service{},
				},
				TLS: &dynamic.TLSConfiguration{},
			},
		},
		{
			desc:  "Empty because no http route defined",
			paths: []string{"services.yml", "httproute/without_httproute.yml"},
			entryPoints: map[string]Entrypoint{"web": {
				Address: ":80",
			}},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:     map[string]*dynamic.TCPRouter{},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services:    map[string]*dynamic.TCPService{},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers:     map[string]*dynamic.Router{},
					Middlewares: map[string]*dynamic.Middleware{},
					Services:    map[string]*dynamic.Service{},
				},
				TLS: &dynamic.TLSConfiguration{},
			},
		},
		{
			desc: "Empty caused by missing GatewayClass",
			entryPoints: map[string]Entrypoint{"web": {
				Address: ":80",
			}},
			paths: []string{"services.yml", "httproute/without_gatewayclass.yml"},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:     map[string]*dynamic.TCPRouter{},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services:    map[string]*dynamic.TCPService{},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers:     map[string]*dynamic.Router{},
					Middlewares: map[string]*dynamic.Middleware{},
					Services:    map[string]*dynamic.Service{},
				},
				TLS: &dynamic.TLSConfiguration{},
			},
		},
		{
			desc: "Empty caused by unknown GatewayClass controller name",
			entryPoints: map[string]Entrypoint{"web": {
				Address: ":80",
			}},
			paths: []string{"services.yml", "httproute/gatewayclass_with_unknown_controller.yml"},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:     map[string]*dynamic.TCPRouter{},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services:    map[string]*dynamic.TCPService{},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers:     map[string]*dynamic.Router{},
					Middlewares: map[string]*dynamic.Middleware{},
					Services:    map[string]*dynamic.Service{},
				},
				TLS: &dynamic.TLSConfiguration{},
			},
		},
		{
			desc: "Empty caused by multiport service with wrong TargetPort",
			entryPoints: map[string]Entrypoint{"web": {
				Address: ":80",
			}},
			paths: []string{"services.yml", "httproute/with_wrong_service_port.yml"},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:     map[string]*dynamic.TCPRouter{},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services:    map[string]*dynamic.TCPService{},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers:     map[string]*dynamic.Router{},
					Middlewares: map[string]*dynamic.Middleware{},
					Services:    map[string]*dynamic.Service{},
				},
				TLS: &dynamic.TLSConfiguration{},
			},
		},
		{
			desc: "Empty caused by HTTPS without TLS",
			entryPoints: map[string]Entrypoint{"websecure": {
				Address: ":443",
			}},
			paths: []string{"services.yml", "httproute/with_protocol_https_without_tls.yml"},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:     map[string]*dynamic.TCPRouter{},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services:    map[string]*dynamic.TCPService{},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers:     map[string]*dynamic.Router{},
					Middlewares: map[string]*dynamic.Middleware{},
					Services:    map[string]*dynamic.Service{},
				},
				TLS: &dynamic.TLSConfiguration{},
			},
		},
		{
			desc: "Empty caused by HTTPS with TLS passthrough",
			entryPoints: map[string]Entrypoint{"websecure": {
				Address: ":443",
			}},
			paths: []string{"services.yml", "httproute/with_protocol_https_with_tls_passthrough.yml"},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:     map[string]*dynamic.TCPRouter{},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services:    map[string]*dynamic.TCPService{},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers:     map[string]*dynamic.Router{},
					Middlewares: map[string]*dynamic.Middleware{},
					Services:    map[string]*dynamic.Service{},
				},
				TLS: &dynamic.TLSConfiguration{},
			},
		},
		{
			desc: "Empty caused by HTTPRoute with protocol TLS",
			entryPoints: map[string]Entrypoint{"websecure": {
				Address: ":443",
			}},
			paths: []string{"services.yml", "httproute/with_protocol_tls.yml"},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:     map[string]*dynamic.TCPRouter{},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services:    map[string]*dynamic.TCPService{},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers:     map[string]*dynamic.Router{},
					Middlewares: map[string]*dynamic.Middleware{},
					Services:    map[string]*dynamic.Service{},
				},
				TLS: &dynamic.TLSConfiguration{},
			},
		},
		{
			desc: "Empty caused by HTTPRoute with protocol TCP",
			entryPoints: map[string]Entrypoint{"websecure": {
				Address: ":443",
			}},
			paths: []string{"services.yml", "httproute/with_protocol_tcp.yml"},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:     map[string]*dynamic.TCPRouter{},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services:    map[string]*dynamic.TCPService{},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers:     map[string]*dynamic.Router{},
					Middlewares: map[string]*dynamic.Middleware{},
					Services:    map[string]*dynamic.Service{},
				},
				TLS: &dynamic.TLSConfiguration{},
			},
		},
		{
			desc: "Empty caused by TCPRoute with protocol HTTP",
			entryPoints: map[string]Entrypoint{"websecure": {
				Address: ":9000",
			}},
			paths: []string{"services.yml", "tcproute/with_protocol_http.yml"},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:     map[string]*dynamic.TCPRouter{},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services:    map[string]*dynamic.TCPService{},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers:     map[string]*dynamic.Router{},
					Middlewares: map[string]*dynamic.Middleware{},
					Services:    map[string]*dynamic.Service{},
				},
				TLS: &dynamic.TLSConfiguration{},
			},
		},
		{
			desc: "Empty caused by TCPRoute with protocol HTTPS",
			entryPoints: map[string]Entrypoint{"websecure": {
				Address: ":9000",
			}},
			paths: []string{"services.yml", "tcproute/with_protocol_https.yml"},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:     map[string]*dynamic.TCPRouter{},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services:    map[string]*dynamic.TCPService{},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers:     map[string]*dynamic.Router{},
					Middlewares: map[string]*dynamic.Middleware{},
					Services:    map[string]*dynamic.Service{},
				},
				TLS: &dynamic.TLSConfiguration{},
			},
		},
		{
			desc: "Empty caused by TLSRoute with protocol TCP",
			entryPoints: map[string]Entrypoint{"websecure": {
				Address: ":9001",
			}},
			paths: []string{"services.yml", "tlsroute/with_protocol_tcp.yml"},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:     map[string]*dynamic.TCPRouter{},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services:    map[string]*dynamic.TCPService{},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers:     map[string]*dynamic.Router{},
					Middlewares: map[string]*dynamic.Middleware{},
					Services:    map[string]*dynamic.Service{},
				},
				TLS: &dynamic.TLSConfiguration{},
			},
		},
		{
			desc: "Empty caused by TLSRoute with protocol HTTP",
			entryPoints: map[string]Entrypoint{"websecure": {
				Address: ":9001",
			}},
			paths: []string{"services.yml", "tlsroute/with_protocol_http.yml"},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:     map[string]*dynamic.TCPRouter{},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services:    map[string]*dynamic.TCPService{},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers:     map[string]*dynamic.Router{},
					Middlewares: map[string]*dynamic.Middleware{},
					Services:    map[string]*dynamic.Service{},
				},
				TLS: &dynamic.TLSConfiguration{},
			},
		},
		{
			desc: "Empty caused by TLSRoute with protocol HTTPS",
			entryPoints: map[string]Entrypoint{"websecure": {
				Address: ":9001",
			}},
			paths: []string{"services.yml", "tlsroute/with_protocol_https.yml"},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:     map[string]*dynamic.TCPRouter{},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services:    map[string]*dynamic.TCPService{},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers:     map[string]*dynamic.Router{},
					Middlewares: map[string]*dynamic.Middleware{},
					Services:    map[string]*dynamic.Service{},
				},
				TLS: &dynamic.TLSConfiguration{},
			},
		},
		{
			desc: "Empty caused use http entrypoint with tls activated with HTTPRoute",
			entryPoints: map[string]Entrypoint{"websecure": {
				Address:        ":443",
				HasHTTPTLSConf: true,
			}},
			paths: []string{"services.yml", "httproute/simple_with_tls_entrypoint.yml"},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:     map[string]*dynamic.TCPRouter{},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services:    map[string]*dynamic.TCPService{},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers:     map[string]*dynamic.Router{},
					Middlewares: map[string]*dynamic.Middleware{},
					Services:    map[string]*dynamic.Service{},
				},
				TLS: &dynamic.TLSConfiguration{},
			},
		},
		{
			desc: "Empty caused unsupported HTTPRoute rule",
			entryPoints: map[string]Entrypoint{"web": {
				Address: ":80",
			}},
			paths: []string{"services.yml", "httproute/simple_with_bad_rule.yml"},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:     map[string]*dynamic.TCPRouter{},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services:    map[string]*dynamic.TCPService{},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers:     map[string]*dynamic.Router{},
					Middlewares: map[string]*dynamic.Middleware{},
					Services:    map[string]*dynamic.Service{},
				},
				TLS: &dynamic.TLSConfiguration{},
			},
		},
		{
			desc:  "Empty because no tcp route defined tls protocol",
			paths: []string{"services.yml", "tcproute/without_tcproute_tls_protocol.yml"},
			entryPoints: map[string]Entrypoint{"TCP": {
				Address: ":8080",
			}},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:     map[string]*dynamic.TCPRouter{},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services:    map[string]*dynamic.TCPService{},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers:     map[string]*dynamic.Router{},
					Middlewares: map[string]*dynamic.Middleware{},
					Services:    map[string]*dynamic.Service{},
				},
				TLS: &dynamic.TLSConfiguration{
					Certificates: []*tls.CertAndStores{
						{
							Certificate: tls.Certificate{
								CertFile: tls.FileOrContent("-----BEGIN CERTIFICATE-----\n-----END CERTIFICATE-----"),
								KeyFile:  tls.FileOrContent("-----BEGIN PRIVATE KEY-----\n-----END PRIVATE KEY-----"),
							},
						},
					},
				},
			},
		},
		{
			desc:  "Simple HTTPRoute, with foo entrypoint",
			paths: []string{"services.yml", "httproute/simple.yml"},
			entryPoints: map[string]Entrypoint{"web": {
				Address: ":80",
			}},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:     map[string]*dynamic.TCPRouter{},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services:    map[string]*dynamic.TCPService{},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers: map[string]*dynamic.Router{
						"default-http-app-1-my-gateway-web-1c0cf64bde37d9d0df06": {
							EntryPoints: []string{"web"},
							Service:     "default-http-app-1-my-gateway-web-1c0cf64bde37d9d0df06-wrr",
							Rule:        "Host(`foo.com`) && Path(`/bar`)",
						},
					},
					Middlewares: map[string]*dynamic.Middleware{},
					Services: map[string]*dynamic.Service{
						"default-http-app-1-my-gateway-web-1c0cf64bde37d9d0df06-wrr": {
							Weighted: &dynamic.WeightedRoundRobin{
								Services: []dynamic.WRRService{
									{
										Name:   "default-whoami-80",
										Weight: func(i int) *int { return &i }(1),
									},
								},
							},
						},
						"default-whoami-80": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								Servers: []dynamic.Server{
									{
										URL: "http://10.10.0.1:80",
									},
									{
										URL: "http://10.10.0.2:80",
									},
								},
								PassHostHeader: Bool(true),
							},
						},
					},
				},
				TLS: &dynamic.TLSConfiguration{},
			},
		},
		{
			desc:  "Simple HTTPRoute, with api@internal service",
			paths: []string{"services.yml", "httproute/simple_to_api_internal.yml"},
			entryPoints: map[string]Entrypoint{"web": {
				Address: ":80",
			}},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:     map[string]*dynamic.TCPRouter{},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services:    map[string]*dynamic.TCPService{},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers: map[string]*dynamic.Router{
						"default-http-app-1-my-gateway-web-1c0cf64bde37d9d0df06": {
							EntryPoints: []string{"web"},
							Service:     "api@internal",
							Rule:        "Host(`foo.com`) && Path(`/bar`)",
						},
					},
					Middlewares: map[string]*dynamic.Middleware{},
					Services:    map[string]*dynamic.Service{},
				},
				TLS: &dynamic.TLSConfiguration{},
			},
		},
		{
			desc:  "Simple HTTPRoute, with myservice@file service",
			paths: []string{"services.yml", "httproute/simple_cross_provider.yml"},
			entryPoints: map[string]Entrypoint{"web": {
				Address: ":80",
			}},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:     map[string]*dynamic.TCPRouter{},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services:    map[string]*dynamic.TCPService{},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers: map[string]*dynamic.Router{
						"default-http-app-1-my-gateway-web-1c0cf64bde37d9d0df06": {
							EntryPoints: []string{"web"},
							Service:     "default-http-app-1-my-gateway-web-1c0cf64bde37d9d0df06-wrr",
							Rule:        "Host(`foo.com`) && Path(`/bar`)",
						},
					},
					Middlewares: map[string]*dynamic.Middleware{},
					Services: map[string]*dynamic.Service{
						"default-http-app-1-my-gateway-web-1c0cf64bde37d9d0df06-wrr": {
							Weighted: &dynamic.WeightedRoundRobin{
								Services: []dynamic.WRRService{
									{
										Name:   "service@file",
										Weight: func(i int) *int { return &i }(1),
									},
									{
										Name:   "default-whoami-80",
										Weight: func(i int) *int { return &i }(1),
									},
								},
							},
						},
						"default-whoami-80": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								Servers: []dynamic.Server{
									{
										URL: "http://10.10.0.1:80",
									},
									{
										URL: "http://10.10.0.2:80",
									},
								},
								PassHostHeader: Bool(true),
							},
						},
					},
				},
				TLS: &dynamic.TLSConfiguration{},
			},
		},
		{
			desc:  "Simple HTTPRoute with protocol HTTPS",
			paths: []string{"services.yml", "httproute/with_protocol_https.yml"},
			entryPoints: map[string]Entrypoint{"websecure": {
				Address: ":443",
			}},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:     map[string]*dynamic.TCPRouter{},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services:    map[string]*dynamic.TCPService{},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers: map[string]*dynamic.Router{
						"default-http-app-1-my-gateway-websecure-1c0cf64bde37d9d0df06": {
							EntryPoints: []string{"websecure"},
							Service:     "default-http-app-1-my-gateway-websecure-1c0cf64bde37d9d0df06-wrr",
							Rule:        "Host(`foo.com`) && Path(`/bar`)",
							TLS:         &dynamic.RouterTLSConfig{},
						},
					},
					Middlewares: map[string]*dynamic.Middleware{},
					Services: map[string]*dynamic.Service{
						"default-http-app-1-my-gateway-websecure-1c0cf64bde37d9d0df06-wrr": {
							Weighted: &dynamic.WeightedRoundRobin{
								Services: []dynamic.WRRService{
									{
										Name:   "default-whoami-80",
										Weight: func(i int) *int { return &i }(1),
									},
								},
							},
						},
						"default-whoami-80": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								Servers: []dynamic.Server{
									{
										URL: "http://10.10.0.1:80",
									},
									{
										URL: "http://10.10.0.2:80",
									},
								},
								PassHostHeader: Bool(true),
							},
						},
					},
				},
				TLS: &dynamic.TLSConfiguration{
					Certificates: []*tls.CertAndStores{
						{
							Certificate: tls.Certificate{
								CertFile: tls.FileOrContent("-----BEGIN CERTIFICATE-----\n-----END CERTIFICATE-----"),
								KeyFile:  tls.FileOrContent("-----BEGIN PRIVATE KEY-----\n-----END PRIVATE KEY-----"),
							},
						},
					},
				},
			},
		},
		{
			desc:  "Simple HTTPRoute, with multiple hosts",
			paths: []string{"services.yml", "httproute/with_multiple_host.yml"},
			entryPoints: map[string]Entrypoint{"web": {
				Address: ":80",
			}},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:     map[string]*dynamic.TCPRouter{},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services:    map[string]*dynamic.TCPService{},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers: map[string]*dynamic.Router{
						"default-http-app-1-my-gateway-web-75dd1ad561e42725558a": {
							EntryPoints: []string{"web"},
							Service:     "default-http-app-1-my-gateway-web-75dd1ad561e42725558a-wrr",
							Rule:        "Host(`foo.com`, `bar.com`) && PathPrefix(`/`)",
						},
					},
					Middlewares: map[string]*dynamic.Middleware{},
					Services: map[string]*dynamic.Service{
						"default-http-app-1-my-gateway-web-75dd1ad561e42725558a-wrr": {
							Weighted: &dynamic.WeightedRoundRobin{
								Services: []dynamic.WRRService{
									{
										Name:   "default-whoami-80",
										Weight: func(i int) *int { return &i }(1),
									},
								},
							},
						},
						"default-whoami-80": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								Servers: []dynamic.Server{
									{
										URL: "http://10.10.0.1:80",
									},
									{
										URL: "http://10.10.0.2:80",
									},
								},
								PassHostHeader: Bool(true),
							},
						},
					},
				},
				TLS: &dynamic.TLSConfiguration{},
			},
		},
		{
			desc:  "Simple HTTPRoute, with two hosts one wildcard",
			paths: []string{"services.yml", "with_two_hosts_one_wildcard.yml"},
			entryPoints: map[string]Entrypoint{"web": {
				Address: ":80",
			}},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:     map[string]*dynamic.TCPRouter{},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services:    map[string]*dynamic.TCPService{},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers: map[string]*dynamic.Router{
						"default-http-app-1-my-gateway-web-2dbd7883f5537db39bca": {
							EntryPoints: []string{"web"},
							Service:     "default-http-app-1-my-gateway-web-2dbd7883f5537db39bca-wrr",
							Rule:        "(Host(`foo.com`) || HostRegexp(`{subdomain:[a-zA-Z0-9-]+}.bar.com`)) && PathPrefix(`/`)",
						},
					},
					Middlewares: map[string]*dynamic.Middleware{},
					Services: map[string]*dynamic.Service{
						"default-http-app-1-my-gateway-web-2dbd7883f5537db39bca-wrr": {
							Weighted: &dynamic.WeightedRoundRobin{
								Services: []dynamic.WRRService{
									{
										Name:   "default-whoami-80",
										Weight: func(i int) *int { return &i }(1),
									},
								},
							},
						},
						"default-whoami-80": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								Servers: []dynamic.Server{
									{
										URL: "http://10.10.0.1:80",
									},
									{
										URL: "http://10.10.0.2:80",
									},
								},
								PassHostHeader: Bool(true),
							},
						},
					},
				},
				TLS: &dynamic.TLSConfiguration{},
			},
		},
		{
			desc:  "Simple HTTPRoute, with one host and a wildcard",
			paths: []string{"services.yml", "with_two_hosts_wildcard.yml"},
			entryPoints: map[string]Entrypoint{"web": {
				Address: ":80",
			}},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:     map[string]*dynamic.TCPRouter{},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services:    map[string]*dynamic.TCPService{},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers: map[string]*dynamic.Router{
						"default-http-app-1-my-gateway-web-a431b128267aabc954fd": {
							EntryPoints: []string{"web"},
							Service:     "default-http-app-1-my-gateway-web-a431b128267aabc954fd-wrr",
							Rule:        "PathPrefix(`/`)",
						},
					},
					Middlewares: map[string]*dynamic.Middleware{},
					Services: map[string]*dynamic.Service{
						"default-http-app-1-my-gateway-web-a431b128267aabc954fd-wrr": {
							Weighted: &dynamic.WeightedRoundRobin{
								Services: []dynamic.WRRService{
									{
										Name:   "default-whoami-80",
										Weight: func(i int) *int { return &i }(1),
									},
								},
							},
						},
						"default-whoami-80": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								Servers: []dynamic.Server{
									{
										URL: "http://10.10.0.1:80",
									},
									{
										URL: "http://10.10.0.2:80",
									},
								},
								PassHostHeader: Bool(true),
							},
						},
					},
				},
				TLS: &dynamic.TLSConfiguration{},
			},
		},
		{
			desc:  "One HTTPRoute with two different rules",
			paths: []string{"services.yml", "httproute/two_rules.yml"},
			entryPoints: map[string]Entrypoint{"web": {
				Address: ":80",
			}},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:     map[string]*dynamic.TCPRouter{},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services:    map[string]*dynamic.TCPService{},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers: map[string]*dynamic.Router{
						"default-http-app-1-my-gateway-web-1c0cf64bde37d9d0df06": {
							EntryPoints: []string{"web"},
							Rule:        "Host(`foo.com`) && Path(`/bar`)",
							Service:     "default-http-app-1-my-gateway-web-1c0cf64bde37d9d0df06-wrr",
						},
						"default-http-app-1-my-gateway-web-d737b4933fa88e68ab8a": {
							EntryPoints: []string{"web"},
							Rule:        "Host(`foo.com`) && Path(`/bir`)",
							Service:     "default-http-app-1-my-gateway-web-d737b4933fa88e68ab8a-wrr",
						},
					},
					Middlewares: map[string]*dynamic.Middleware{},
					Services: map[string]*dynamic.Service{
						"default-http-app-1-my-gateway-web-1c0cf64bde37d9d0df06-wrr": {
							Weighted: &dynamic.WeightedRoundRobin{
								Services: []dynamic.WRRService{
									{
										Name:   "default-whoami-80",
										Weight: func(i int) *int { return &i }(1),
									},
								},
							},
						},
						"default-http-app-1-my-gateway-web-d737b4933fa88e68ab8a-wrr": {
							Weighted: &dynamic.WeightedRoundRobin{
								Services: []dynamic.WRRService{
									{
										Name:   "default-whoami2-8080",
										Weight: func(i int) *int { return &i }(1),
									},
								},
							},
						},
						"default-whoami-80": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								Servers: []dynamic.Server{
									{
										URL: "http://10.10.0.1:80",
									},
									{
										URL: "http://10.10.0.2:80",
									},
								},
								PassHostHeader: Bool(true),
							},
						},
						"default-whoami2-8080": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								Servers: []dynamic.Server{
									{
										URL: "http://10.10.0.3:8080",
									},
									{
										URL: "http://10.10.0.4:8080",
									},
								},
								PassHostHeader: Bool(true),
							},
						},
					},
				},
				TLS: &dynamic.TLSConfiguration{},
			},
		},
		{
			desc:  "One HTTPRoute with one rule two targets",
			paths: []string{"services.yml", "httproute/one_rule_two_targets.yml"},
			entryPoints: map[string]Entrypoint{"web": {
				Address: ":80",
			}},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:     map[string]*dynamic.TCPRouter{},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services:    map[string]*dynamic.TCPService{},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers: map[string]*dynamic.Router{
						"default-http-app-1-my-gateway-web-1c0cf64bde37d9d0df06": {
							EntryPoints: []string{"web"},
							Rule:        "Host(`foo.com`) && Path(`/bar`)",
							Service:     "default-http-app-1-my-gateway-web-1c0cf64bde37d9d0df06-wrr",
						},
					},
					Middlewares: map[string]*dynamic.Middleware{},
					Services: map[string]*dynamic.Service{
						"default-http-app-1-my-gateway-web-1c0cf64bde37d9d0df06-wrr": {
							Weighted: &dynamic.WeightedRoundRobin{
								Services: []dynamic.WRRService{
									{
										Name:   "default-whoami-80",
										Weight: func(i int) *int { return &i }(1),
									},
									{
										Name:   "default-whoami2-8080",
										Weight: func(i int) *int { return &i }(1),
									},
								},
							},
						},
						"default-whoami-80": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								Servers: []dynamic.Server{
									{
										URL: "http://10.10.0.1:80",
									},
									{
										URL: "http://10.10.0.2:80",
									},
								},
								PassHostHeader: Bool(true),
							},
						},
						"default-whoami2-8080": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								Servers: []dynamic.Server{
									{
										URL: "http://10.10.0.3:8080",
									},
									{
										URL: "http://10.10.0.4:8080",
									},
								},
								PassHostHeader: Bool(true),
							},
						},
					},
				},
				TLS: &dynamic.TLSConfiguration{},
			},
		},
		{
			desc:  "Two Gateways and one HTTPRoute",
			paths: []string{"services.yml", "httproute/with_two_gateways_one_httproute.yml"},
			entryPoints: map[string]Entrypoint{
				"web": {
					Address: ":80",
				},
				"websecure": {
					Address: ":443",
				},
			},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:     map[string]*dynamic.TCPRouter{},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services:    map[string]*dynamic.TCPService{},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers: map[string]*dynamic.Router{
						"default-http-app-1-my-gateway-http-web-1c0cf64bde37d9d0df06": {
							EntryPoints: []string{"web"},
							Service:     "default-http-app-1-my-gateway-http-web-1c0cf64bde37d9d0df06-wrr",
							Rule:        "Host(`foo.com`) && Path(`/bar`)",
						},
						"default-http-app-1-my-gateway-https-websecure-1c0cf64bde37d9d0df06": {
							EntryPoints: []string{"websecure"},
							Service:     "default-http-app-1-my-gateway-https-websecure-1c0cf64bde37d9d0df06-wrr",
							Rule:        "Host(`foo.com`) && Path(`/bar`)",
							TLS:         &dynamic.RouterTLSConfig{},
						},
					},
					Middlewares: map[string]*dynamic.Middleware{},
					Services: map[string]*dynamic.Service{
						"default-http-app-1-my-gateway-http-web-1c0cf64bde37d9d0df06-wrr": {
							Weighted: &dynamic.WeightedRoundRobin{
								Services: []dynamic.WRRService{
									{
										Name:   "default-whoami-80",
										Weight: func(i int) *int { return &i }(1),
									},
								},
							},
						},
						"default-http-app-1-my-gateway-https-websecure-1c0cf64bde37d9d0df06-wrr": {
							Weighted: &dynamic.WeightedRoundRobin{
								Services: []dynamic.WRRService{
									{
										Name:   "default-whoami-80",
										Weight: func(i int) *int { return &i }(1),
									},
								},
							},
						},
						"default-whoami-80": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								Servers: []dynamic.Server{
									{
										URL: "http://10.10.0.1:80",
									},
									{
										URL: "http://10.10.0.2:80",
									},
								},
								PassHostHeader: Bool(true),
							},
						},
					},
				},
				TLS: &dynamic.TLSConfiguration{
					Certificates: []*tls.CertAndStores{
						{
							Certificate: tls.Certificate{
								CertFile: tls.FileOrContent("-----BEGIN CERTIFICATE-----\n-----END CERTIFICATE-----"),
								KeyFile:  tls.FileOrContent("-----BEGIN PRIVATE KEY-----\n-----END PRIVATE KEY-----"),
							},
						},
					},
				},
			},
		},
		{
			desc:  "Gateway with two listeners and one HTTPRoute",
			paths: []string{"services.yml", "httproute/with_two_listeners_one_httproute.yml"},
			entryPoints: map[string]Entrypoint{
				"web": {
					Address: ":80",
				},
				"websecure": {
					Address: ":443",
				},
			},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:     map[string]*dynamic.TCPRouter{},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services:    map[string]*dynamic.TCPService{},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers: map[string]*dynamic.Router{
						"default-http-app-1-my-gateway-web-1c0cf64bde37d9d0df06": {
							EntryPoints: []string{"web"},
							Service:     "default-http-app-1-my-gateway-web-1c0cf64bde37d9d0df06-wrr",
							Rule:        "Host(`foo.com`) && Path(`/bar`)",
						},
						"default-http-app-1-my-gateway-websecure-1c0cf64bde37d9d0df06": {
							EntryPoints: []string{"websecure"},
							Service:     "default-http-app-1-my-gateway-websecure-1c0cf64bde37d9d0df06-wrr",
							Rule:        "Host(`foo.com`) && Path(`/bar`)",
							TLS:         &dynamic.RouterTLSConfig{},
						},
					},
					Middlewares: map[string]*dynamic.Middleware{},
					Services: map[string]*dynamic.Service{
						"default-http-app-1-my-gateway-web-1c0cf64bde37d9d0df06-wrr": {
							Weighted: &dynamic.WeightedRoundRobin{
								Services: []dynamic.WRRService{
									{
										Name:   "default-whoami-80",
										Weight: func(i int) *int { return &i }(1),
									},
								},
							},
						},
						"default-http-app-1-my-gateway-websecure-1c0cf64bde37d9d0df06-wrr": {
							Weighted: &dynamic.WeightedRoundRobin{
								Services: []dynamic.WRRService{
									{
										Name:   "default-whoami-80",
										Weight: func(i int) *int { return &i }(1),
									},
								},
							},
						},
						"default-whoami-80": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								Servers: []dynamic.Server{
									{
										URL: "http://10.10.0.1:80",
									},
									{
										URL: "http://10.10.0.2:80",
									},
								},
								PassHostHeader: Bool(true),
							},
						},
					},
				},
				TLS: &dynamic.TLSConfiguration{
					Certificates: []*tls.CertAndStores{
						{
							Certificate: tls.Certificate{
								CertFile: tls.FileOrContent("-----BEGIN CERTIFICATE-----\n-----END CERTIFICATE-----"),
								KeyFile:  tls.FileOrContent("-----BEGIN PRIVATE KEY-----\n-----END PRIVATE KEY-----"),
							},
						},
					},
				},
			},
		},
		{
			desc:  "Simple HTTPRoute, with several rules",
			paths: []string{"services.yml", "httproute/with_several_rules.yml"},
			entryPoints: map[string]Entrypoint{"web": {
				Address: ":80",
			}},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:     map[string]*dynamic.TCPRouter{},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services:    map[string]*dynamic.TCPService{},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers: map[string]*dynamic.Router{
						"default-http-app-1-my-gateway-web-6211a6376ce8f78494a8": {
							EntryPoints: []string{"web"},
							Service:     "default-http-app-1-my-gateway-web-6211a6376ce8f78494a8-wrr",
							Rule:        "Host(`foo.com`) && PathPrefix(`/bar`) && Headers(`my-header2`,`bar`) && Headers(`my-header`,`foo`)",
						},
						"default-http-app-1-my-gateway-web-fe80e69a38713941ea22": {
							EntryPoints: []string{"web"},
							Service:     "default-http-app-1-my-gateway-web-fe80e69a38713941ea22-wrr",
							Rule:        "Host(`foo.com`) && Path(`/bar`) && Headers(`my-header`,`bar`)",
						},
					},
					Middlewares: map[string]*dynamic.Middleware{},
					Services: map[string]*dynamic.Service{
						"default-http-app-1-my-gateway-web-6211a6376ce8f78494a8-wrr": {
							Weighted: &dynamic.WeightedRoundRobin{
								Services: []dynamic.WRRService{
									{
										Name:   "default-whoami-80",
										Weight: func(i int) *int { return &i }(1),
									},
								},
							},
						},
						"default-http-app-1-my-gateway-web-fe80e69a38713941ea22-wrr": {
							Weighted: &dynamic.WeightedRoundRobin{
								Services: []dynamic.WRRService{
									{
										Name:   "default-whoami-80",
										Weight: func(i int) *int { return &i }(1),
									},
								},
							},
						},
						"default-whoami-80": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								Servers: []dynamic.Server{
									{
										URL: "http://10.10.0.1:80",
									},
									{
										URL: "http://10.10.0.2:80",
									},
								},
								PassHostHeader: Bool(true),
							},
						},
					},
				},
				TLS: &dynamic.TLSConfiguration{},
			},
		},
	}

	for _, test := range testCases {
		test := test

		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			if test.expected == nil {
				return
			}

			p := Provider{EntryPoints: test.entryPoints}
			conf := p.loadConfigurationFromGateway(context.Background(), newClientMock(test.paths...))
			assert.Equal(t, test.expected, conf)
		})
	}
}

func TestLoadTCPRoutes(t *testing.T) {
	testCases := []struct {
		desc         string
		ingressClass string
		paths        []string
		expected     *dynamic.Configuration
		entryPoints  map[string]Entrypoint
	}{
		{
			desc: "Empty",
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:     map[string]*dynamic.TCPRouter{},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services:    map[string]*dynamic.TCPService{},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers:     map[string]*dynamic.Router{},
					Middlewares: map[string]*dynamic.Middleware{},
					Services:    map[string]*dynamic.Service{},
				},
				TLS: &dynamic.TLSConfiguration{},
			},
		},
		{
			desc:  "Empty because missing entry point",
			paths: []string{"services.yml", "tcproute/simple.yml"},
			entryPoints: map[string]Entrypoint{"TCP": {
				Address: ":8000",
			}},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:     map[string]*dynamic.TCPRouter{},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services:    map[string]*dynamic.TCPService{},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers:     map[string]*dynamic.Router{},
					Middlewares: map[string]*dynamic.Middleware{},
					Services:    map[string]*dynamic.Service{},
				},
				TLS: &dynamic.TLSConfiguration{},
			},
		},
		{
			desc:  "Empty because no tcp route defined",
			paths: []string{"services.yml", "tcproute/without_tcproute.yml"},
			entryPoints: map[string]Entrypoint{"TCP": {
				Address: ":8080",
			}},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:     map[string]*dynamic.TCPRouter{},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services:    map[string]*dynamic.TCPService{},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers:     map[string]*dynamic.Router{},
					Middlewares: map[string]*dynamic.Middleware{},
					Services:    map[string]*dynamic.Service{},
				},
				TLS: &dynamic.TLSConfiguration{},
			},
		},
		{
			desc: "Empty caused by missing GatewayClass",
			entryPoints: map[string]Entrypoint{"TCP": {
				Address: ":8080",
			}},
			paths: []string{"services.yml", "tcproute/without_gatewayclass.yml"},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:     map[string]*dynamic.TCPRouter{},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services:    map[string]*dynamic.TCPService{},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers:     map[string]*dynamic.Router{},
					Middlewares: map[string]*dynamic.Middleware{},
					Services:    map[string]*dynamic.Service{},
				},
				TLS: &dynamic.TLSConfiguration{},
			},
		},
		{
			desc: "Empty caused by unknown GatewayClass controller name",
			entryPoints: map[string]Entrypoint{"TCP": {
				Address: ":8080",
			}},
			paths: []string{"services.yml", "tcproute/gatewayclass_with_unknown_controller.yml"},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:     map[string]*dynamic.TCPRouter{},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services:    map[string]*dynamic.TCPService{},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers:     map[string]*dynamic.Router{},
					Middlewares: map[string]*dynamic.Middleware{},
					Services:    map[string]*dynamic.Service{},
				},
				TLS: &dynamic.TLSConfiguration{},
			},
		},
		{
			desc: "Empty caused by multiport service with wrong TargetPort",
			entryPoints: map[string]Entrypoint{"TCP": {
				Address: ":8080",
			}},
			paths: []string{"services.yml", "tcproute/with_wrong_service_port.yml"},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:     map[string]*dynamic.TCPRouter{},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services:    map[string]*dynamic.TCPService{},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers:     map[string]*dynamic.Router{},
					Middlewares: map[string]*dynamic.Middleware{},
					Services:    map[string]*dynamic.Service{},
				},
				TLS: &dynamic.TLSConfiguration{},
			},
		},
		{
			desc:  "Simple TCPRoute, with foo entrypoint",
			paths: []string{"services.yml", "tcproute/simple.yml"},
			entryPoints: map[string]Entrypoint{
				"tcp": {Address: ":9000"},
			},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers: map[string]*dynamic.TCPRouter{
						"default-tcp-app-1-my-tcp-gateway-tcp-e3b0c44298fc1c149afb": {
							EntryPoints: []string{"tcp"},
							Service:     "default-tcp-app-1-my-tcp-gateway-tcp-e3b0c44298fc1c149afb-wrr",
							Rule:        "HostSNI(`*`)",
						},
					},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services: map[string]*dynamic.TCPService{
						"default-tcp-app-1-my-tcp-gateway-tcp-e3b0c44298fc1c149afb-wrr": {
							Weighted: &dynamic.TCPWeightedRoundRobin{
								Services: []dynamic.TCPWRRService{
									{
										Name:   "default-whoamitcp-9000",
										Weight: func(i int) *int { return &i }(1),
									},
								},
							},
						},
						"default-whoamitcp-9000": {
							LoadBalancer: &dynamic.TCPServersLoadBalancer{
								Servers: []dynamic.TCPServer{
									{
										Address: "10.10.0.9:9000",
									},
									{
										Address: "10.10.0.10:9000",
									},
								},
							},
						},
					},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers:     map[string]*dynamic.Router{},
					Middlewares: map[string]*dynamic.Middleware{},
					Services:    map[string]*dynamic.Service{},
				},
				TLS: &dynamic.TLSConfiguration{},
			},
		},
		{
			desc:  "Multiple TCPRoute, with foo entrypoint",
			paths: []string{"services.yml", "tcproute/with_multiple_routes.yml"},
			entryPoints: map[string]Entrypoint{
				"tcp-1":   {Address: ":9000"},
				"tcp-2":   {Address: ":10000"},
				"not-tcp": {Address: ":11000"},
			},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers: map[string]*dynamic.TCPRouter{
						"default-tcp-app-1-my-tcp-gateway-tcp-1-e3b0c44298fc1c149afb": {
							EntryPoints: []string{"tcp-1"},
							Service:     "default-tcp-app-1-my-tcp-gateway-tcp-1-e3b0c44298fc1c149afb-wrr",
							Rule:        "HostSNI(`*`)",
						},
						"default-tcp-app-2-my-tcp-gateway-tcp-2-e3b0c44298fc1c149afb": {
							EntryPoints: []string{"tcp-2"},
							Service:     "default-tcp-app-2-my-tcp-gateway-tcp-2-e3b0c44298fc1c149afb-wrr",
							Rule:        "HostSNI(`*`)",
						},
					},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services: map[string]*dynamic.TCPService{
						"default-tcp-app-1-my-tcp-gateway-tcp-1-e3b0c44298fc1c149afb-wrr": {
							Weighted: &dynamic.TCPWeightedRoundRobin{
								Services: []dynamic.TCPWRRService{
									{
										Name:   "default-whoamitcp-9000",
										Weight: func(i int) *int { return &i }(1),
									},
								},
							},
						},
						"default-tcp-app-2-my-tcp-gateway-tcp-2-e3b0c44298fc1c149afb-wrr": {
							Weighted: &dynamic.TCPWeightedRoundRobin{
								Services: []dynamic.TCPWRRService{
									{
										Name:   "default-whoamitcp-10000",
										Weight: func(i int) *int { return &i }(1),
									},
								},
							},
						},
						"default-whoamitcp-9000": {
							LoadBalancer: &dynamic.TCPServersLoadBalancer{
								Servers: []dynamic.TCPServer{
									{
										Address: "10.10.0.9:9000",
									},
									{
										Address: "10.10.0.10:9000",
									},
								},
							},
						},
						"default-whoamitcp-10000": {
							LoadBalancer: &dynamic.TCPServersLoadBalancer{
								Servers: []dynamic.TCPServer{
									{
										Address: "10.10.0.9:10000",
									},
									{
										Address: "10.10.0.10:10000",
									},
								},
							},
						},
					},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers:     map[string]*dynamic.Router{},
					Middlewares: map[string]*dynamic.Middleware{},
					Services:    map[string]*dynamic.Service{},
				},
				TLS: &dynamic.TLSConfiguration{},
			},
		},
		{
			desc:  "Empty caused by TCPRoute with multiple rules",
			paths: []string{"services.yml", "tcproute/with_multiple_rules.yml"},
			entryPoints: map[string]Entrypoint{
				"tcp-1":   {Address: ":9000"},
				"tcp-2":   {Address: ":10000"},
				"not-tcp": {Address: ":11000"},
			},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:     map[string]*dynamic.TCPRouter{},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services:    map[string]*dynamic.TCPService{},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers:     map[string]*dynamic.Router{},
					Middlewares: map[string]*dynamic.Middleware{},
					Services:    map[string]*dynamic.Service{},
				},
				TLS: &dynamic.TLSConfiguration{},
			},
		},
		{
			desc:  "Simple TCPRoute, with backendRef",
			paths: []string{"services.yml", "tcproute/simple_cross_provider.yml"},
			entryPoints: map[string]Entrypoint{
				"tcp": {Address: ":9000"},
			},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers: map[string]*dynamic.TCPRouter{
						"default-tcp-app-1-my-gateway-tcp-e3b0c44298fc1c149afb": {
							EntryPoints: []string{"tcp"},
							Service:     "default-tcp-app-1-my-gateway-tcp-e3b0c44298fc1c149afb-wrr",
							Rule:        "HostSNI(`*`)",
						},
					},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services: map[string]*dynamic.TCPService{
						"default-tcp-app-1-my-gateway-tcp-e3b0c44298fc1c149afb-wrr": {
							Weighted: &dynamic.TCPWeightedRoundRobin{
								Services: []dynamic.TCPWRRService{
									{
										Name:   "service@file",
										Weight: func(i int) *int { return &i }(1),
									},
									{
										Name:   "default-whoamitcp-9000",
										Weight: func(i int) *int { return &i }(1),
									},
								},
							},
						},
						"default-whoamitcp-9000": {
							LoadBalancer: &dynamic.TCPServersLoadBalancer{
								Servers: []dynamic.TCPServer{
									{
										Address: "10.10.0.9:9000",
									},
									{
										Address: "10.10.0.10:9000",
									},
								},
							},
						},
					},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers:     map[string]*dynamic.Router{},
					Middlewares: map[string]*dynamic.Middleware{},
					Services:    map[string]*dynamic.Service{},
				},
				TLS: &dynamic.TLSConfiguration{},
			},
		},
		{
			desc:  "Simple TCPRoute, with TLS",
			paths: []string{"services.yml", "tcproute/with_protocol_tls.yml"},
			entryPoints: map[string]Entrypoint{
				"tls": {Address: ":9000"},
			},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers: map[string]*dynamic.TCPRouter{
						"default-tcp-app-1-my-gateway-tls-e3b0c44298fc1c149afb": {
							EntryPoints: []string{"tls"},
							Service:     "default-tcp-app-1-my-gateway-tls-e3b0c44298fc1c149afb-wrr",
							Rule:        "HostSNI(`*`)",
							TLS:         &dynamic.RouterTCPTLSConfig{},
						},
					},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services: map[string]*dynamic.TCPService{
						"default-tcp-app-1-my-gateway-tls-e3b0c44298fc1c149afb-wrr": {
							Weighted: &dynamic.TCPWeightedRoundRobin{
								Services: []dynamic.TCPWRRService{{
									Name:   "default-whoamitcp-9000",
									Weight: func(i int) *int { return &i }(1),
								}},
							},
						},
						"default-whoamitcp-9000": {
							LoadBalancer: &dynamic.TCPServersLoadBalancer{
								Servers: []dynamic.TCPServer{
									{
										Address: "10.10.0.9:9000",
									},
									{
										Address: "10.10.0.10:9000",
									},
								},
							},
						},
					},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers:     map[string]*dynamic.Router{},
					Middlewares: map[string]*dynamic.Middleware{},
					Services:    map[string]*dynamic.Service{},
				},
				TLS: &dynamic.TLSConfiguration{
					Certificates: []*tls.CertAndStores{
						{
							Certificate: tls.Certificate{
								CertFile: tls.FileOrContent("-----BEGIN CERTIFICATE-----\n-----END CERTIFICATE-----"),
								KeyFile:  tls.FileOrContent("-----BEGIN PRIVATE KEY-----\n-----END PRIVATE KEY-----"),
							},
						},
					},
				},
			},
		},
	}

	for _, test := range testCases {
		test := test

		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			if test.expected == nil {
				return
			}

			p := Provider{EntryPoints: test.entryPoints}
			conf := p.loadConfigurationFromGateway(context.Background(), newClientMock(test.paths...))
			assert.Equal(t, test.expected, conf)
		})
	}
}

func TestLoadTLSRoutes(t *testing.T) {
	testCases := []struct {
		desc         string
		ingressClass string
		paths        []string
		expected     *dynamic.Configuration
		entryPoints  map[string]Entrypoint
	}{
		{
			desc: "Empty",
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:     map[string]*dynamic.TCPRouter{},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services:    map[string]*dynamic.TCPService{},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers:     map[string]*dynamic.Router{},
					Middlewares: map[string]*dynamic.Middleware{},
					Services:    map[string]*dynamic.Service{},
				},
				TLS: &dynamic.TLSConfiguration{},
			},
		},
		{
			desc:  "Empty because no matching entry point",
			paths: []string{"services.yml", "tlsroute/simple_TLS_to_TCPRoute.yml"},
			entryPoints: map[string]Entrypoint{"TCP": {
				Address: ":8000",
			}},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:     map[string]*dynamic.TCPRouter{},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services:    map[string]*dynamic.TCPService{},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers:     map[string]*dynamic.Router{},
					Middlewares: map[string]*dynamic.Middleware{},
					Services:    map[string]*dynamic.Service{},
				},
				TLS: &dynamic.TLSConfiguration{},
			},
		},
		{
			desc:  "Empty because no tls route defined",
			paths: []string{"services.yml", "tlsroute/without_tlsroute.yml"},
			entryPoints: map[string]Entrypoint{"TCP": {
				Address: ":8080",
			}},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:     map[string]*dynamic.TCPRouter{},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services:    map[string]*dynamic.TCPService{},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers:     map[string]*dynamic.Router{},
					Middlewares: map[string]*dynamic.Middleware{},
					Services:    map[string]*dynamic.Service{},
				},
				TLS: &dynamic.TLSConfiguration{},
			},
		},
		{
			desc: "Empty caused by missing GatewayClass",
			entryPoints: map[string]Entrypoint{"TCP": {
				Address: ":8080",
			}},
			paths: []string{"services.yml", "tlsroute/without_gatewayclass.yml"},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:     map[string]*dynamic.TCPRouter{},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services:    map[string]*dynamic.TCPService{},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers:     map[string]*dynamic.Router{},
					Middlewares: map[string]*dynamic.Middleware{},
					Services:    map[string]*dynamic.Service{},
				},
				TLS: &dynamic.TLSConfiguration{},
			},
		},
		{
			desc: "Empty caused by unknown GatewayClass controller name",
			entryPoints: map[string]Entrypoint{"TCP": {
				Address: ":8080",
			}},
			paths: []string{"services.yml", "tlsroute/gatewayclass_with_unknown_controller.yml"},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:     map[string]*dynamic.TCPRouter{},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services:    map[string]*dynamic.TCPService{},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers:     map[string]*dynamic.Router{},
					Middlewares: map[string]*dynamic.Middleware{},
					Services:    map[string]*dynamic.Service{},
				},
				TLS: &dynamic.TLSConfiguration{},
			},
		},
		{
			desc: "Empty caused by multiport service with wrong TargetPort",
			entryPoints: map[string]Entrypoint{"TCP": {
				Address: ":8080",
			}},
			paths: []string{"services.yml", "tlsroute/with_wrong_service_port.yml"},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:     map[string]*dynamic.TCPRouter{},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services:    map[string]*dynamic.TCPService{},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers:     map[string]*dynamic.Router{},
					Middlewares: map[string]*dynamic.Middleware{},
					Services:    map[string]*dynamic.Service{},
				},
				TLS: &dynamic.TLSConfiguration{},
			},
		},
		{
			desc: "Empty caused by mixed routes wrong selector",
			entryPoints: map[string]Entrypoint{
				"tcp": {
					Address: ":9000",
				},
				"tcp-tls": {
					Address: ":9443",
				},
				"http": {
					Address: ":80",
				},
			},
			paths: []string{"services.yml", "mixed/with_wrong_routes_selector.yml"},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:     map[string]*dynamic.TCPRouter{},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services:    map[string]*dynamic.TCPService{},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers:     map[string]*dynamic.Router{},
					Middlewares: map[string]*dynamic.Middleware{},
					Services:    map[string]*dynamic.Service{},
				},
				TLS: &dynamic.TLSConfiguration{},
			},
		},
		{
			desc: "Empty caused by TLSRoute using a certificateRef",
			entryPoints: map[string]Entrypoint{"TCP": {
				Address: ":9000",
			}},
			paths: []string{"services.yml", "tlsroute/simple_TLS_to_TCPRoute_with_certificateRef.yml"},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:     map[string]*dynamic.TCPRouter{},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services:    map[string]*dynamic.TCPService{},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers:     map[string]*dynamic.Router{},
					Middlewares: map[string]*dynamic.Middleware{},
					Services:    map[string]*dynamic.Service{},
				},
				TLS: &dynamic.TLSConfiguration{},
			},
		},
		{
			desc:  "Empty caused by simple TLSRoute with invalid SNI matching",
			paths: []string{"services.yml", "tlsroute/with_invalid_SNI_matching.yml"},
			entryPoints: map[string]Entrypoint{
				"tls": {Address: ":9001"},
			},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:     map[string]*dynamic.TCPRouter{},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services:    map[string]*dynamic.TCPService{},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers:     map[string]*dynamic.Router{},
					Middlewares: map[string]*dynamic.Middleware{},
					Services:    map[string]*dynamic.Service{},
				},
				TLS: &dynamic.TLSConfiguration{},
			},
		},
		{
			desc:  "Simple TLS listener to TCPRoute, with foo entrypoint",
			paths: []string{"services.yml", "tlsroute/simple_TLS_to_TCPRoute.yml"},
			entryPoints: map[string]Entrypoint{
				"tcp": {Address: ":9000"},
			},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers: map[string]*dynamic.TCPRouter{
						"default-tcp-app-1-my-tls-gateway-tcp-e3b0c44298fc1c149afb": {
							EntryPoints: []string{"tcp"},
							Service:     "default-tcp-app-1-my-tls-gateway-tcp-e3b0c44298fc1c149afb-wrr",
							Rule:        "HostSNI(`*`)",
							TLS:         &dynamic.RouterTCPTLSConfig{},
						},
					},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services: map[string]*dynamic.TCPService{
						"default-tcp-app-1-my-tls-gateway-tcp-e3b0c44298fc1c149afb-wrr": {
							Weighted: &dynamic.TCPWeightedRoundRobin{
								Services: []dynamic.TCPWRRService{
									{
										Name:   "default-whoamitcp-9000",
										Weight: func(i int) *int { return &i }(1),
									},
								},
							},
						},
						"default-whoamitcp-9000": {
							LoadBalancer: &dynamic.TCPServersLoadBalancer{
								Servers: []dynamic.TCPServer{
									{
										Address: "10.10.0.9:9000",
									},
									{
										Address: "10.10.0.10:9000",
									},
								},
							},
						},
					},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers:     map[string]*dynamic.Router{},
					Middlewares: map[string]*dynamic.Middleware{},
					Services:    map[string]*dynamic.Service{},
				},
				TLS: &dynamic.TLSConfiguration{
					Certificates: []*tls.CertAndStores{
						{
							Certificate: tls.Certificate{
								CertFile: tls.FileOrContent("-----BEGIN CERTIFICATE-----\n-----END CERTIFICATE-----"),
								KeyFile:  tls.FileOrContent("-----BEGIN PRIVATE KEY-----\n-----END PRIVATE KEY-----"),
							},
						},
					},
				},
			},
		},
		{
			desc:  "Simple TLS listener to TLSRoute",
			paths: []string{"services.yml", "tlsroute/simple_TLS_to_TLSRoute.yml"},
			entryPoints: map[string]Entrypoint{
				"tcp": {Address: ":9000"},
			},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers: map[string]*dynamic.TCPRouter{
						"default-tls-app-1-my-tls-gateway-tcp-673acf455cb2dab0b43a": {
							EntryPoints: []string{"tcp"},
							Service:     "default-tls-app-1-my-tls-gateway-tcp-673acf455cb2dab0b43a-wrr",
							Rule:        "HostSNI(`*`)",
							TLS: &dynamic.RouterTCPTLSConfig{
								Passthrough: true,
							},
						},
					},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services: map[string]*dynamic.TCPService{
						"default-tls-app-1-my-tls-gateway-tcp-673acf455cb2dab0b43a-wrr": {
							Weighted: &dynamic.TCPWeightedRoundRobin{
								Services: []dynamic.TCPWRRService{
									{
										Name:   "default-whoamitcp-9000",
										Weight: func(i int) *int { return &i }(1),
									},
								},
							},
						},
						"default-whoamitcp-9000": {
							LoadBalancer: &dynamic.TCPServersLoadBalancer{
								Servers: []dynamic.TCPServer{
									{
										Address: "10.10.0.9:9000",
									},
									{
										Address: "10.10.0.10:9000",
									},
								},
							},
						},
					},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers:     map[string]*dynamic.Router{},
					Middlewares: map[string]*dynamic.Middleware{},
					Services:    map[string]*dynamic.Service{},
				},
				TLS: &dynamic.TLSConfiguration{},
			},
		},
		{
			desc:  "Multiple TLSRoute, with foo entrypoint",
			paths: []string{"services.yml", "tlsroute/with_multiple_routes_kind.yml"},
			entryPoints: map[string]Entrypoint{
				"tls": {Address: ":9000"},
				"tcp": {Address: ":10000"},
			},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers: map[string]*dynamic.TCPRouter{
						"default-tcp-app-1-my-tls-gateway-tls-e3b0c44298fc1c149afb": {
							EntryPoints: []string{"tls"},
							Service:     "default-tcp-app-1-my-tls-gateway-tls-e3b0c44298fc1c149afb-wrr",
							Rule:        "HostSNI(`*`)",
							TLS:         &dynamic.RouterTCPTLSConfig{},
						},
						"default-tls-app-1-my-tls-gateway-tcp-673acf455cb2dab0b43a": {
							EntryPoints: []string{"tcp"},
							Service:     "default-tls-app-1-my-tls-gateway-tcp-673acf455cb2dab0b43a-wrr",
							Rule:        "HostSNI(`*`)",
							TLS: &dynamic.RouterTCPTLSConfig{
								Passthrough: true,
							},
						},
					},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services: map[string]*dynamic.TCPService{
						"default-tcp-app-1-my-tls-gateway-tls-e3b0c44298fc1c149afb-wrr": {
							Weighted: &dynamic.TCPWeightedRoundRobin{
								Services: []dynamic.TCPWRRService{
									{
										Name:   "default-whoamitcp-9000",
										Weight: func(i int) *int { return &i }(1),
									},
								},
							},
						},
						"default-tls-app-1-my-tls-gateway-tcp-673acf455cb2dab0b43a-wrr": {
							Weighted: &dynamic.TCPWeightedRoundRobin{
								Services: []dynamic.TCPWRRService{
									{
										Name:   "default-whoamitcp-10000",
										Weight: func(i int) *int { return &i }(1),
									},
								},
							},
						},
						"default-whoamitcp-9000": {
							LoadBalancer: &dynamic.TCPServersLoadBalancer{
								Servers: []dynamic.TCPServer{
									{
										Address: "10.10.0.9:9000",
									},
									{
										Address: "10.10.0.10:9000",
									},
								},
							},
						},
						"default-whoamitcp-10000": {
							LoadBalancer: &dynamic.TCPServersLoadBalancer{
								Servers: []dynamic.TCPServer{
									{
										Address: "10.10.0.9:10000",
									},
									{
										Address: "10.10.0.10:10000",
									},
								},
							},
						},
					},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers:     map[string]*dynamic.Router{},
					Middlewares: map[string]*dynamic.Middleware{},
					Services:    map[string]*dynamic.Service{},
				},
				TLS: &dynamic.TLSConfiguration{
					Certificates: []*tls.CertAndStores{
						{
							Certificate: tls.Certificate{
								CertFile: tls.FileOrContent("-----BEGIN CERTIFICATE-----\n-----END CERTIFICATE-----"),
								KeyFile:  tls.FileOrContent("-----BEGIN PRIVATE KEY-----\n-----END PRIVATE KEY-----"),
							},
						},
					},
				},
			},
		},
		{
			desc:  "Simple TLSRoute, with backendRef",
			paths: []string{"services.yml", "tlsroute/simple_cross_provider.yml"},
			entryPoints: map[string]Entrypoint{
				"tls": {Address: ":9000"},
			},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers: map[string]*dynamic.TCPRouter{
						"default-tcp-app-1-my-gateway-tls-e3b0c44298fc1c149afb": {
							EntryPoints: []string{"tls"},
							Service:     "default-tcp-app-1-my-gateway-tls-e3b0c44298fc1c149afb-wrr",
							Rule:        "HostSNI(`*`)",
							TLS:         &dynamic.RouterTCPTLSConfig{},
						},
					},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services: map[string]*dynamic.TCPService{
						"default-tcp-app-1-my-gateway-tls-e3b0c44298fc1c149afb-wrr": {
							Weighted: &dynamic.TCPWeightedRoundRobin{
								Services: []dynamic.TCPWRRService{
									{
										Name:   "service@file",
										Weight: func(i int) *int { return &i }(1),
									},
									{
										Name:   "default-whoamitcp-9000",
										Weight: func(i int) *int { return &i }(1),
									},
								},
							},
						},
						"default-whoamitcp-9000": {
							LoadBalancer: &dynamic.TCPServersLoadBalancer{
								Servers: []dynamic.TCPServer{
									{
										Address: "10.10.0.9:9000",
									},
									{
										Address: "10.10.0.10:9000",
									},
								},
							},
						},
					},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers:     map[string]*dynamic.Router{},
					Middlewares: map[string]*dynamic.Middleware{},
					Services:    map[string]*dynamic.Service{},
				},
				TLS: &dynamic.TLSConfiguration{
					Certificates: []*tls.CertAndStores{
						{
							Certificate: tls.Certificate{
								CertFile: tls.FileOrContent("-----BEGIN CERTIFICATE-----\n-----END CERTIFICATE-----"),
								KeyFile:  tls.FileOrContent("-----BEGIN PRIVATE KEY-----\n-----END PRIVATE KEY-----"),
							},
						},
					},
				},
			},
		},
		{
			desc:  "Simple TLSRoute, with Passthrough and TLS configuration should raise a warn",
			paths: []string{"services.yml", "tlsroute/with_passthrough_tls.yml"},
			entryPoints: map[string]Entrypoint{
				"tls": {Address: ":9001"},
			},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers: map[string]*dynamic.TCPRouter{
						"default-tls-app-1-my-gateway-tls-673acf455cb2dab0b43a": {
							EntryPoints: []string{"tls"},
							Service:     "default-tls-app-1-my-gateway-tls-673acf455cb2dab0b43a-wrr",
							Rule:        "HostSNI(`*`)",
							TLS: &dynamic.RouterTCPTLSConfig{
								Passthrough: true,
							},
						},
					},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services: map[string]*dynamic.TCPService{
						"default-tls-app-1-my-gateway-tls-673acf455cb2dab0b43a-wrr": {
							Weighted: &dynamic.TCPWeightedRoundRobin{
								Services: []dynamic.TCPWRRService{
									{
										Name:   "default-whoamitcp-9000",
										Weight: func(i int) *int { return &i }(1),
									},
								},
							},
						},
						"default-whoamitcp-9000": {
							LoadBalancer: &dynamic.TCPServersLoadBalancer{
								Servers: []dynamic.TCPServer{
									{
										Address: "10.10.0.9:9000",
									},
									{
										Address: "10.10.0.10:9000",
									},
								},
							},
						},
					},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers:     map[string]*dynamic.Router{},
					Middlewares: map[string]*dynamic.Middleware{},
					Services:    map[string]*dynamic.Service{},
				},
				TLS: &dynamic.TLSConfiguration{},
			},
		},
		{
			desc:  "Simple TLSRoute, with Passthrough",
			paths: []string{"services.yml", "tlsroute/with_passthrough.yml"},
			entryPoints: map[string]Entrypoint{
				"tls": {Address: ":9001"},
			},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers: map[string]*dynamic.TCPRouter{
						"default-tls-app-1-my-gateway-tls-673acf455cb2dab0b43a": {
							EntryPoints: []string{"tls"},
							Service:     "default-tls-app-1-my-gateway-tls-673acf455cb2dab0b43a-wrr",
							Rule:        "HostSNI(`*`)",
							TLS: &dynamic.RouterTCPTLSConfig{
								Passthrough: true,
							},
						},
					},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services: map[string]*dynamic.TCPService{
						"default-tls-app-1-my-gateway-tls-673acf455cb2dab0b43a-wrr": {
							Weighted: &dynamic.TCPWeightedRoundRobin{
								Services: []dynamic.TCPWRRService{
									{
										Name:   "default-whoamitcp-9000",
										Weight: func(i int) *int { return &i }(1),
									},
								},
							},
						},
						"default-whoamitcp-9000": {
							LoadBalancer: &dynamic.TCPServersLoadBalancer{
								Servers: []dynamic.TCPServer{
									{
										Address: "10.10.0.9:9000",
									},
									{
										Address: "10.10.0.10:9000",
									},
								},
							},
						},
					},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers:     map[string]*dynamic.Router{},
					Middlewares: map[string]*dynamic.Middleware{},
					Services:    map[string]*dynamic.Service{},
				},
				TLS: &dynamic.TLSConfiguration{},
			},
		},
		{
			desc:  "Simple TLSRoute, with single SNI matching",
			paths: []string{"services.yml", "tlsroute/with_SNI_matching.yml"},
			entryPoints: map[string]Entrypoint{
				"tls": {Address: ":9001"},
			},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers: map[string]*dynamic.TCPRouter{
						"default-tls-app-1-my-gateway-tls-2279fe75c5156dc5eb26": {
							EntryPoints: []string{"tls"},
							Service:     "default-tls-app-1-my-gateway-tls-2279fe75c5156dc5eb26-wrr",
							Rule:        "HostSNI(`foo.bar`)",
							TLS: &dynamic.RouterTCPTLSConfig{
								Passthrough: true,
							},
						},
					},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services: map[string]*dynamic.TCPService{
						"default-tls-app-1-my-gateway-tls-2279fe75c5156dc5eb26-wrr": {
							Weighted: &dynamic.TCPWeightedRoundRobin{
								Services: []dynamic.TCPWRRService{
									{
										Name:   "default-whoamitcp-9000",
										Weight: func(i int) *int { return &i }(1),
									},
								},
							},
						},
						"default-whoamitcp-9000": {
							LoadBalancer: &dynamic.TCPServersLoadBalancer{
								Servers: []dynamic.TCPServer{
									{
										Address: "10.10.0.9:9000",
									},
									{
										Address: "10.10.0.10:9000",
									},
								},
							},
						},
					},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers:     map[string]*dynamic.Router{},
					Middlewares: map[string]*dynamic.Middleware{},
					Services:    map[string]*dynamic.Service{},
				},
				TLS: &dynamic.TLSConfiguration{},
			},
		},
		{
			desc:  "Simple TLSRoute, with multiple SNI matching",
			paths: []string{"services.yml", "tlsroute/with_multiple_SNI_matching.yml"},
			entryPoints: map[string]Entrypoint{
				"tls": {Address: ":9001"},
			},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers: map[string]*dynamic.TCPRouter{
						"default-tls-app-1-my-gateway-tls-177bd313b8e78ce821eb": {
							EntryPoints: []string{"tls"},
							Service:     "default-tls-app-1-my-gateway-tls-177bd313b8e78ce821eb-wrr",
							Rule:        "HostSNI(`foo.bar`,`fiz.baz`)",
							TLS: &dynamic.RouterTCPTLSConfig{
								Passthrough: true,
							},
						},
					},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services: map[string]*dynamic.TCPService{
						"default-tls-app-1-my-gateway-tls-177bd313b8e78ce821eb-wrr": {
							Weighted: &dynamic.TCPWeightedRoundRobin{
								Services: []dynamic.TCPWRRService{
									{
										Name:   "default-whoamitcp-9000",
										Weight: func(i int) *int { return &i }(1),
									},
								},
							},
						},
						"default-whoamitcp-9000": {
							LoadBalancer: &dynamic.TCPServersLoadBalancer{
								Servers: []dynamic.TCPServer{
									{
										Address: "10.10.0.9:9000",
									},
									{
										Address: "10.10.0.10:9000",
									},
								},
							},
						},
					},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers:     map[string]*dynamic.Router{},
					Middlewares: map[string]*dynamic.Middleware{},
					Services:    map[string]*dynamic.Service{},
				},
				TLS: &dynamic.TLSConfiguration{},
			},
		},
	}

	for _, test := range testCases {
		test := test

		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			if test.expected == nil {
				return
			}

			p := Provider{EntryPoints: test.entryPoints}
			conf := p.loadConfigurationFromGateway(context.Background(), newClientMock(test.paths...))
			assert.Equal(t, test.expected, conf)
		})
	}
}

func TestLoadMixedRoutes(t *testing.T) {
	testCases := []struct {
		desc         string
		ingressClass string
		paths        []string
		expected     *dynamic.Configuration
		entryPoints  map[string]Entrypoint
	}{
		{
			desc: "Empty",
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:     map[string]*dynamic.TCPRouter{},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services:    map[string]*dynamic.TCPService{},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers:     map[string]*dynamic.Router{},
					Middlewares: map[string]*dynamic.Middleware{},
					Services:    map[string]*dynamic.Service{},
				},
				TLS: &dynamic.TLSConfiguration{},
			},
		},
		{
			desc:  "Empty caused by unsupported listener.Protocol",
			paths: []string{"services.yml", "mixed/with_bad_listener_protocol.yml"},
			entryPoints: map[string]Entrypoint{
				"web": {Address: ":9080"},
			},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:     map[string]*dynamic.TCPRouter{},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services:    map[string]*dynamic.TCPService{},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers:     map[string]*dynamic.Router{},
					Middlewares: map[string]*dynamic.Middleware{},
					Services:    map[string]*dynamic.Service{},
				},
				TLS: &dynamic.TLSConfiguration{},
			},
		},
		{
			desc: "Empty caused by unsupported listener.Route.Kind",
			entryPoints: map[string]Entrypoint{
				"web": {Address: ":9080"},
			},
			paths: []string{"services.yml", "mixed/with_bad_listener_route_kind.yml"},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:     map[string]*dynamic.TCPRouter{},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services:    map[string]*dynamic.TCPService{},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers:     map[string]*dynamic.Router{},
					Middlewares: map[string]*dynamic.Middleware{},
					Services:    map[string]*dynamic.Service{},
				},
				TLS: &dynamic.TLSConfiguration{},
			},
		},
		{
			desc:  "Empty caused by listener.Protocol does not support listener.Route.Kind",
			paths: []string{"services.yml", "mixed/with_incompatible_protocol_and_route_kind.yml"},
			entryPoints: map[string]Entrypoint{
				"web": {Address: ":9080"},
			},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:     map[string]*dynamic.TCPRouter{},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services:    map[string]*dynamic.TCPService{},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers:     map[string]*dynamic.Router{},
					Middlewares: map[string]*dynamic.Middleware{},
					Services:    map[string]*dynamic.Service{},
				},
				TLS: &dynamic.TLSConfiguration{},
			},
		},
		{
			desc:  "Mixed routes",
			paths: []string{"services.yml", "mixed/simple.yml"},
			entryPoints: map[string]Entrypoint{
				"web":       {Address: ":9080"},
				"websecure": {Address: ":9443"},
				"tcp":       {Address: ":9000"},
				"tls-1":     {Address: ":10000"},
				"tls-2":     {Address: ":11000"},
			},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers: map[string]*dynamic.TCPRouter{
						"default-tcp-app-1-my-gateway-tcp-e3b0c44298fc1c149afb": {
							EntryPoints: []string{"tcp"},
							Service:     "default-tcp-app-1-my-gateway-tcp-e3b0c44298fc1c149afb-wrr",
							Rule:        "HostSNI(`*`)",
						},
						"default-tcp-app-1-my-gateway-tls-1-e3b0c44298fc1c149afb": {
							EntryPoints: []string{"tls-1"},
							Service:     "default-tcp-app-1-my-gateway-tls-1-e3b0c44298fc1c149afb-wrr",
							Rule:        "HostSNI(`*`)",
							TLS:         &dynamic.RouterTCPTLSConfig{},
						},
						"default-tls-app-1-my-gateway-tls-2-673acf455cb2dab0b43a": {
							EntryPoints: []string{"tls-2"},
							Service:     "default-tls-app-1-my-gateway-tls-2-673acf455cb2dab0b43a-wrr",
							Rule:        "HostSNI(`*`)",
							TLS: &dynamic.RouterTCPTLSConfig{
								Passthrough: true,
							},
						},
					},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services: map[string]*dynamic.TCPService{
						"default-tcp-app-1-my-gateway-tcp-e3b0c44298fc1c149afb-wrr": {
							Weighted: &dynamic.TCPWeightedRoundRobin{
								Services: []dynamic.TCPWRRService{
									{
										Name:   "default-whoamitcp-9000",
										Weight: func(i int) *int { return &i }(1),
									},
								},
							},
						},
						"default-tcp-app-1-my-gateway-tls-1-e3b0c44298fc1c149afb-wrr": {
							Weighted: &dynamic.TCPWeightedRoundRobin{
								Services: []dynamic.TCPWRRService{
									{
										Name:   "default-whoamitcp-9000",
										Weight: func(i int) *int { return &i }(1),
									},
								},
							},
						},
						"default-tls-app-1-my-gateway-tls-2-673acf455cb2dab0b43a-wrr": {
							Weighted: &dynamic.TCPWeightedRoundRobin{
								Services: []dynamic.TCPWRRService{
									{
										Name:   "default-whoamitcp-9000",
										Weight: func(i int) *int { return &i }(1),
									},
								},
							},
						},
						"default-whoamitcp-9000": {
							LoadBalancer: &dynamic.TCPServersLoadBalancer{
								Servers: []dynamic.TCPServer{
									{
										Address: "10.10.0.9:9000",
									},
									{
										Address: "10.10.0.10:9000",
									},
								},
							},
						},
					},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers: map[string]*dynamic.Router{
						"default-http-app-1-my-gateway-web-a431b128267aabc954fd": {
							EntryPoints: []string{"web"},
							Service:     "default-http-app-1-my-gateway-web-a431b128267aabc954fd-wrr",
							Rule:        "PathPrefix(`/`)",
						},
						"default-http-app-1-my-gateway-websecure-a431b128267aabc954fd": {
							EntryPoints: []string{"websecure"},
							Service:     "default-http-app-1-my-gateway-websecure-a431b128267aabc954fd-wrr",
							Rule:        "PathPrefix(`/`)",
							TLS:         &dynamic.RouterTLSConfig{},
						},
					},
					Middlewares: map[string]*dynamic.Middleware{},
					Services: map[string]*dynamic.Service{
						"default-http-app-1-my-gateway-web-a431b128267aabc954fd-wrr": {
							Weighted: &dynamic.WeightedRoundRobin{
								Services: []dynamic.WRRService{
									{
										Name:   "default-whoami-80",
										Weight: func(i int) *int { return &i }(1),
									},
								},
							},
						},
						"default-http-app-1-my-gateway-websecure-a431b128267aabc954fd-wrr": {
							Weighted: &dynamic.WeightedRoundRobin{
								Services: []dynamic.WRRService{
									{
										Name:   "default-whoami-80",
										Weight: func(i int) *int { return &i }(1),
									},
								},
							},
						},
						"default-whoami-80": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								Servers: []dynamic.Server{
									{
										URL: "http://10.10.0.1:80",
									},
									{
										URL: "http://10.10.0.2:80",
									},
								},
								PassHostHeader: Bool(true),
							},
						},
					},
				},
				TLS: &dynamic.TLSConfiguration{
					Certificates: []*tls.CertAndStores{
						{
							Certificate: tls.Certificate{
								CertFile: tls.FileOrContent("-----BEGIN CERTIFICATE-----\n-----END CERTIFICATE-----"),
								KeyFile:  tls.FileOrContent("-----BEGIN PRIVATE KEY-----\n-----END PRIVATE KEY-----"),
							},
						},
					},
				},
			},
		},
		{
			desc:  "Empty caused by mixed routes multiple protocol using same port",
			paths: []string{"services.yml", "mixed/with_multiple_protocol_using_same_port.yml"},
			entryPoints: map[string]Entrypoint{
				"web": {Address: ":9080"},
				"tcp": {Address: ":9000"},
				"tls": {Address: ":9001"},
			},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:     map[string]*dynamic.TCPRouter{},
					Middlewares: map[string]*dynamic.TCPMiddleware{},
					Services:    map[string]*dynamic.TCPService{},
				},
				HTTP: &dynamic.HTTPConfiguration{
					Routers:     map[string]*dynamic.Router{},
					Middlewares: map[string]*dynamic.Middleware{},
					Services:    map[string]*dynamic.Service{},
				},
				TLS: &dynamic.TLSConfiguration{},
			},
		},
	}

	for _, test := range testCases {
		test := test

		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			if test.expected == nil {
				return
			}

			p := Provider{EntryPoints: test.entryPoints}
			conf := p.loadConfigurationFromGateway(context.Background(), newClientMock(test.paths...))
			assert.Equal(t, test.expected, conf)
		})
	}
}

func TestHostRule(t *testing.T) {
	testCases := []struct {
		desc         string
		routeSpec    v1alpha1.HTTPRouteSpec
		expectedRule string
		expectErr    bool
	}{
		{
			desc:         "Empty rule and matches",
			expectedRule: "",
		},
		{
			desc: "One Host",
			routeSpec: v1alpha1.HTTPRouteSpec{
				Hostnames: []v1alpha1.Hostname{
					"Foo",
				},
			},
			expectedRule: "Host(`Foo`)",
		},
		{
			desc: "Multiple Hosts",
			routeSpec: v1alpha1.HTTPRouteSpec{
				Hostnames: []v1alpha1.Hostname{
					"Foo",
					"Bar",
					"Bir",
				},
			},
			expectedRule: "Host(`Foo`, `Bar`, `Bir`)",
		},
		{
			desc: "Multiple Hosts with empty one",
			routeSpec: v1alpha1.HTTPRouteSpec{
				Hostnames: []v1alpha1.Hostname{
					"Foo",
					"",
					"Bir",
				},
			},
			expectedRule: "",
		},
		{
			desc: "Multiple empty hosts",
			routeSpec: v1alpha1.HTTPRouteSpec{
				Hostnames: []v1alpha1.Hostname{
					"",
					"",
					"",
				},
			},
			expectedRule: "",
		},
		{
			desc: "Several Host and wildcard",
			routeSpec: v1alpha1.HTTPRouteSpec{
				Hostnames: []v1alpha1.Hostname{
					"*.bar.foo",
					"bar.foo",
					"foo.foo",
				},
			},
			expectedRule: "(Host(`bar.foo`, `foo.foo`) || HostRegexp(`{subdomain:[a-zA-Z0-9-]+}.bar.foo`))",
		},
		{
			desc: "Host with wildcard",
			routeSpec: v1alpha1.HTTPRouteSpec{
				Hostnames: []v1alpha1.Hostname{
					"*.bar.foo",
				},
			},
			expectedRule: "HostRegexp(`{subdomain:[a-zA-Z0-9-]+}.bar.foo`)",
		},
		{
			desc: "Alone wildcard",
			routeSpec: v1alpha1.HTTPRouteSpec{
				Hostnames: []v1alpha1.Hostname{
					"*",
					"*.foo.foo",
				},
			},
		},
		{
			desc: "Multiple alone Wildcard",
			routeSpec: v1alpha1.HTTPRouteSpec{
				Hostnames: []v1alpha1.Hostname{
					"foo.foo",
					"*.*",
				},
			},
			expectErr: true,
		},
		{
			desc: "Multiple Wildcard",
			routeSpec: v1alpha1.HTTPRouteSpec{
				Hostnames: []v1alpha1.Hostname{
					"foo.foo",
					"*.toto.*.bar.foo",
				},
			},
			expectErr: true,
		},
		{
			desc: "Multiple subdomain with misplaced wildcard",
			routeSpec: v1alpha1.HTTPRouteSpec{
				Hostnames: []v1alpha1.Hostname{
					"foo.foo",
					"toto.*.bar.foo",
				},
			},
			expectErr: true,
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()
			rule, err := hostRule(test.routeSpec)

			assert.Equal(t, test.expectedRule, rule)
			if test.expectErr {
				assert.Error(t, err)
			}
		})
	}
}

func TestExtractRule(t *testing.T) {
	testCases := []struct {
		desc          string
		routeRule     v1alpha1.HTTPRouteRule
		hostRule      string
		expectedRule  string
		expectedError bool
	}{
		{
			desc:         "Empty rule and matches",
			expectedRule: "PathPrefix(`/`)",
		},
		{
			desc:         "One Host rule without matches",
			hostRule:     "Host(`foo.com`)",
			expectedRule: "Host(`foo.com`) && PathPrefix(`/`)",
		},
		{
			desc: "One Path in matches",
			routeRule: v1alpha1.HTTPRouteRule{
				Matches: []v1alpha1.HTTPRouteMatch{
					{
						Path: v1alpha1.HTTPPathMatch{
							Type:  v1alpha1.PathMatchExact,
							Value: "/foo/",
						},
					},
				},
			},
			expectedRule: "Path(`/foo/`)",
		},
		{
			desc: "One Path in matches and another unknown",
			routeRule: v1alpha1.HTTPRouteRule{
				Matches: []v1alpha1.HTTPRouteMatch{
					{
						Path: v1alpha1.HTTPPathMatch{
							Type:  v1alpha1.PathMatchExact,
							Value: "/foo/",
						},
					},
					{
						Path: v1alpha1.HTTPPathMatch{
							Type:  "unknown",
							Value: "/foo/",
						},
					},
				},
			},
			expectedError: true,
		},
		{
			desc: "One Path in matches and another empty",
			routeRule: v1alpha1.HTTPRouteRule{
				Matches: []v1alpha1.HTTPRouteMatch{
					{
						Path: v1alpha1.HTTPPathMatch{
							Type:  v1alpha1.PathMatchExact,
							Value: "/foo/",
						},
					},
					{},
				},
			},
			expectedRule: "Path(`/foo/`)",
		},
		{
			desc: "Path OR Header rules",
			routeRule: v1alpha1.HTTPRouteRule{
				Matches: []v1alpha1.HTTPRouteMatch{
					{
						Path: v1alpha1.HTTPPathMatch{
							Type:  v1alpha1.PathMatchExact,
							Value: "/foo/",
						},
					},
					{
						Headers: &v1alpha1.HTTPHeaderMatch{
							Type: v1alpha1.HeaderMatchExact,
							Values: map[string]string{
								"my-header": "foo",
							},
						},
					},
				},
			},
			expectedRule: "Path(`/foo/`) || Headers(`my-header`,`foo`)",
		},
		{
			desc: "Path && Header rules",
			routeRule: v1alpha1.HTTPRouteRule{
				Matches: []v1alpha1.HTTPRouteMatch{
					{
						Path: v1alpha1.HTTPPathMatch{
							Type:  v1alpha1.PathMatchExact,
							Value: "/foo/",
						},
						Headers: &v1alpha1.HTTPHeaderMatch{
							Type: v1alpha1.HeaderMatchExact,
							Values: map[string]string{
								"my-header": "foo",
							},
						},
					},
				},
			},
			expectedRule: "Path(`/foo/`) && Headers(`my-header`,`foo`)",
		},
		{
			desc:     "Host && Path && Header rules",
			hostRule: "Host(`foo.com`)",
			routeRule: v1alpha1.HTTPRouteRule{
				Matches: []v1alpha1.HTTPRouteMatch{
					{
						Path: v1alpha1.HTTPPathMatch{
							Type:  v1alpha1.PathMatchExact,
							Value: "/foo/",
						},
						Headers: &v1alpha1.HTTPHeaderMatch{
							Type: v1alpha1.HeaderMatchExact,
							Values: map[string]string{
								"my-header": "foo",
							},
						},
					},
				},
			},
			expectedRule: "Host(`foo.com`) && Path(`/foo/`) && Headers(`my-header`,`foo`)",
		},
		{
			desc:     "Host && (Path || Header) rules",
			hostRule: "Host(`foo.com`)",
			routeRule: v1alpha1.HTTPRouteRule{
				Matches: []v1alpha1.HTTPRouteMatch{
					{
						Path: v1alpha1.HTTPPathMatch{
							Type:  v1alpha1.PathMatchExact,
							Value: "/foo/",
						},
					},
					{
						Headers: &v1alpha1.HTTPHeaderMatch{
							Type: v1alpha1.HeaderMatchExact,
							Values: map[string]string{
								"my-header": "foo",
							},
						},
					},
				},
			},
			expectedRule: "Host(`foo.com`) && (Path(`/foo/`) || Headers(`my-header`,`foo`))",
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			rule, err := extractRule(test.routeRule, test.hostRule)
			if test.expectedError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.expectedRule, rule)
		})
	}
}

func TestHostSNI(t *testing.T) {
	testCases := []struct {
		desc         string
		routeRule    v1alpha1.TLSRouteRule
		expectedRule string
		expectError  bool
	}{
		{
			desc:         "Empty rule",
			expectedRule: "HostSNI(`*`)",
		},
		{
			desc: "Empty rule and matches",
			routeRule: v1alpha1.TLSRouteRule{
				Matches: []v1alpha1.TLSRouteMatch{},
			},
			expectedRule: "HostSNI(`*`)",
		},
		{
			desc: "One match, SNI with empty hostname",
			routeRule: v1alpha1.TLSRouteRule{
				Matches: []v1alpha1.TLSRouteMatch{
					{
						SNIs: []v1alpha1.Hostname{""},
					},
				},
			},
			expectedRule: "HostSNI(`*`)",
		},
		{
			desc: "One match, SNI with one unsupported wildcard",
			routeRule: v1alpha1.TLSRouteRule{
				Matches: []v1alpha1.TLSRouteMatch{
					{
						SNIs: []v1alpha1.Hostname{"*"},
					},
				},
			},
			expectError: true,
		},
		{
			desc: "One match, SNI with multiple malformed wildcard",
			routeRule: v1alpha1.TLSRouteRule{
				Matches: []v1alpha1.TLSRouteMatch{
					{
						SNIs: []v1alpha1.Hostname{"*.foo.*"},
					},
				},
			},
			expectError: true,
		},
		{
			desc: "One match, SNI with some empty hostnames",
			routeRule: v1alpha1.TLSRouteRule{
				Matches: []v1alpha1.TLSRouteMatch{
					{
						SNIs: []v1alpha1.Hostname{"foo", "", "bar"},
					},
				},
			},
			expectedRule: "HostSNI(`foo`,`bar`)",
		},
		{
			desc: "One match, one SNI hostname",
			routeRule: v1alpha1.TLSRouteRule{
				Matches: []v1alpha1.TLSRouteMatch{
					{
						SNIs: []v1alpha1.Hostname{"foo"},
					},
				},
			},
			expectedRule: "HostSNI(`foo`)",
		},
		{
			desc: "One match, multiple SNI hostnames",
			routeRule: v1alpha1.TLSRouteRule{
				Matches: []v1alpha1.TLSRouteMatch{
					{
						SNIs: []v1alpha1.Hostname{"foo", "bar"},
					},
				},
			},
			expectedRule: "HostSNI(`foo`,`bar`)",
		},
		{
			desc: "One SNI multiple hostnames",
			routeRule: v1alpha1.TLSRouteRule{
				Matches: []v1alpha1.TLSRouteMatch{
					{
						SNIs: []v1alpha1.Hostname{"foo", "bar"},
					},
				},
			},
			expectedRule: "HostSNI(`foo`,`bar`)",
		},
		{
			desc: "Multiple SNI multiple hostnames",
			routeRule: v1alpha1.TLSRouteRule{
				Matches: []v1alpha1.TLSRouteMatch{
					{
						SNIs: []v1alpha1.Hostname{"foo", "bar"},
					},
					{
						SNIs: []v1alpha1.Hostname{"foz", "baz"},
					},
				},
			},
			expectedRule: "HostSNI(`foo`,`bar`,`foz`,`baz`)",
		},
		{
			desc: "Multiple SNI multiple hostnames overlapping",
			routeRule: v1alpha1.TLSRouteRule{
				Matches: []v1alpha1.TLSRouteMatch{
					{
						SNIs: []v1alpha1.Hostname{"foo", "bar"},
					},
					{
						SNIs: []v1alpha1.Hostname{"foo", "baz"},
					},
				},
			},
			expectedRule: "HostSNI(`foo`,`bar`,`baz`)",
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			rule, err := hostSNIRule(test.routeRule)
			if test.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.expectedRule, rule)
		})
	}
}
