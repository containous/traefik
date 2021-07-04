package router

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/config/runtime"
	"github.com/traefik/traefik/v2/pkg/config/static"
	"github.com/traefik/traefik/v2/pkg/metrics"
	"github.com/traefik/traefik/v2/pkg/middlewares/accesslog"
	"github.com/traefik/traefik/v2/pkg/middlewares/requestdecorator"
	"github.com/traefik/traefik/v2/pkg/server/middleware"
	"github.com/traefik/traefik/v2/pkg/server/service"
	"github.com/traefik/traefik/v2/pkg/testhelpers"
	"github.com/traefik/traefik/v2/pkg/types"
)

func TestRouterManager_Get(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	t.Cleanup(func() { server.Close() })

	type expectedResult struct {
		StatusCode     int
		RequestHeaders map[string]string
	}

	testCases := []struct {
		desc              string
		routersConfig     map[string]*dynamic.Router
		serviceConfig     map[string]*dynamic.Service
		middlewaresConfig map[string]*dynamic.Middleware
		entryPoints       []string
		expected          expectedResult
	}{
		{
			desc: "no middleware",
			routersConfig: map[string]*dynamic.Router{
				"foo": {
					EntryPoints: []string{"web"},
					Service:     "foo-service",
					Rule:        "Host(`foo.bar`)",
				},
			},
			serviceConfig: map[string]*dynamic.Service{
				"foo-service": {
					LoadBalancer: &dynamic.ServersLoadBalancer{
						Servers: []dynamic.Server{
							{
								URL: server.URL,
							},
						},
					},
				},
			},
			entryPoints: []string{"web"},
			expected:    expectedResult{StatusCode: http.StatusOK},
		},
		{
			desc: "empty host",
			routersConfig: map[string]*dynamic.Router{
				"foo": {
					EntryPoints: []string{"web"},
					Service:     "foo-service",
					Rule:        "Host(``)",
				},
			},
			serviceConfig: map[string]*dynamic.Service{
				"foo-service": {
					LoadBalancer: &dynamic.ServersLoadBalancer{
						Servers: []dynamic.Server{
							{
								URL: server.URL,
							},
						},
					},
				},
			},
			entryPoints: []string{"web"},
			expected:    expectedResult{StatusCode: http.StatusNotFound},
		},
		{
			desc: "no load balancer",
			routersConfig: map[string]*dynamic.Router{
				"foo": {
					EntryPoints: []string{"web"},
					Service:     "foo-service",
					Rule:        "Host(`foo.bar`)",
				},
			},
			serviceConfig: map[string]*dynamic.Service{
				"foo-service": {},
			},
			entryPoints: []string{"web"},
			expected:    expectedResult{StatusCode: http.StatusNotFound},
		},
		{
			desc: "no middleware, no matching",
			routersConfig: map[string]*dynamic.Router{
				"foo": {
					EntryPoints: []string{"web"},
					Service:     "foo-service",
					Rule:        "Host(`bar.bar`)",
				},
			},
			serviceConfig: map[string]*dynamic.Service{
				"foo-service": {
					LoadBalancer: &dynamic.ServersLoadBalancer{
						Servers: []dynamic.Server{
							{
								URL: server.URL,
							},
						},
					},
				},
			},
			entryPoints: []string{"web"},
			expected:    expectedResult{StatusCode: http.StatusNotFound},
		},
		{
			desc: "middleware: headers > auth",
			routersConfig: map[string]*dynamic.Router{
				"foo": {
					EntryPoints: []string{"web"},
					Middlewares: []string{"headers-middle", "auth-middle"},
					Service:     "foo-service",
					Rule:        "Host(`foo.bar`)",
				},
			},
			serviceConfig: map[string]*dynamic.Service{
				"foo-service": {
					LoadBalancer: &dynamic.ServersLoadBalancer{
						Servers: []dynamic.Server{
							{
								URL: server.URL,
							},
						},
					},
				},
			},
			middlewaresConfig: map[string]*dynamic.Middleware{
				"auth-middle": {
					BasicAuth: &dynamic.BasicAuth{
						Users: []string{"toto:titi"},
					},
				},
				"headers-middle": {
					Headers: &dynamic.Headers{
						CustomRequestHeaders: map[string]string{"X-Apero": "beer"},
					},
				},
			},
			entryPoints: []string{"web"},
			expected: expectedResult{
				StatusCode: http.StatusUnauthorized,
				RequestHeaders: map[string]string{
					"X-Apero": "beer",
				},
			},
		},
		{
			desc: "middleware: auth > header",
			routersConfig: map[string]*dynamic.Router{
				"foo": {
					EntryPoints: []string{"web"},
					Middlewares: []string{"auth-middle", "headers-middle"},
					Service:     "foo-service",
					Rule:        "Host(`foo.bar`)",
				},
			},
			serviceConfig: map[string]*dynamic.Service{
				"foo-service": {
					LoadBalancer: &dynamic.ServersLoadBalancer{
						Servers: []dynamic.Server{
							{
								URL: server.URL,
							},
						},
					},
				},
			},
			middlewaresConfig: map[string]*dynamic.Middleware{
				"auth-middle": {
					BasicAuth: &dynamic.BasicAuth{
						Users: []string{"toto:titi"},
					},
				},
				"headers-middle": {
					Headers: &dynamic.Headers{
						CustomRequestHeaders: map[string]string{"X-Apero": "beer"},
					},
				},
			},
			entryPoints: []string{"web"},
			expected: expectedResult{
				StatusCode: http.StatusUnauthorized,
				RequestHeaders: map[string]string{
					"X-Apero": "",
				},
			},
		},
		{
			desc: "no middleware with provider name",
			routersConfig: map[string]*dynamic.Router{
				"foo@provider-1": {
					EntryPoints: []string{"web"},
					Service:     "foo-service",
					Rule:        "Host(`foo.bar`)",
				},
			},
			serviceConfig: map[string]*dynamic.Service{
				"foo-service@provider-1": {
					LoadBalancer: &dynamic.ServersLoadBalancer{
						Servers: []dynamic.Server{
							{
								URL: server.URL,
							},
						},
					},
				},
			},
			entryPoints: []string{"web"},
			expected:    expectedResult{StatusCode: http.StatusOK},
		},
		{
			desc: "no middleware with specified provider name",
			routersConfig: map[string]*dynamic.Router{
				"foo@provider-1": {
					EntryPoints: []string{"web"},
					Service:     "foo-service@provider-2",
					Rule:        "Host(`foo.bar`)",
				},
			},
			serviceConfig: map[string]*dynamic.Service{
				"foo-service@provider-2": {
					LoadBalancer: &dynamic.ServersLoadBalancer{
						Servers: []dynamic.Server{
							{
								URL: server.URL,
							},
						},
					},
				},
			},
			entryPoints: []string{"web"},
			expected:    expectedResult{StatusCode: http.StatusOK},
		},
		{
			desc: "middleware: chain with provider name",
			routersConfig: map[string]*dynamic.Router{
				"foo@provider-1": {
					EntryPoints: []string{"web"},
					Middlewares: []string{"chain-middle@provider-2", "headers-middle"},
					Service:     "foo-service",
					Rule:        "Host(`foo.bar`)",
				},
			},
			serviceConfig: map[string]*dynamic.Service{
				"foo-service@provider-1": {
					LoadBalancer: &dynamic.ServersLoadBalancer{
						Servers: []dynamic.Server{
							{
								URL: server.URL,
							},
						},
					},
				},
			},
			middlewaresConfig: map[string]*dynamic.Middleware{
				"chain-middle@provider-2": {
					Chain: &dynamic.Chain{Middlewares: []string{"auth-middle"}},
				},
				"auth-middle@provider-2": {
					BasicAuth: &dynamic.BasicAuth{
						Users: []string{"toto:titi"},
					},
				},
				"headers-middle@provider-1": {
					Headers: &dynamic.Headers{
						CustomRequestHeaders: map[string]string{"X-Apero": "beer"},
					},
				},
			},
			entryPoints: []string{"web"},
			expected: expectedResult{
				StatusCode: http.StatusUnauthorized,
				RequestHeaders: map[string]string{
					"X-Apero": "",
				},
			},
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			rtConf := runtime.NewConfig(dynamic.Configuration{
				HTTP: &dynamic.HTTPConfiguration{
					Services:    test.serviceConfig,
					Routers:     test.routersConfig,
					Middlewares: test.middlewaresConfig,
				},
			})

			roundTripperManager := service.NewRoundTripperManager()
			roundTripperManager.Update(map[string]*dynamic.ServersTransport{"default@internal": {}})
			serviceManager := service.NewManager(rtConf.Services, nil, nil, roundTripperManager)
			middlewaresBuilder := middleware.NewBuilder(rtConf.Middlewares, serviceManager, nil)
			chainBuilder := middleware.NewChainBuilder(static.Configuration{}, nil, nil)

			routerManager := NewManager(rtConf, serviceManager, middlewaresBuilder, chainBuilder, metrics.NewVoidRegistry())

			handlers := routerManager.BuildHandlers(context.Background(), test.entryPoints, false)

			w := httptest.NewRecorder()
			req := testhelpers.MustNewRequest(http.MethodGet, "http://foo.bar/", nil)

			reqHost := requestdecorator.New(nil)
			reqHost.ServeHTTP(w, req, handlers["web"].ServeHTTP)

			assert.Equal(t, test.expected.StatusCode, w.Code)

			for key, value := range test.expected.RequestHeaders {
				assert.Equal(t, value, req.Header.Get(key))
			}
		})
	}
}

