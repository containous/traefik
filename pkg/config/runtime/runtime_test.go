package runtime_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/config/runtime"
)

// all the Routers/Middlewares/Services are considered fully qualified.
func TestPopulateUsedBy(t *testing.T) {
	testCases := []struct {
		desc     string
		conf     *runtime.Configuration
		expected runtime.Configuration
	}{
		{
			desc:     "nil config",
			conf:     nil,
			expected: runtime.Configuration{},
		},
		{
			desc: "One service used by two routers",
			conf: &runtime.Configuration{
				Routers: map[string]*runtime.RouterInfo{
					"foo@myprovider": {
						Router: &dynamic.Router{
							EntryPoints: []string{"web"},
							Service:     "foo-service@myprovider",
							Rule:        "Host(`bar.foo`)",
						},
					},
					"bar@myprovider": {
						Router: &dynamic.Router{
							EntryPoints: []string{"web"},
							Service:     "foo-service@myprovider",
							Rule:        "Host(`foo.bar`)",
						},
					},
				},
				Services: map[string]*runtime.ServiceInfo{
					"foo-service@myprovider": {
						Service: &dynamic.Service{
							LoadBalancer: &dynamic.ServersLoadBalancer{
								Servers: []dynamic.Server{
									{URL: "http://127.0.0.1:8085"},
									{URL: "http://127.0.0.1:8086"},
								},
								HealthCheck: &dynamic.ServerHealthCheck{
									Interval: "500ms",
									Path:     "/health",
								},
							},
						},
					},
				},
			},
			expected: runtime.Configuration{
				Routers: map[string]*runtime.RouterInfo{
					"foo@myprovider": {},
					"bar@myprovider": {},
				},
				Services: map[string]*runtime.ServiceInfo{
					"foo-service@myprovider": {
						UsedBy: []string{"bar@myprovider", "foo@myprovider"},
					},
				},
			},
		},
		{
			desc: "One service used by two routers, but one router with wrong rule",
			conf: &runtime.Configuration{
				Services: map[string]*runtime.ServiceInfo{
					"foo-service@myprovider": {
						Service: &dynamic.Service{
							LoadBalancer: &dynamic.ServersLoadBalancer{
								Servers: []dynamic.Server{
									{URL: "http://127.0.0.1"},
								},
							},
						},
					},
				},
				Routers: map[string]*runtime.RouterInfo{
					"foo@myprovider": {
						Router: &dynamic.Router{
							EntryPoints: []string{"web"},
							Service:     "foo-service@myprovider",
							Rule:        "WrongRule(`bar.foo`)",
						},
					},
					"bar@myprovider": {
						Router: &dynamic.Router{
							EntryPoints: []string{"web"},
							Service:     "foo-service@myprovider",
							Rule:        "Host(`foo.bar`)",
						},
					},
				},
			},
			expected: runtime.Configuration{
				Routers: map[string]*runtime.RouterInfo{
					"foo@myprovider": {},
					"bar@myprovider": {},
				},
				Services: map[string]*runtime.ServiceInfo{
					"foo-service@myprovider": {
						UsedBy: []string{"bar@myprovider", "foo@myprovider"},
					},
				},
			},
		},
		{
			desc: "Broken Service used by one Router",
			conf: &runtime.Configuration{
				Services: map[string]*runtime.ServiceInfo{
					"foo-service@myprovider": {
						Service: &dynamic.Service{
							LoadBalancer: nil,
						},
					},
				},
				Routers: map[string]*runtime.RouterInfo{
					"bar@myprovider": {
						Router: &dynamic.Router{
							EntryPoints: []string{"web"},
							Service:     "foo-service@myprovider",
							Rule:        "Host(`foo.bar`)",
						},
					},
				},
			},
			expected: runtime.Configuration{
				Routers: map[string]*runtime.RouterInfo{
					"bar@myprovider": {},
				},
				Services: map[string]*runtime.ServiceInfo{
					"foo-service@myprovider": {
						UsedBy: []string{"bar@myprovider"},
					},
				},
			},
		},
		{
			desc: "2 different Services each used by a disctinct router.",
			conf: &runtime.Configuration{
				Services: map[string]*runtime.ServiceInfo{
					"foo-service@myprovider": {
						Service: &dynamic.Service{
							LoadBalancer: &dynamic.ServersLoadBalancer{
								Servers: []dynamic.Server{
									{
										URL: "http://127.0.0.1:8085",
									},
									{
										URL: "http://127.0.0.1:8086",
									},
								},
								HealthCheck: &dynamic.ServerHealthCheck{
									Interval: "500ms",
									Path:     "/health",
								},
							},
						},
					},
					"bar-service@myprovider": {
						Service: &dynamic.Service{
							LoadBalancer: &dynamic.ServersLoadBalancer{
								Servers: []dynamic.Server{
									{
										URL: "http://127.0.0.1:8087",
									},
									{
										URL: "http://127.0.0.1:8088",
									},
								},
								HealthCheck: &dynamic.ServerHealthCheck{
									Interval: "500ms",
									Path:     "/health",
								},
							},
						},
					},
				},
				Routers: map[string]*runtime.RouterInfo{
					"foo@myprovider": {
						Router: &dynamic.Router{
							EntryPoints: []string{"web"},
							Service:     "foo-service@myprovider",
							Rule:        "Host(`bar.foo`)",
						},
					},
					"bar@myprovider": {
						Router: &dynamic.Router{
							EntryPoints: []string{"web"},
							Service:     "bar-service@myprovider",
							Rule:        "Host(`foo.bar`)",
						},
					},
				},
			},
			expected: runtime.Configuration{
				Routers: map[string]*runtime.RouterInfo{
					"bar@myprovider": {},
					"foo@myprovider": {},
				},
				Services: map[string]*runtime.ServiceInfo{
					"foo-service@myprovider": {
						UsedBy: []string{"foo@myprovider"},
					},
					"bar-service@myprovider": {
						UsedBy: []string{"bar@myprovider"},
					},
				},
			},
		},
		{
			desc: "2 middlewares both used by 2 Routers",
			conf: &runtime.Configuration{
				Services: map[string]*runtime.ServiceInfo{
					"foo-service@myprovider": {
						Service: &dynamic.Service{
							LoadBalancer: &dynamic.ServersLoadBalancer{
								Servers: []dynamic.Server{
									{
										URL: "http://127.0.0.1",
									},
								},
							},
						},
					},
				},
				Middlewares: map[string]*runtime.MiddlewareInfo{
					"auth@myprovider": {
						Middleware: &dynamic.Middleware{
							BasicAuth: &dynamic.BasicAuth{
								Users: []string{"admin:admin"},
							},
						},
					},
					"addPrefixTest@myprovider": {
						Middleware: &dynamic.Middleware{
							AddPrefix: &dynamic.AddPrefix{
								Prefix: "/toto",
							},
						},
					},
				},
				Routers: map[string]*runtime.RouterInfo{
					"bar@myprovider": {
						Router: &dynamic.Router{
							EntryPoints: []string{"web"},
							Service:     "foo-service@myprovider",
							Rule:        "Host(`foo.bar`)",
							Middlewares: []string{"auth", "addPrefixTest"},
						},
					},
					"test@myprovider": {
						Router: &dynamic.Router{
							EntryPoints: []string{"web"},
							Service:     "foo-service@myprovider",
							Rule:        "Host(`foo.bar.other`)",
							Middlewares: []string{"addPrefixTest", "auth"},
						},
					},
				},
			},
			expected: runtime.Configuration{
				Routers: map[string]*runtime.RouterInfo{
					"bar@myprovider":  {},
					"test@myprovider": {},
				},
				Services: map[string]*runtime.ServiceInfo{
					"foo-service@myprovider": {
						UsedBy: []string{"bar@myprovider", "test@myprovider"},
					},
				},
				Middlewares: map[string]*runtime.MiddlewareInfo{
					"auth@myprovider": {
						UsedBy: []string{"bar@myprovider", "test@myprovider"},
					},
					"addPrefixTest@myprovider": {
						UsedBy: []string{"bar@myprovider", "test@myprovider"},
					},
				},
			},
		},
		{
			desc: "Unknown middleware is not used by the Router",
			conf: &runtime.Configuration{
				Services: map[string]*runtime.ServiceInfo{
					"foo-service@myprovider": {
						Service: &dynamic.Service{
							LoadBalancer: &dynamic.ServersLoadBalancer{
								Servers: []dynamic.Server{
									{
										URL: "http://127.0.0.1",
									},
								},
							},
						},
					},
				},
				Middlewares: map[string]*runtime.MiddlewareInfo{
					"auth@myprovider": {
						Middleware: &dynamic.Middleware{
							BasicAuth: &dynamic.BasicAuth{
								Users: []string{"admin:admin"},
							},
						},
					},
				},
				Routers: map[string]*runtime.RouterInfo{
					"bar@myprovider": {
						Router: &dynamic.Router{
							EntryPoints: []string{"web"},
							Service:     "foo-service@myprovider",
							Rule:        "Host(`foo.bar`)",
							Middlewares: []string{"unknown"},
						},
					},
				},
			},
			expected: runtime.Configuration{
				Services: map[string]*runtime.ServiceInfo{
					"foo-service@myprovider": {
						UsedBy: []string{"bar@myprovider"},
					},
				},
			},
		},
		{
			desc: "Broken middleware is used by Router",
			conf: &runtime.Configuration{
				Services: map[string]*runtime.ServiceInfo{
					"foo-service@myprovider": {
						Service: &dynamic.Service{
							LoadBalancer: &dynamic.ServersLoadBalancer{
								Servers: []dynamic.Server{
									{
										URL: "http://127.0.0.1",
									},
								},
							},
						},
					},
				},
				Middlewares: map[string]*runtime.MiddlewareInfo{
					"auth@myprovider": {
						Middleware: &dynamic.Middleware{
							BasicAuth: &dynamic.BasicAuth{
								Users: []string{"badConf"},
							},
						},
					},
				},
				Routers: map[string]*runtime.RouterInfo{
					"bar@myprovider": {
						Router: &dynamic.Router{
							EntryPoints: []string{"web"},
							Service:     "foo-service@myprovider",
							Rule:        "Host(`foo.bar`)",
							Middlewares: []string{"auth@myprovider"},
						},
					},
				},
			},
			expected: runtime.Configuration{
				Routers: map[string]*runtime.RouterInfo{
					"bar@myprovider": {},
				},
				Services: map[string]*runtime.ServiceInfo{
					"foo-service@myprovider": {
						UsedBy: []string{"bar@myprovider"},
					},
				},
				Middlewares: map[string]*runtime.MiddlewareInfo{
					"auth@myprovider": {
						UsedBy: []string{"bar@myprovider"},
					},
				},
			},
		},
		{
			desc: "2 middlewares from 2 disctinct providers both used by 2 Routers",
			conf: &runtime.Configuration{
				Services: map[string]*runtime.ServiceInfo{
					"foo-service@myprovider": {
						Service: &dynamic.Service{
							LoadBalancer: &dynamic.ServersLoadBalancer{
								Servers: []dynamic.Server{
									{
										URL: "http://127.0.0.1",
									},
								},
							},
						},
					},
				},
				Middlewares: map[string]*runtime.MiddlewareInfo{
					"auth@myprovider": {
						Middleware: &dynamic.Middleware{
							BasicAuth: &dynamic.BasicAuth{
								Users: []string{"admin:admin"},
							},
						},
					},
					"addPrefixTest@myprovider": {
						Middleware: &dynamic.Middleware{
							AddPrefix: &dynamic.AddPrefix{
								Prefix: "/titi",
							},
						},
					},
					"addPrefixTest@anotherprovider": {
						Middleware: &dynamic.Middleware{
							AddPrefix: &dynamic.AddPrefix{
								Prefix: "/toto",
							},
						},
					},
				},
				Routers: map[string]*runtime.RouterInfo{
					"bar@myprovider": {
						Router: &dynamic.Router{
							EntryPoints: []string{"web"},
							Service:     "foo-service@myprovider",
							Rule:        "Host(`foo.bar`)",
							Middlewares: []string{"auth", "addPrefixTest@anotherprovider"},
						},
					},
					"test@myprovider": {
						Router: &dynamic.Router{
							EntryPoints: []string{"web"},
							Service:     "foo-service@myprovider",
							Rule:        "Host(`foo.bar.other`)",
							Middlewares: []string{"addPrefixTest", "auth"},
						},
					},
				},
			},
			expected: runtime.Configuration{
				Routers: map[string]*runtime.RouterInfo{
					"bar@myprovider":  {},
					"test@myprovider": {},
				},
				Services: map[string]*runtime.ServiceInfo{
					"foo-service@myprovider": {
						UsedBy: []string{"bar@myprovider", "test@myprovider"},
					},
				},
				Middlewares: map[string]*runtime.MiddlewareInfo{
					"auth@myprovider": {
						UsedBy: []string{"bar@myprovider", "test@myprovider"},
					},
					"addPrefixTest@myprovider": {
						UsedBy: []string{"test@myprovider"},
					},
					"addPrefixTest@anotherprovider": {
						UsedBy: []string{"bar@myprovider"},
					},
				},
			},
		},

		// TCP tests from hereon
		{
			desc: "TCP, One service used by two routers",
			conf: &runtime.Configuration{
				TCPRouters: map[string]*runtime.TCPRouterInfo{
					"foo@myprovider": {
						TCPRouter: &dynamic.TCPRouter{
							EntryPoints: []string{"web"},
							Service:     "foo-service@myprovider",
							Rule:        "Host(`bar.foo`)",
						},
					},
					"bar@myprovider": {
						TCPRouter: &dynamic.TCPRouter{
							EntryPoints: []string{"web"},
							Service:     "foo-service@myprovider",
							Rule:        "Host(`foo.bar`)",
						},
					},
				},
				TCPServices: map[string]*runtime.TCPServiceInfo{
					"foo-service@myprovider": {
						TCPService: &dynamic.TCPService{
							LoadBalancer: &dynamic.TCPServersLoadBalancer{
								Servers: []dynamic.TCPServer{
									{
										Address: "127.0.0.1",
										Port:    "8085",
									},
									{
										Address: "127.0.0.1",
										Port:    "8086",
									},
								},
							},
						},
					},
				},
			},
			expected: runtime.Configuration{
				TCPRouters: map[string]*runtime.TCPRouterInfo{
					"foo@myprovider": {},
					"bar@myprovider": {},
				},
				TCPServices: map[string]*runtime.TCPServiceInfo{
					"foo-service@myprovider": {
						UsedBy: []string{"bar@myprovider", "foo@myprovider"},
					},
				},
			},
		},
		{
			desc: "TCP, One service used by two routers, but one router with wrong rule",
			conf: &runtime.Configuration{
				TCPServices: map[string]*runtime.TCPServiceInfo{
					"foo-service@myprovider": {
						TCPService: &dynamic.TCPService{
							LoadBalancer: &dynamic.TCPServersLoadBalancer{
								Servers: []dynamic.TCPServer{
									{
										Address: "127.0.0.1",
									},
								},
							},
						},
					},
				},
				TCPRouters: map[string]*runtime.TCPRouterInfo{
					"foo@myprovider": {
						TCPRouter: &dynamic.TCPRouter{
							EntryPoints: []string{"web"},
							Service:     "foo-service@myprovider",
							Rule:        "WrongRule(`bar.foo`)",
						},
					},
					"bar@myprovider": {
						TCPRouter: &dynamic.TCPRouter{
							EntryPoints: []string{"web"},
							Service:     "foo-service@myprovider",
							Rule:        "Host(`foo.bar`)",
						},
					},
				},
			},
			expected: runtime.Configuration{
				TCPRouters: map[string]*runtime.TCPRouterInfo{
					"foo@myprovider": {},
					"bar@myprovider": {},
				},
				TCPServices: map[string]*runtime.TCPServiceInfo{
					"foo-service@myprovider": {
						UsedBy: []string{"bar@myprovider", "foo@myprovider"},
					},
				},
			},
		},
		{
			desc: "TCP, Broken Service used by one Router",
			conf: &runtime.Configuration{
				TCPServices: map[string]*runtime.TCPServiceInfo{
					"foo-service@myprovider": {
						TCPService: &dynamic.TCPService{
							LoadBalancer: nil,
						},
					},
				},
				TCPRouters: map[string]*runtime.TCPRouterInfo{
					"bar@myprovider": {
						TCPRouter: &dynamic.TCPRouter{
							EntryPoints: []string{"web"},
							Service:     "foo-service@myprovider",
							Rule:        "Host(`foo.bar`)",
						},
					},
				},
			},
			expected: runtime.Configuration{
				TCPRouters: map[string]*runtime.TCPRouterInfo{
					"bar@myprovider": {},
				},
				TCPServices: map[string]*runtime.TCPServiceInfo{
					"foo-service@myprovider": {
						UsedBy: []string{"bar@myprovider"},
					},
				},
			},
		},
		{
			desc: "TCP, 2 different Services each used by a disctinct router.",
			conf: &runtime.Configuration{
				TCPServices: map[string]*runtime.TCPServiceInfo{
					"foo-service@myprovider": {
						TCPService: &dynamic.TCPService{
							LoadBalancer: &dynamic.TCPServersLoadBalancer{
								Servers: []dynamic.TCPServer{
									{
										Address: "127.0.0.1",
										Port:    "8085",
									},
									{
										Address: "127.0.0.1",
										Port:    "8086",
									},
								},
							},
						},
					},
					"bar-service@myprovider": {
						TCPService: &dynamic.TCPService{
							LoadBalancer: &dynamic.TCPServersLoadBalancer{
								Servers: []dynamic.TCPServer{
									{
										Address: "127.0.0.1",
										Port:    "8087",
									},
									{
										Address: "127.0.0.1",
										Port:    "8088",
									},
								},
							},
						},
					},
				},
				TCPRouters: map[string]*runtime.TCPRouterInfo{
					"foo@myprovider": {
						TCPRouter: &dynamic.TCPRouter{
							EntryPoints: []string{"web"},
							Service:     "foo-service@myprovider",
							Rule:        "Host(`bar.foo`)",
						},
					},
					"bar@myprovider": {
						TCPRouter: &dynamic.TCPRouter{
							EntryPoints: []string{"web"},
							Service:     "bar-service@myprovider",
							Rule:        "Host(`foo.bar`)",
						},
					},
				},
			},
			expected: runtime.Configuration{
				TCPRouters: map[string]*runtime.TCPRouterInfo{
					"bar@myprovider": {},
					"foo@myprovider": {},
				},
				TCPServices: map[string]*runtime.TCPServiceInfo{
					"foo-service@myprovider": {
						UsedBy: []string{"foo@myprovider"},
					},
					"bar-service@myprovider": {
						UsedBy: []string{"bar@myprovider"},
					},
				},
			},
		},
	}
	for _, test := range testCases {
		test := test

		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			runtimeConf := test.conf
			runtimeConf.PopulateUsedBy()

			for key, expectedService := range test.expected.Services {
				require.NotNil(t, runtimeConf.Services[key])
				assert.Equal(t, expectedService.UsedBy, runtimeConf.Services[key].UsedBy)
			}

			for key, expectedMiddleware := range test.expected.Middlewares {
				require.NotNil(t, runtimeConf.Middlewares[key])
				assert.Equal(t, expectedMiddleware.UsedBy, runtimeConf.Middlewares[key].UsedBy)
			}

			for key, expectedTCPService := range test.expected.TCPServices {
				require.NotNil(t, runtimeConf.TCPServices[key])
				assert.Equal(t, expectedTCPService.UsedBy, runtimeConf.TCPServices[key].UsedBy)
			}
		})
	}
}
