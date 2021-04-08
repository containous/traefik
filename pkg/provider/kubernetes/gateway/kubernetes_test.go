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
					Routers:  map[string]*dynamic.TCPRouter{},
					Services: map[string]*dynamic.TCPService{},
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
			paths: []string{"services.yml", "simple.yml"},
			entryPoints: map[string]Entrypoint{"web": {
				Address: ":443",
			}},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:  map[string]*dynamic.TCPRouter{},
					Services: map[string]*dynamic.TCPService{},
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
			paths: []string{"services.yml", "without_httproute.yml"},
			entryPoints: map[string]Entrypoint{"web": {
				Address: ":80",
			}},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:  map[string]*dynamic.TCPRouter{},
					Services: map[string]*dynamic.TCPService{},
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
			paths: []string{"services.yml", "without_gatewayclass.yml"},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:  map[string]*dynamic.TCPRouter{},
					Services: map[string]*dynamic.TCPService{},
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
			paths: []string{"services.yml", "gatewayclass_with_unknown_controller.yml"},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:  map[string]*dynamic.TCPRouter{},
					Services: map[string]*dynamic.TCPService{},
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
			paths: []string{"services.yml", "with_wrong_service_port.yml"},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:  map[string]*dynamic.TCPRouter{},
					Services: map[string]*dynamic.TCPService{},
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
			paths: []string{"services.yml", "with_protocol_https_without_tls.yml"},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:  map[string]*dynamic.TCPRouter{},
					Services: map[string]*dynamic.TCPService{},
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
			desc:  "Simple HTTPRoute, with foo entrypoint",
			paths: []string{"services.yml", "simple.yml"},
			entryPoints: map[string]Entrypoint{"web": {
				Address: ":80",
			}},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:  map[string]*dynamic.TCPRouter{},
					Services: map[string]*dynamic.TCPService{},
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
			paths: []string{"services.yml", "simple_to_api_internal.yml"},
			entryPoints: map[string]Entrypoint{"web": {
				Address: ":80",
			}},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:  map[string]*dynamic.TCPRouter{},
					Services: map[string]*dynamic.TCPService{},
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
			paths: []string{"services.yml", "simple_cross_provider.yml"},
			entryPoints: map[string]Entrypoint{"web": {
				Address: ":80",
			}},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:  map[string]*dynamic.TCPRouter{},
					Services: map[string]*dynamic.TCPService{},
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
			paths: []string{"services.yml", "with_protocol_https.yml"},
			entryPoints: map[string]Entrypoint{"websecure": {
				Address: ":443",
			}},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:  map[string]*dynamic.TCPRouter{},
					Services: map[string]*dynamic.TCPService{},
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
			paths: []string{"services.yml", "with_multiple_host.yml"},
			entryPoints: map[string]Entrypoint{"web": {
				Address: ":80",
			}},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:  map[string]*dynamic.TCPRouter{},
					Services: map[string]*dynamic.TCPService{},
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
			desc:  "One HTTPRoute with two different rules",
			paths: []string{"services.yml", "two_rules.yml"},
			entryPoints: map[string]Entrypoint{"web": {
				Address: ":80",
			}},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:  map[string]*dynamic.TCPRouter{},
					Services: map[string]*dynamic.TCPService{},
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
			paths: []string{"services.yml", "one_rule_two_targets.yml"},
			entryPoints: map[string]Entrypoint{"web": {
				Address: ":80",
			}},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:  map[string]*dynamic.TCPRouter{},
					Services: map[string]*dynamic.TCPService{},
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
			paths: []string{"services.yml", "with_two_gateways_one_httproute.yml"},
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
					Routers:  map[string]*dynamic.TCPRouter{},
					Services: map[string]*dynamic.TCPService{},
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
			paths: []string{"services.yml", "with_two_listeners_one_httproute.yml"},
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
					Routers:  map[string]*dynamic.TCPRouter{},
					Services: map[string]*dynamic.TCPService{},
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
			paths: []string{"services.yml", "with_several_rules.yml"},
			entryPoints: map[string]Entrypoint{"web": {
				Address: ":80",
			}},
			expected: &dynamic.Configuration{
				UDP: &dynamic.UDPConfiguration{
					Routers:  map[string]*dynamic.UDPRouter{},
					Services: map[string]*dynamic.UDPService{},
				},
				TCP: &dynamic.TCPConfiguration{
					Routers:  map[string]*dynamic.TCPRouter{},
					Services: map[string]*dynamic.TCPService{},
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

func TestHostRule(t *testing.T) {
	testCases := []struct {
		desc         string
		routeSpec    v1alpha1.HTTPRouteSpec
		expectedRule string
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
			expectedRule: "Host(`Foo`, `Bir`)",
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
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, test.expectedRule, hostRule(test.routeSpec))
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