func TestAccessLog(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	t.Cleanup(func() { server.Close() })

	testCases := []struct {
		desc              string
		routersConfig     map[string]*dynamic.Router
		serviceConfig     map[string]*dynamic.Service
		middlewaresConfig map[string]*dynamic.Middleware
		entryPoints       []string
		expected          string
	}{
		{
			desc: "apply routerName in accesslog (first match)",
			routersConfig: map[string]*dynamic.Router{
				"foo": {
					EntryPoints: []string{"web"},
					Service:     "foo-service",
					Rule:        "Host(`foo.bar`)",
				},
				"bar": {
					EntryPoints: []string{"web"},
					Service:     "foo-service",
					Rule:        "Host(`bar.foo`)",
				},
			},
			serviceConfig: map[string]*dynamic.Service{
				"foo-service": {
					LoadBalancer: &dynamic.ServersLoadBalancer{
						Servers: []dynamic.Server{
							{
								URL: server.URL,
							},
						},
					},
				},
			},
			entryPoints: []string{"web"},
			expected:    "foo",
		},
		{
			desc: "apply routerName in accesslog (second match)",
			routersConfig: map[string]*dynamic.Router{
				"foo": {
					EntryPoints: []string{"web"},
					Service:     "foo-service",
					Rule:        "Host(`bar.foo`)",
				},
				"bar": {
					EntryPoints: []string{"web"},
					Service:     "foo-service",
					Rule:        "Host(`foo.bar`)",
				},
			},
			serviceConfig: map[string]*dynamic.Service{
				"foo-service": {
					LoadBalancer: &dynamic.ServersLoadBalancer{
						Servers: []dynamic.Server{
							{
								URL: server.URL,
							},
						},
					},
				},
			},
			entryPoints: []string{"web"},
			expected:    "bar",
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			rtConf := runtime.NewConfig(dynamic.Configuration{
				HTTP: &dynamic.HTTPConfiguration{
					Services:    test.serviceConfig,
					Routers:     test.routersConfig,
					Middlewares: test.middlewaresConfig,
				},
			})

			roundTripperManager := service.NewRoundTripperManager()
			roundTripperManager.Update(map[string]*dynamic.ServersTransport{"default@internal": {}})
			serviceManager := service.NewManager(rtConf.Services, nil, nil, roundTripperManager)
			middlewaresBuilder := middleware.NewBuilder(rtConf.Middlewares, serviceManager, nil)
			chainBuilder := middleware.NewChainBuilder(static.Configuration{}, nil, nil)

			routerManager := NewManager(rtConf, serviceManager, middlewaresBuilder, chainBuilder, metrics.NewVoidRegistry())

			handlers := routerManager.BuildHandlers(context.Background(), test.entryPoints, false)

			w := httptest.NewRecorder()
			req := testhelpers.MustNewRequest(http.MethodGet, "http://foo.bar/", nil)

			accesslogger, err := accesslog.NewHandler(&types.AccessLog{
				Format: "json",
			})
			require.NoError(t, err)

			reqHost := requestdecorator.New(nil)

			accesslogger.ServeHTTP(w, req, http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				reqHost.ServeHTTP(w, req, handlers["web"].ServeHTTP)

				data := accesslog.GetLogData(req)
				require.NotNil(t, data)

				assert.Equal(t, test.expected, data.Core[accesslog.RouterName])
			}))
		})
	}
}

