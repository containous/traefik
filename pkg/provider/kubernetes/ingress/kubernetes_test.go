package ingress

import (
	"context"
	"errors"
	"math"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/provider"
	"github.com/traefik/traefik/v2/pkg/tls"
	"github.com/traefik/traefik/v2/pkg/types"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ provider.Provider = (*Provider)(nil)

func Bool(v bool) *bool { return &v }

func TestLoadConfigurationFromIngresses(t *testing.T) {
	testCases := []struct {
		desc               string
		ingressClass       string
		serverVersion      string
		expected           *dynamic.Configuration
		allowEmptyServices bool
	}{
		{
			desc: "Empty ingresses",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Routers:     map[string]*dynamic.Router{},
					Middlewares: map[string]*dynamic.Middleware{},
					Services:    map[string]*dynamic.Service{},
				},
			},
		},
		{
			desc: "Ingress one rule host only",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Routers:     map[string]*dynamic.Router{},
					Middlewares: map[string]*dynamic.Middleware{},
					Services:    map[string]*dynamic.Service{},
				},
			},
		},
		{
			desc: "Ingress with a basic rule on one path",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},
					Routers: map[string]*dynamic.Router{
						"testing-bar": {
							Rule:    "PathPrefix(`/bar`)",
							Service: "testing-service1-80",
						},
					},
					Services: map[string]*dynamic.Service{
						"testing-service1-80": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								PassHostHeader: Bool(true),
								Servers: []dynamic.Server{
									{
										URL: "http://10.10.0.1:8080",
									},
									{
										URL: "http://10.21.0.1:8080",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc: "Ingress with annotations",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},
					Routers: map[string]*dynamic.Router{
						"testing-bar": {
							Rule:        "Path(`/bar`)",
							EntryPoints: []string{"ep1", "ep2"},
							Service:     "testing-service1-80",
							Middlewares: []string{"md1", "md2"},
							Priority:    42,
							TLS: &dynamic.RouterTLSConfig{
								CertResolver: "foobar",
								Domains: []types.Domain{
									{
										Main: "domain.com",
										SANs: []string{"one.domain.com", "two.domain.com"},
									},
									{
										Main: "example.com",
										SANs: []string{"one.example.com", "two.example.com"},
									},
								},
								Options: "foobar",
							},
						},
					},
					Services: map[string]*dynamic.Service{
						"testing-service1-80": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								PassHostHeader: Bool(true),
								Sticky: &dynamic.Sticky{
									Cookie: &dynamic.Cookie{
										Name:     "foobar",
										Secure:   true,
										HTTPOnly: true,
									},
								},
								Servers: []dynamic.Server{
									{
										URL: "protocol://10.10.0.1:8080",
									},
									{
										URL: "protocol://10.21.0.1:8080",
									},
								},
								ServersTransport: "foobar@file",
							},
						},
					},
				},
			},
		},
		{
			desc: "Ingress with two different rules with one path",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},
					Routers: map[string]*dynamic.Router{
						"testing-bar": {
							Rule:    "PathPrefix(`/bar`)",
							Service: "testing-service1-80",
						},
						"testing-foo": {
							Rule:    "PathPrefix(`/foo`)",
							Service: "testing-service1-80",
						},
					},
					Services: map[string]*dynamic.Service{
						"testing-service1-80": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								PassHostHeader: Bool(true),
								Servers: []dynamic.Server{
									{
										URL: "http://10.10.0.1:8080",
									},
									{
										URL: "http://10.21.0.1:8080",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc: "Ingress with conflicting routers on host",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},
					Routers: map[string]*dynamic.Router{
						"testing-bar-bar-3be6cfd7daba66cf2fdd": {
							Rule:    "HostRegexp(`{subdomain:[a-zA-Z0-9-]+}.bar`) && PathPrefix(`/bar`)",
							Service: "testing-service1-80",
						},
						"testing-bar-bar-636bf36c00fedaab3d44": {
							Rule:    "Host(`bar`) && PathPrefix(`/bar`)",
							Service: "testing-service1-80",
						},
					},
					Services: map[string]*dynamic.Service{
						"testing-service1-80": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								PassHostHeader: Bool(true),
								Servers: []dynamic.Server{
									{
										URL: "http://10.10.0.1:8080",
									},
									{
										URL: "http://10.21.0.1:8080",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc: "Ingress with conflicting routers on path",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},
					Routers: map[string]*dynamic.Router{
						"testing-foo-bar-d0b30949e54d6a7515ca": {
							Rule:    "PathPrefix(`/foo/bar`)",
							Service: "testing-service1-80",
						},
						"testing-foo-bar-dcd54bae39a6d7557f48": {
							Rule:    "PathPrefix(`/foo-bar`)",
							Service: "testing-service1-80",
						},
					},
					Services: map[string]*dynamic.Service{
						"testing-service1-80": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								PassHostHeader: Bool(true),
								Servers: []dynamic.Server{
									{
										URL: "http://10.10.0.1:8080",
									},
									{
										URL: "http://10.21.0.1:8080",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc: "Ingress one rule with two paths",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},
					Routers: map[string]*dynamic.Router{
						"testing-bar": {
							Rule:    "PathPrefix(`/bar`)",
							Service: "testing-service1-80",
						},
						"testing-foo": {
							Rule:    "PathPrefix(`/foo`)",
							Service: "testing-service1-80",
						},
					},
					Services: map[string]*dynamic.Service{
						"testing-service1-80": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								PassHostHeader: Bool(true),
								Servers: []dynamic.Server{
									{
										URL: "http://10.10.0.1:8080",
									},
									{
										URL: "http://10.21.0.1:8080",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc: "Ingress one rule with one path and one host",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},
					Routers: map[string]*dynamic.Router{
						"testing-traefik-tchouk-bar": {
							Rule:    "Host(`traefik.tchouk`) && PathPrefix(`/bar`)",
							Service: "testing-service1-80",
						},
					},
					Services: map[string]*dynamic.Service{
						"testing-service1-80": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								PassHostHeader: Bool(true),
								Servers: []dynamic.Server{
									{
										URL: "http://10.10.0.1:8080",
									},
									{
										URL: "http://10.21.0.1:8080",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc: "Ingress with one host without path",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},
					Routers: map[string]*dynamic.Router{
						"testing-example-com": {
							Rule:    "Host(`example.com`)",
							Service: "testing-example-com-80",
						},
					},
					Services: map[string]*dynamic.Service{
						"testing-example-com-80": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								PassHostHeader: Bool(true),
								Servers: []dynamic.Server{
									{
										URL: "http://10.11.0.1:80",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc: "Ingress one rule with one host and two paths",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},
					Routers: map[string]*dynamic.Router{
						"testing-traefik-tchouk-bar": {
							Rule:    "Host(`traefik.tchouk`) && PathPrefix(`/bar`)",
							Service: "testing-service1-80",
						},
						"testing-traefik-tchouk-foo": {
							Rule:    "Host(`traefik.tchouk`) && PathPrefix(`/foo`)",
							Service: "testing-service1-80",
						},
					},
					Services: map[string]*dynamic.Service{
						"testing-service1-80": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								PassHostHeader: Bool(true),
								Servers: []dynamic.Server{
									{
										URL: "http://10.10.0.1:8080",
									},
									{
										URL: "http://10.21.0.1:8080",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc: "Ingress Two rules with one host and one path",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},
					Routers: map[string]*dynamic.Router{
						"testing-traefik-tchouk-bar": {
							Rule:    "Host(`traefik.tchouk`) && PathPrefix(`/bar`)",
							Service: "testing-service1-80",
						},
						"testing-traefik-courgette-carotte": {
							Rule:    "Host(`traefik.courgette`) && PathPrefix(`/carotte`)",
							Service: "testing-service1-80",
						},
					},
					Services: map[string]*dynamic.Service{
						"testing-service1-80": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								PassHostHeader: Bool(true),
								Servers: []dynamic.Server{
									{
										URL: "http://10.10.0.1:8080",
									},
									{
										URL: "http://10.21.0.1:8080",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc: "Ingress with two services",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},
					Routers: map[string]*dynamic.Router{
						"testing-traefik-tchouk-bar": {
							Rule:    "Host(`traefik.tchouk`) && PathPrefix(`/bar`)",
							Service: "testing-service1-80",
						},
						"testing-traefik-courgette-carotte": {
							Rule:    "Host(`traefik.courgette`) && PathPrefix(`/carotte`)",
							Service: "testing-service2-8082",
						},
					},
					Services: map[string]*dynamic.Service{
						"testing-service1-80": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								PassHostHeader: Bool(true),
								Servers: []dynamic.Server{
									{
										URL: "http://10.10.0.1:8080",
									},
									{
										URL: "http://10.21.0.1:8080",
									},
								},
							},
						},
						"testing-service2-8082": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								PassHostHeader: Bool(true),
								Servers: []dynamic.Server{
									{
										URL: "http://10.10.0.2:8080",
									},
									{
										URL: "http://10.21.0.2:8080",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc:               "Ingress with one service without endpoints subset",
			allowEmptyServices: true,
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},
					Routers: map[string]*dynamic.Router{
						"testing-traefik-tchouk-bar": {
							Rule:    "Host(`traefik.tchouk`) && PathPrefix(`/bar`)",
							Service: "testing-service1-80",
						},
					},
					Services: map[string]*dynamic.Service{
						"testing-service1-80": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								PassHostHeader: Bool(true),
							},
						},
					},
				},
			},
		},
		{
			desc: "Ingress with one service without endpoint",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},
					Routers:     map[string]*dynamic.Router{},
					Services:    map[string]*dynamic.Service{},
				},
			},
		},
		{
			desc: "Single Service Ingress (without any rules)",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},
					Routers: map[string]*dynamic.Router{
						"default-router": {
							Rule:     "PathPrefix(`/`)",
							Service:  "default-backend",
							Priority: math.MinInt32,
						},
					},
					Services: map[string]*dynamic.Service{
						"default-backend": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								PassHostHeader: Bool(true),
								Servers: []dynamic.Server{
									{
										URL: "http://10.10.0.1:8080",
									},
									{
										URL: "http://10.21.0.1:8080",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc: "Ingress with port value in backend and no pod replica",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},
					Routers: map[string]*dynamic.Router{
						"testing-traefik-tchouk-bar": {
							Rule:    "Host(`traefik.tchouk`) && PathPrefix(`/bar`)",
							Service: "testing-service1-80",
						},
					},
					Services: map[string]*dynamic.Service{
						"testing-service1-80": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								PassHostHeader: Bool(true),
								Servers: []dynamic.Server{
									{
										URL: "http://10.10.0.1:8089",
									},
									{
										URL: "http://10.21.0.1:8089",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc: "Ingress with port name in backend and no pod replica",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},
					Routers: map[string]*dynamic.Router{
						"testing-traefik-tchouk-bar": {
							Rule:    "Host(`traefik.tchouk`) && PathPrefix(`/bar`)",
							Service: "testing-service1-tchouk",
						},
					},
					Services: map[string]*dynamic.Service{
						"testing-service1-tchouk": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								PassHostHeader: Bool(true),
								Servers: []dynamic.Server{
									{
										URL: "http://10.10.0.1:8089",
									},
									{
										URL: "http://10.21.0.1:8089",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc: "Ingress with with port name in backend and 2 pod replica",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},
					Routers: map[string]*dynamic.Router{
						"testing-traefik-tchouk-bar": {
							Rule:    "Host(`traefik.tchouk`) && PathPrefix(`/bar`)",
							Service: "testing-service1-tchouk",
						},
					},
					Services: map[string]*dynamic.Service{
						"testing-service1-tchouk": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								PassHostHeader: Bool(true),
								Servers: []dynamic.Server{
									{
										URL: "http://10.10.0.1:8089",
									},
									{
										URL: "http://10.10.0.2:8089",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc: "Ingress with two paths using same service and different port name",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},
					Routers: map[string]*dynamic.Router{
						"testing-traefik-tchouk-bar": {
							Rule:    "Host(`traefik.tchouk`) && PathPrefix(`/bar`)",
							Service: "testing-service1-tchouk",
						},
						"testing-traefik-tchouk-foo": {
							Rule:    "Host(`traefik.tchouk`) && PathPrefix(`/foo`)",
							Service: "testing-service1-carotte",
						},
					},
					Services: map[string]*dynamic.Service{
						"testing-service1-tchouk": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								PassHostHeader: Bool(true),
								Servers: []dynamic.Server{
									{
										URL: "http://10.10.0.1:8089",
									},
									{
										URL: "http://10.10.0.2:8089",
									},
								},
							},
						},
						"testing-service1-carotte": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								PassHostHeader: Bool(true),
								Servers: []dynamic.Server{
									{
										URL: "http://10.10.0.1:8090",
									},
									{
										URL: "http://10.10.0.2:8090",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc: "2 ingresses in different namespace with same service name",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},
					Routers: map[string]*dynamic.Router{
						"testing-traefik-tchouk-bar": {
							Rule:    "Host(`traefik.tchouk`) && PathPrefix(`/bar`)",
							Service: "testing-service1-tchouk",
						},
						"toto-toto-traefik-tchouk-bar": {
							Rule:    "Host(`toto.traefik.tchouk`) && PathPrefix(`/bar`)",
							Service: "toto-service1-tchouk",
						},
					},
					Services: map[string]*dynamic.Service{
						"testing-service1-tchouk": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								PassHostHeader: Bool(true),
								Servers: []dynamic.Server{
									{
										URL: "http://10.10.0.1:8089",
									},
									{
										URL: "http://10.10.0.2:8089",
									},
								},
							},
						},
						"toto-service1-tchouk": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								PassHostHeader: Bool(true),
								Servers: []dynamic.Server{
									{
										URL: "http://10.11.0.1:8089",
									},
									{
										URL: "http://10.11.0.2:8089",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc: "Ingress with unknown service port name",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},
					Routers:     map[string]*dynamic.Router{},
					Services:    map[string]*dynamic.Service{},
				},
			},
		},
		{
			desc: "Ingress with unknown service port",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},
					Routers:     map[string]*dynamic.Router{},
					Services:    map[string]*dynamic.Service{},
				},
			},
		},
		{
			desc: "Ingress with service with externalName",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},
					Routers: map[string]*dynamic.Router{
						"testing-traefik-tchouk-bar": {
							Rule:    "Host(`traefik.tchouk`) && PathPrefix(`/bar`)",
							Service: "testing-service1-8080",
						},
					},
					Services: map[string]*dynamic.Service{
						"testing-service1-8080": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								PassHostHeader: Bool(true),
								Servers: []dynamic.Server{
									{
										URL: "http://traefik.wtf:8080",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc: "Ingress with port invalid for one service",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},
					Routers: map[string]*dynamic.Router{
						"testing-traefik-port-port": {
							Rule:    "Host(`traefik.port`) && PathPrefix(`/port`)",
							Service: "testing-service1-8080",
						},
					},
					Services: map[string]*dynamic.Service{
						"testing-service1-8080": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								PassHostHeader: Bool(true),
								Servers: []dynamic.Server{
									{
										URL: "http://10.0.0.1:8080",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc: "Ingress with IPv6 endpoints",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},
					Routers: map[string]*dynamic.Router{
						"example-com-testing-bar": {
							Rule:    "PathPrefix(`/bar`)",
							Service: "testing-service-bar-8080",
						},
						"example-com-testing-foo": {
							Rule:    "PathPrefix(`/foo`)",
							Service: "testing-service-foo-8080",
						},
					},
					Services: map[string]*dynamic.Service{
						"testing-service-bar-8080": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								Servers: []dynamic.Server{
									{
										URL: "http://[2001:0db8:3c4d:0015:0000:0000:1a2f:1a2b]:8080",
									},
								},
								PassHostHeader: Bool(true),
							},
						},
						"testing-service-foo-8080": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								Servers: []dynamic.Server{
									{
										URL: "http://[2001:0db8:3c4d:0015:0000:0000:1a2f:2a3b]:8080",
									},
								},
								PassHostHeader: Bool(true),
							},
						},
					},
				},
			},
		},
		{
			desc: "TLS support",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},
					Routers: map[string]*dynamic.Router{
						"testing-example-com": {
							Rule:    "Host(`example.com`)",
							Service: "testing-example-com-80",
							TLS:     &dynamic.RouterTLSConfig{},
						},
					},
					Services: map[string]*dynamic.Service{
						"testing-example-com-80": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								PassHostHeader: Bool(true),
								Servers: []dynamic.Server{
									{
										URL: "http://10.11.0.1:80",
									},
								},
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
			desc: "Ingress with a basic rule on one path with https (port == 443)",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},
					Routers: map[string]*dynamic.Router{
						"testing-bar": {
							Rule:    "PathPrefix(`/bar`)",
							Service: "testing-service1-443",
						},
					},
					Services: map[string]*dynamic.Service{
						"testing-service1-443": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								PassHostHeader: Bool(true),
								Servers: []dynamic.Server{
									{
										URL: "https://10.10.0.1:8443",
									},
									{
										URL: "https://10.21.0.1:8443",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc: "Ingress with a basic rule on one path with https (portname == https)",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},
					Routers: map[string]*dynamic.Router{
						"testing-bar": {
							Rule:    "PathPrefix(`/bar`)",
							Service: "testing-service1-8443",
						},
					},
					Services: map[string]*dynamic.Service{
						"testing-service1-8443": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								PassHostHeader: Bool(true),
								Servers: []dynamic.Server{
									{
										URL: "https://10.10.0.1:8443",
									},
									{
										URL: "https://10.21.0.1:8443",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc: "Ingress with a basic rule on one path with https (portname starts with https)",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},

					Routers: map[string]*dynamic.Router{
						"testing-bar": {
							Rule:    "PathPrefix(`/bar`)",
							Service: "testing-service1-8443",
						},
					},
					Services: map[string]*dynamic.Service{
						"testing-service1-8443": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								PassHostHeader: Bool(true),
								Servers: []dynamic.Server{
									{
										URL: "https://10.10.0.1:8443",
									},
									{
										URL: "https://10.21.0.1:8443",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc: "Double Single Service Ingress",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},
					Routers: map[string]*dynamic.Router{
						"default-router": {
							Rule:     "PathPrefix(`/`)",
							Service:  "default-backend",
							Priority: math.MinInt32,
						},
					},
					Services: map[string]*dynamic.Service{
						"default-backend": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								PassHostHeader: Bool(true),
								Servers: []dynamic.Server{
									{
										URL: "http://10.30.0.1:8080",
									},
									{
										URL: "http://10.41.0.1:8080",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc: "Ingress with default traefik ingressClass",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},
					Routers: map[string]*dynamic.Router{
						"testing-bar": {
							Rule:    "PathPrefix(`/bar`)",
							Service: "testing-service1-80",
						},
					},
					Services: map[string]*dynamic.Service{
						"testing-service1-80": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								PassHostHeader: Bool(true),
								Servers: []dynamic.Server{
									{
										URL: "http://10.10.0.1:8080",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc: "Ingress without provider traefik ingressClass and unknown annotation",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},
					Routers:     map[string]*dynamic.Router{},
					Services:    map[string]*dynamic.Service{},
				},
			},
		},
		{
			desc:         "Ingress with non matching provider traefik ingressClass and annotation",
			ingressClass: "tchouk",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},
					Routers:     map[string]*dynamic.Router{},
					Services:    map[string]*dynamic.Service{},
				},
			},
		},
		{
			desc:         "Ingress with ingressClass without annotation",
			ingressClass: "tchouk",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},
					Routers:     map[string]*dynamic.Router{},
					Services:    map[string]*dynamic.Service{},
				},
			},
		},
		{
			desc:         "Ingress with ingressClass without annotation",
			ingressClass: "toto",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},
					Routers:     map[string]*dynamic.Router{},
					Services:    map[string]*dynamic.Service{},
				},
			},
		},
		{
			desc: "Ingress with wildcard host",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},
					Routers: map[string]*dynamic.Router{
						"testing-foobar-com-bar": {
							Rule:    "HostRegexp(`{subdomain:[a-zA-Z0-9-]+}.foobar.com`) && PathPrefix(`/bar`)",
							Service: "testing-service1-80",
						},
					},
					Services: map[string]*dynamic.Service{
						"testing-service1-80": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								PassHostHeader: Bool(true),
								Servers: []dynamic.Server{
									{
										URL:    "http://10.10.0.1:8080",
										Scheme: "",
										Port:   "",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc:          "v18 Ingress with ingressClass",
			serverVersion: "v1.18",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},
					Routers: map[string]*dynamic.Router{
						"testing-bar": {
							Rule:    "PathPrefix(`/bar`)",
							Service: "testing-service1-80",
						},
					},
					Services: map[string]*dynamic.Service{
						"testing-service1-80": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								PassHostHeader: Bool(true),
								Servers: []dynamic.Server{
									{
										URL: "http://10.10.0.1:8080",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc:          "v18 Ingress with multiple ingressClasses",
			serverVersion: "v1.18",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},
					Routers: map[string]*dynamic.Router{
						"testing-foo": {
							Rule:    "PathPrefix(`/foo`)",
							Service: "testing-service1-80",
						},
						"testing-bar": {
							Rule:    "PathPrefix(`/bar`)",
							Service: "testing-service1-80",
						},
					},
					Services: map[string]*dynamic.Service{
						"testing-service1-80": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								PassHostHeader: Bool(true),
								Servers: []dynamic.Server{
									{
										URL: "http://10.10.0.1:8080",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc:          "v18 Ingress with no pathType",
			serverVersion: "v1.18",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},
					Routers: map[string]*dynamic.Router{
						"testing-bar": {
							Rule:    "Path(`/bar`)",
							Service: "testing-service1-80",
						},
					},
					Services: map[string]*dynamic.Service{
						"testing-service1-80": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								PassHostHeader: Bool(true),
								Servers: []dynamic.Server{
									{
										URL: "http://10.10.0.1:8080",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc:          "v18 Ingress with empty pathType",
			serverVersion: "v1.18",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},
					Routers: map[string]*dynamic.Router{
						"testing-bar": {
							Rule:    "Path(`/bar`)",
							Service: "testing-service1-80",
						},
					},
					Services: map[string]*dynamic.Service{
						"testing-service1-80": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								PassHostHeader: Bool(true),
								Servers: []dynamic.Server{
									{
										URL: "http://10.10.0.1:8080",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc:          "v18 Ingress with implementationSpecific pathType",
			serverVersion: "v1.18",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},
					Routers: map[string]*dynamic.Router{
						"testing-bar": {
							Rule:    "Path(`/bar`)",
							Service: "testing-service1-80",
						},
					},
					Services: map[string]*dynamic.Service{
						"testing-service1-80": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								PassHostHeader: Bool(true),
								Servers: []dynamic.Server{
									{
										URL: "http://10.10.0.1:8080",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc:          "v18 Ingress with prefix pathType",
			serverVersion: "v1.18",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},
					Routers: map[string]*dynamic.Router{
						"testing-bar": {
							Rule:    "PathPrefix(`/bar`)",
							Service: "testing-service1-80",
						},
					},
					Services: map[string]*dynamic.Service{
						"testing-service1-80": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								PassHostHeader: Bool(true),
								Servers: []dynamic.Server{
									{
										URL: "http://10.10.0.1:8080",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc:          "v18 Ingress with exact pathType",
			serverVersion: "v1.18",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},
					Routers: map[string]*dynamic.Router{
						"testing-bar": {
							Rule:    "Path(`/bar`)",
							Service: "testing-service1-80",
						},
					},
					Services: map[string]*dynamic.Service{
						"testing-service1-80": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								PassHostHeader: Bool(true),
								Servers: []dynamic.Server{
									{
										URL: "http://10.10.0.1:8080",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc:          "v18 Ingress with missing ingressClass",
			serverVersion: "v1.18",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},
					Routers:     map[string]*dynamic.Router{},
					Services:    map[string]*dynamic.Service{},
				},
			},
		},
		{
			desc:          "v18 Ingress with ingress annotation",
			serverVersion: "v1.18",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},
					Routers: map[string]*dynamic.Router{
						"testing-bar": {
							Rule:    "PathPrefix(`/bar`)",
							Service: "testing-service1-80",
						},
					},
					Services: map[string]*dynamic.Service{
						"testing-service1-80": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								PassHostHeader: Bool(true),
								Servers: []dynamic.Server{
									{
										URL: "http://10.10.0.1:8080",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc:          "v18 Ingress with ingressClasses filter",
			serverVersion: "v1.18",
			ingressClass:  "traefik-lb2",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},
					Routers: map[string]*dynamic.Router{
						"testing-foo": {
							Rule:    "PathPrefix(`/foo`)",
							Service: "testing-service1-80",
						},
					},
					Services: map[string]*dynamic.Service{
						"testing-service1-80": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								PassHostHeader: Bool(true),
								Servers: []dynamic.Server{
									{
										URL: "http://10.10.0.1:8080",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc:          "v19 Ingress with prefix pathType",
			serverVersion: "v1.19",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},
					Routers: map[string]*dynamic.Router{
						"testing-bar": {
							Rule:    "PathPrefix(`/bar`)",
							Service: "testing-service1-80",
						},
					},
					Services: map[string]*dynamic.Service{
						"testing-service1-80": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								PassHostHeader: Bool(true),
								Servers: []dynamic.Server{
									{
										URL: "http://10.10.0.1:8080",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc:          "v19 Ingress with no pathType",
			serverVersion: "v1.19",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},
					Routers: map[string]*dynamic.Router{
						"testing-bar": {
							Rule:    "Path(`/bar`)",
							Service: "testing-service1-80",
						},
					},
					Services: map[string]*dynamic.Service{
						"testing-service1-80": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								PassHostHeader: Bool(true),
								Servers: []dynamic.Server{
									{
										URL: "http://10.10.0.1:8080",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc:          "v19 Ingress with empty pathType",
			serverVersion: "v1.19",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},
					Routers: map[string]*dynamic.Router{
						"testing-bar": {
							Rule:    "Path(`/bar`)",
							Service: "testing-service1-80",
						},
					},
					Services: map[string]*dynamic.Service{
						"testing-service1-80": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								PassHostHeader: Bool(true),
								Servers: []dynamic.Server{
									{
										URL: "http://10.10.0.1:8080",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc:          "v19 Ingress with exact pathType",
			serverVersion: "v1.19",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},
					Routers: map[string]*dynamic.Router{
						"testing-bar": {
							Rule:    "Path(`/bar`)",
							Service: "testing-service1-80",
						},
					},
					Services: map[string]*dynamic.Service{
						"testing-service1-80": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								PassHostHeader: Bool(true),
								Servers: []dynamic.Server{
									{
										URL: "http://10.10.0.1:8080",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc:          "v19 Ingress with implementationSpecific pathType",
			serverVersion: "v1.19",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},
					Routers: map[string]*dynamic.Router{
						"testing-bar": {
							Rule:    "Path(`/bar`)",
							Service: "testing-service1-80",
						},
					},
					Services: map[string]*dynamic.Service{
						"testing-service1-80": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								PassHostHeader: Bool(true),
								Servers: []dynamic.Server{
									{
										URL: "http://10.10.0.1:8080",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc:          "v19 Ingress with ingress annotation",
			serverVersion: "v1.19",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},
					Routers: map[string]*dynamic.Router{
						"testing-bar": {
							Rule:    "PathPrefix(`/bar`)",
							Service: "testing-service1-80",
						},
					},
					Services: map[string]*dynamic.Service{
						"testing-service1-80": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								PassHostHeader: Bool(true),
								Servers: []dynamic.Server{
									{
										URL: "http://10.10.0.1:8080",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc:          "v19 Ingress with ingressClass",
			serverVersion: "v1.19",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},
					Routers: map[string]*dynamic.Router{
						"testing-bar": {
							Rule:    "PathPrefix(`/bar`)",
							Service: "testing-service1-80",
						},
					},
					Services: map[string]*dynamic.Service{
						"testing-service1-80": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								PassHostHeader: Bool(true),
								Servers: []dynamic.Server{
									{
										URL: "http://10.10.0.1:8080",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc:          "v19 Ingress with ingressClassv1",
			serverVersion: "v1.19",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},
					Routers: map[string]*dynamic.Router{
						"testing-bar": {
							Rule:    "PathPrefix(`/bar`)",
							Service: "testing-service1-80",
						},
					},
					Services: map[string]*dynamic.Service{
						"testing-service1-80": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								PassHostHeader: Bool(true),
								Servers: []dynamic.Server{
									{
										URL: "http://10.10.0.1:8080",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc:          "v19 Ingress with named port",
			serverVersion: "v1.19",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},
					Routers: map[string]*dynamic.Router{
						"testing-bar": {
							Rule:    "PathPrefix(`/bar`)",
							Service: "testing-service1-foobar",
						},
					},
					Services: map[string]*dynamic.Service{
						"testing-service1-foobar": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								PassHostHeader: Bool(true),
								Servers: []dynamic.Server{
									{
										URL: "http://10.10.0.1:4711",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc:          "v19 Ingress with missing ingressClass",
			serverVersion: "v1.19",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},
					Routers:     map[string]*dynamic.Router{},
					Services:    map[string]*dynamic.Service{},
				},
			},
		},
		{
			desc:          "v19 Ingress with defaultbackend",
			serverVersion: "v1.19",
			expected: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{},
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{},
					Routers: map[string]*dynamic.Router{
						"default-router": {
							Rule:     "PathPrefix(`/`)",
							Priority: math.MinInt32,
							Service:  "default-backend",
						},
					},
					Services: map[string]*dynamic.Service{
						"default-backend": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								PassHostHeader: Bool(true),
								Servers: []dynamic.Server{
									{
										URL: "http://10.10.0.1:8080",
									},
								},
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

			var paths []string
			_, err := os.Stat(generateTestFilename("_ingress", test.desc))
			if err == nil {
				paths = append(paths, generateTestFilename("_ingress", test.desc))
			}
			_, err = os.Stat(generateTestFilename("_endpoint", test.desc))
			if err == nil {
				paths = append(paths, generateTestFilename("_endpoint", test.desc))
			}
			_, err = os.Stat(generateTestFilename("_service", test.desc))
			if err == nil {
				paths = append(paths, generateTestFilename("_service", test.desc))
			}
			_, err = os.Stat(generateTestFilename("_secret", test.desc))
			if err == nil {
				paths = append(paths, generateTestFilename("_secret", test.desc))
			}
			_, err = os.Stat(generateTestFilename("_ingressclass", test.desc))
			if err == nil {
				paths = append(paths, generateTestFilename("_ingressclass", test.desc))
			}

			serverVersion := test.serverVersion
			if serverVersion == "" {
				serverVersion = "v1.17"
			}

			clientMock := newClientMock(serverVersion, paths...)

			p := Provider{IngressClass: test.ingressClass, AllowEmptyServices: test.allowEmptyServices}
			conf := p.loadConfigurationFromIngresses(context.Background(), clientMock)

			assert.Equal(t, test.expected, conf)
		})
	}
}

func generateTestFilename(suffix, desc string) string {
	return "./fixtures/" + strings.ReplaceAll(desc, " ", "-") + suffix + ".yml"
}

func TestGetCertificates(t *testing.T) {
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
		ingress   *networkingv1.Ingress
		client    Client
		result    map[string]*tls.CertAndStores
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
							"tls.crt": []byte("tls-crt"),
							"tls.key": []byte("tls-key"),
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-secret",
							Namespace: "testing",
						},
						Data: map[string][]byte{
							"tls.crt": []byte("tls-crt"),
							"tls.key": []byte("tls-key"),
						},
					},
				},
			},
			result: map[string]*tls.CertAndStores{
				"testing-test-secret": {
					Certificate: tls.Certificate{
						CertFile: tls.FileOrContent("tls-crt"),
						KeyFile:  tls.FileOrContent("tls-key"),
					},
				},
				"testing-test-secret2": {
					Certificate: tls.Certificate{
						CertFile: tls.FileOrContent("tls-crt"),
						KeyFile:  tls.FileOrContent("tls-key"),
					},
				},
			},
		},
		{
			desc:    "return nil when no secret is defined",
			ingress: testIngressWithoutSecret,
			client:  clientMock{},
			result:  map[string]*tls.CertAndStores{},
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			tlsConfigs := map[string]*tls.CertAndStores{}
			err := getCertificates(context.Background(), test.ingress, test.client, tlsConfigs)

			if test.errResult != "" {
				assert.EqualError(t, err, test.errResult)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.result, tlsConfigs)
			}
		})
	}
}