func TestRuntimeConfiguration(t *testing.T) {
	testCases := []struct {
		desc             string
		serviceConfig    map[string]*dynamic.Service
		routerConfig     map[string]*dynamic.Router
		middlewareConfig map[string]*dynamic.Middleware
		expectedError    int
	}{
		{
			desc: "No error",
			serviceConfig: map[string]*dynamic.Service{
				"foo-service": {
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
			routerConfig: map[string]*dynamic.Router{
				"foo": {
					EntryPoints: []string{"web"},
					Service:     "foo-service",
					Rule:        "Host(`bar.foo`)",
				},
				"bar": {
					EntryPoints: []string{"web"},
					Service:     "foo-service",
					Rule:        "Host(`foo.bar`)",
				},
			},
			expectedError: 0,
		},
		{
			desc: "One router with wrong rule",
			serviceConfig: map[string]*dynamic.Service{
				"foo-service": {
					LoadBalancer: &dynamic.ServersLoadBalancer{
						Servers: []dynamic.Server{
							{
								URL: "http://127.0.0.1",
							},
						},
					},
				},
			},
			routerConfig: map[string]*dynamic.Router{
				"foo": {
					EntryPoints: []string{"web"},
					Service:     "foo-service",
					Rule:        "WrongRule(`bar.foo`)",
				},
				"bar": {
					EntryPoints: []string{"web"},
					Service:     "foo-service",
					Rule:        "Host(`foo.bar`)",
				},
			},
			expectedError: 1,
		},
		{
			desc: "All router with wrong rule",
			serviceConfig: map[string]*dynamic.Service{
				"foo-service": {
					LoadBalancer: &dynamic.ServersLoadBalancer{
						Servers: []dynamic.Server{
							{
								URL: "http://127.0.0.1",
							},
						},
					},
				},
			},
			routerConfig: map[string]*dynamic.Router{
				"foo": {
					EntryPoints: []string{"web"},
					Service:     "foo-service",
					Rule:        "WrongRule(`bar.foo`)",
				},
				"bar": {
					EntryPoints: []string{"web"},
					Service:     "foo-service",
					Rule:        "WrongRule(`foo.bar`)",
				},
			},
			expectedError: 2,
		},
		{
			desc: "Router with unknown service",
			serviceConfig: map[string]*dynamic.Service{
				"foo-service": {
					LoadBalancer: &dynamic.ServersLoadBalancer{
						Servers: []dynamic.Server{
							{
								URL: "http://127.0.0.1",
							},
						},
					},
				},
			},
			routerConfig: map[string]*dynamic.Router{
				"foo": {
					EntryPoints: []string{"web"},
					Service:     "wrong-service",
					Rule:        "Host(`bar.foo`)",
				},
				"bar": {
					EntryPoints: []string{"web"},
					Service:     "foo-service",
					Rule:        "Host(`foo.bar`)",
				},
			},
			expectedError: 1,
		},
		{
			desc: "Router with broken service",
			serviceConfig: map[string]*dynamic.Service{
				"foo-service": {
					LoadBalancer: nil,
				},
			},
			routerConfig: map[string]*dynamic.Router{
				"bar": {
					EntryPoints: []string{"web"},
					Service:     "foo-service",
					Rule:        "Host(`foo.bar`)",
				},
			},
			expectedError: 2,
		},
		{
			desc: "Router with middleware",
			serviceConfig: map[string]*dynamic.Service{
				"foo-service": {
					LoadBalancer: &dynamic.ServersLoadBalancer{
						Servers: []dynamic.Server{
							{
								URL: "http://127.0.0.1",
							},
						},
					},
				},
			},
			middlewareConfig: map[string]*dynamic.Middleware{
				"auth": {
					BasicAuth: &dynamic.BasicAuth{
						Users: []string{"admin:admin"},
					},
				},
				"addPrefixTest": {
					AddPrefix: &dynamic.AddPrefix{
						Prefix: "/toto",
					},
				},
			},
			routerConfig: map[string]*dynamic.Router{
				"bar": {
					EntryPoints: []string{"web"},
					Service:     "foo-service",
					Rule:        "Host(`foo.bar`)",
					Middlewares: []string{"auth", "addPrefixTest"},
				},
				"test": {
					EntryPoints: []string{"web"},
					Service:     "foo-service",
					Rule:        "Host(`foo.bar.other`)",
					Middlewares: []string{"addPrefixTest", "auth"},
				},
			},
		},
		{
			desc: "Router with unknown middleware",
			serviceConfig: map[string]*dynamic.Service{
				"foo-service": {
					LoadBalancer: &dynamic.ServersLoadBalancer{
						Servers: []dynamic.Server{
							{
								URL: "http://127.0.0.1",
							},
						},
					},
				},
			},
			middlewareConfig: map[string]*dynamic.Middleware{
				"auth": {
					BasicAuth: &dynamic.BasicAuth{
						Users: []string{"admin:admin"},
					},
				},
			},
			routerConfig: map[string]*dynamic.Router{
				"bar": {
					EntryPoints: []string{"web"},
					Service:     "foo-service",
					Rule:        "Host(`foo.bar`)",
					Middlewares: []string{"unknown"},
				},
			},
			expectedError: 1,
		},

		{
			desc: "Router with broken middleware",
			serviceConfig: map[string]*dynamic.Service{
				"foo-service": {
					LoadBalancer: &dynamic.ServersLoadBalancer{
						Servers: []dynamic.Server{
							{
								URL: "http://127.0.0.1",
							},
						},
					},
				},
			},
			middlewareConfig: map[string]*dynamic.Middleware{
				"auth": {
					BasicAuth: &dynamic.BasicAuth{
						Users: []string{"foo"},
					},
				},
			},
			routerConfig: map[string]*dynamic.Router{
				"bar": {
					EntryPoints: []string{"web"},
					Service:     "foo-service",
					Rule:        "Host(`foo.bar`)",
					Middlewares: []string{"auth"},
				},
			},
			expectedError: 2,
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			entryPoints := []string{"web"}

			rtConf := runtime.NewConfig(dynamic.Configuration{
				HTTP: &dynamic.HTTPConfiguration{
					Services:    test.serviceConfig,
					Routers:     test.routerConfig,
					Middlewares: test.middlewareConfig,
				},
			})

			roundTripperManager := service.NewRoundTripperManager()
			roundTripperManager.Update(map[string]*dynamic.ServersTransport{"default@internal": {}})
			serviceManager := service.NewManager(rtConf.Services, nil, nil, roundTripperManager)
			middlewaresBuilder := middleware.NewBuilder(rtConf.Middlewares, serviceManager, nil)
			chainBuilder := middleware.NewChainBuilder(static.Configuration{}, nil, nil)

			routerManager := NewManager(rtConf, serviceManager, middlewaresBuilder, chainBuilder, metrics.NewVoidRegistry())

			_ = routerManager.BuildHandlers(context.Background(), entryPoints, false)

			// even though rtConf was passed by argument to the manager builders above,
			// it's ok to use it as the result we check, because everything worth checking
			// can be accessed by pointers in it.
			var allErrors int
			for _, v := range rtConf.Services {
				if v.Err != nil {
					allErrors++
				}
			}
			for _, v := range rtConf.Routers {
				if len(v.Err) > 0 {
					allErrors++
				}
			}
			for _, v := range rtConf.Middlewares {
				if v.Err != nil {
					allErrors++
				}
			}
			assert.Equal(t, test.expectedError, allErrors)
		})
	}
}

func TestProviderOnMiddlewares(t *testing.T) {
	entryPoints := []string{"web"}

	staticCfg := static.Configuration{
		EntryPoints: map[string]*static.EntryPoint{
			"web": {
				Address: ":80",
			},
		},
	}

	rtConf := runtime.NewConfig(dynamic.Configuration{
		HTTP: &dynamic.HTTPConfiguration{
			Services: map[string]*dynamic.Service{
				"test@file": {
					LoadBalancer: &dynamic.ServersLoadBalancer{
						Servers: []dynamic.Server{},
					},
				},
			},
			Routers: map[string]*dynamic.Router{
				"router@file": {
					EntryPoints: []string{"web"},
					Rule:        "Host(`test`)",
					Service:     "test@file",
					Middlewares: []string{"chain@file", "m1"},
				},
				"router@docker": {
					EntryPoints: []string{"web"},
					Rule:        "Host(`test`)",
					Service:     "test@file",
					Middlewares: []string{"chain", "m1@file"},
				},
			},
			Middlewares: map[string]*dynamic.Middleware{
				"chain@file": {
					Chain: &dynamic.Chain{Middlewares: []string{"m1", "m2", "m1@file"}},
				},
				"chain@docker": {
					Chain: &dynamic.Chain{Middlewares: []string{"m1", "m2", "m1@file"}},
				},
				"m1@file":   {AddPrefix: &dynamic.AddPrefix{Prefix: "/m1"}},
				"m2@file":   {AddPrefix: &dynamic.AddPrefix{Prefix: "/m2"}},
				"m1@docker": {AddPrefix: &dynamic.AddPrefix{Prefix: "/m1"}},
				"m2@docker": {AddPrefix: &dynamic.AddPrefix{Prefix: "/m2"}},
			},
		},
	})

	roundTripperManager := service.NewRoundTripperManager()
	roundTripperManager.Update(map[string]*dynamic.ServersTransport{"default@internal": {}})
	serviceManager := service.NewManager(rtConf.Services, nil, nil, roundTripperManager)
	middlewaresBuilder := middleware.NewBuilder(rtConf.Middlewares, serviceManager, nil)
	chainBuilder := middleware.NewChainBuilder(staticCfg, nil, nil)

	routerManager := NewManager(rtConf, serviceManager, middlewaresBuilder, chainBuilder, metrics.NewVoidRegistry())

	_ = routerManager.BuildHandlers(context.Background(), entryPoints, false)

	assert.Equal(t, []string{"chain@file", "m1@file"}, rtConf.Routers["router@file"].Middlewares)
	assert.Equal(t, []string{"m1@file", "m2@file", "m1@file"}, rtConf.Middlewares["chain@file"].Chain.Middlewares)
	assert.Equal(t, []string{"chain@docker", "m1@file"}, rtConf.Routers["router@docker"].Middlewares)
	assert.Equal(t, []string{"m1@docker", "m2@docker", "m1@file"}, rtConf.Middlewares["chain@docker"].Chain.Middlewares)
}

type staticRoundTripperGetter struct {
	res *http.Response
}

func (s staticRoundTripperGetter) Get(name string) (http.RoundTripper, error) {
	return &staticTransport{res: s.res}, nil
}

type staticTransport struct {
	res *http.Response
}

func (t *staticTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	return t.res, nil
}

func BenchmarkRouterServe(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	b.Cleanup(func() { server.Close() })

	res := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader("")),
	}

	routersConfig := map[string]*dynamic.Router{
		"foo": {
			EntryPoints: []string{"web"},
			Service:     "foo-service",
			Rule:        "Host(`foo.bar`) && Path(`/`)",
		},
	}
	serviceConfig := map[string]*dynamic.Service{
		"foo-service": {
			LoadBalancer: &dynamic.ServersLoadBalancer{
				Servers: []dynamic.Server{
					{
						URL: server.URL,
					},
				},
			},
		},
	}
	entryPoints := []string{"web"}

	rtConf := runtime.NewConfig(dynamic.Configuration{
		HTTP: &dynamic.HTTPConfiguration{
			Services:    serviceConfig,
			Routers:     routersConfig,
			Middlewares: map[string]*dynamic.Middleware{},
		},
	})

	serviceManager := service.NewManager(rtConf.Services, nil, nil, staticRoundTripperGetter{res})
	middlewaresBuilder := middleware.NewBuilder(rtConf.Middlewares, serviceManager, nil)
	chainBuilder := middleware.NewChainBuilder(static.Configuration{}, nil, nil)

	routerManager := NewManager(rtConf, serviceManager, middlewaresBuilder, chainBuilder, metrics.NewVoidRegistry())

	handlers := routerManager.BuildHandlers(context.Background(), entryPoints, false)

	w := httptest.NewRecorder()
	req := testhelpers.MustNewRequest(http.MethodGet, "http://foo.bar/", nil)

	reqHost := requestdecorator.New(nil)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		reqHost.ServeHTTP(w, req, handlers["web"].ServeHTTP)
	}
}

func BenchmarkService(b *testing.B) {
	res := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader("")),
	}

	serviceConfig := map[string]*dynamic.Service{
		"foo-service": {
			LoadBalancer: &dynamic.ServersLoadBalancer{
				Servers: []dynamic.Server{
					{
						URL: "tchouck",
					},
				},
			},
		},
	}

	rtConf := runtime.NewConfig(dynamic.Configuration{
		HTTP: &dynamic.HTTPConfiguration{
			Services: serviceConfig,
		},
	})

	serviceManager := service.NewManager(rtConf.Services, nil, nil, staticRoundTripperGetter{res})
	w := httptest.NewRecorder()
	req := testhelpers.MustNewRequest(http.MethodGet, "http://foo.bar/", nil)

	handler, _ := serviceManager.BuildHTTP(context.Background(), "foo-service")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		handler.ServeHTTP(w, req)
	}
}
