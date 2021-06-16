package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/config/runtime"
	"github.com/traefik/traefik/v2/pkg/config/static"
	"github.com/traefik/traefik/v2/pkg/metrics"
	"github.com/traefik/traefik/v2/pkg/server/middleware"
	"github.com/traefik/traefik/v2/pkg/server/service"
	th "github.com/traefik/traefik/v2/pkg/testhelpers"
	"github.com/traefik/traefik/v2/pkg/tls"
)

func TestReuseService(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusOK)
	}))
	defer testServer.Close()

	staticConfig := static.Configuration{
		EntryPoints: map[string]*static.EntryPoint{
			"web": {},
		},
	}

	dynamicConfigs := th.BuildConfiguration(
		th.WithRouters(
			th.WithRouter("foo",
				th.WithEntryPoints("web"),
				th.WithServiceName("bar"),
				th.WithRule("Path(`/ok`)")),
			th.WithRouter("foo2",
				th.WithEntryPoints("web"),
				th.WithRule("Path(`/unauthorized`)"),
				th.WithServiceName("bar"),
				th.WithRouterMiddlewares("basicauth")),
		),
		th.WithMiddlewares(th.WithMiddleware("basicauth",
			th.WithBasicAuth(&dynamic.BasicAuth{Users: []string{"foo:bar"}}),
		)),
		th.WithLoadBalancerServices(th.WithService("bar",
			th.WithServers(th.WithServer(testServer.URL))),
		),
	)

	roundTripperManager := service.NewRoundTripperManager()
	roundTripperManager.Update(map[string]*dynamic.ServersTransport{"default@internal": {}})
	managerFactory := service.NewManagerFactory(staticConfig, nil, metrics.NewVoidRegistry(), roundTripperManager, nil)
	tlsManager := tls.NewManager()

	factory := NewRouterFactory(staticConfig, managerFactory, tlsManager, middleware.NewChainBuilder(staticConfig, metrics.NewVoidRegistry(), nil), nil, metrics.NewVoidRegistry())

	entryPointsHandlers, _ := factory.CreateRouters(runtime.NewConfig(dynamic.Configuration{HTTP: dynamicConfigs}))

	// Test that the /ok path returns a status 200.
	responseRecorderOk := &httptest.ResponseRecorder{}
	requestOk := httptest.NewRequest(http.MethodGet, testServer.URL+"/ok", nil)
	entryPointsHandlers["web"].GetHTTPHandler().ServeHTTP(responseRecorderOk, requestOk)

	assert.Equal(t, http.StatusOK, responseRecorderOk.Result().StatusCode, "status code")

	// Test that the /unauthorized path returns a 401 because of
	// the basic authentication defined on the frontend.
	responseRecorderUnauthorized := &httptest.ResponseRecorder{}
	requestUnauthorized := httptest.NewRequest(http.MethodGet, testServer.URL+"/unauthorized", nil)
	entryPointsHandlers["web"].GetHTTPHandler().ServeHTTP(responseRecorderUnauthorized, requestUnauthorized)

	assert.Equal(t, http.StatusUnauthorized, responseRecorderUnauthorized.Result().StatusCode, "status code")
}

func TestServerResponseEmptyBackend(t *testing.T) {
	const requestPath = "/path"
	const routeRule = "Path(`" + requestPath + "`)"

	testCases := []struct {
		desc               string
		config             func(testServerURL string) *dynamic.HTTPConfiguration
		expectedStatusCode int
	}{
		{
			desc: "Ok",
			config: func(testServerURL string) *dynamic.HTTPConfiguration {
				return th.BuildConfiguration(
					th.WithRouters(th.WithRouter("foo",
						th.WithEntryPoints("web"),
						th.WithServiceName("bar"),
						th.WithRule(routeRule)),
					),
					th.WithLoadBalancerServices(th.WithService("bar",
						th.WithServers(th.WithServer(testServerURL))),
					),
				)
			},
			expectedStatusCode: http.StatusOK,
		},
		{
			desc: "No Frontend",
			config: func(testServerURL string) *dynamic.HTTPConfiguration {
				return th.BuildConfiguration()
			},
			expectedStatusCode: http.StatusNotFound,
		},
		{
			desc: "Empty Backend LB",
			config: func(testServerURL string) *dynamic.HTTPConfiguration {
				return th.BuildConfiguration(
					th.WithRouters(th.WithRouter("foo",
						th.WithEntryPoints("web"),
						th.WithServiceName("bar"),
						th.WithRule(routeRule)),
					),
					th.WithLoadBalancerServices(th.WithService("bar")),
				)
			},
			expectedStatusCode: http.StatusServiceUnavailable,
		},
		{
			desc: "Empty Backend LB Sticky",
			config: func(testServerURL string) *dynamic.HTTPConfiguration {
				return th.BuildConfiguration(
					th.WithRouters(th.WithRouter("foo",
						th.WithEntryPoints("web"),
						th.WithServiceName("bar"),
						th.WithRule(routeRule)),
					),
					th.WithLoadBalancerServices(th.WithService("bar",
						th.WithSticky("test")),
					),
				)
			},
			expectedStatusCode: http.StatusServiceUnavailable,
		},
		{
			desc: "Empty Backend LB",
			config: func(testServerURL string) *dynamic.HTTPConfiguration {
				return th.BuildConfiguration(
					th.WithRouters(th.WithRouter("foo",
						th.WithEntryPoints("web"),
						th.WithServiceName("bar"),
						th.WithRule(routeRule)),
					),
					th.WithLoadBalancerServices(th.WithService("bar")),
				)
			},
			expectedStatusCode: http.StatusServiceUnavailable,
		},
		{
			desc: "Empty Backend LB Sticky",
			config: func(testServerURL string) *dynamic.HTTPConfiguration {
				return th.BuildConfiguration(
					th.WithRouters(th.WithRouter("foo",
						th.WithEntryPoints("web"),
						th.WithServiceName("bar"),
						th.WithRule(routeRule)),
					),
					th.WithLoadBalancerServices(th.WithService("bar",
						th.WithSticky("test")),
					),
				)
			},
			expectedStatusCode: http.StatusServiceUnavailable,
		},
	}

	for _, test := range testCases {
		test := test

		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			testServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				rw.WriteHeader(http.StatusOK)
			}))
			defer testServer.Close()

			staticConfig := static.Configuration{
				EntryPoints: map[string]*static.EntryPoint{
					"web": {},
				},
			}

			roundTripperManager := service.NewRoundTripperManager()
			roundTripperManager.Update(map[string]*dynamic.ServersTransport{"default@internal": {}})
			managerFactory := service.NewManagerFactory(staticConfig, nil, metrics.NewVoidRegistry(), roundTripperManager, nil)
			tlsManager := tls.NewManager()

			factory := NewRouterFactory(staticConfig, managerFactory, tlsManager, middleware.NewChainBuilder(staticConfig, metrics.NewVoidRegistry(), nil), nil, metrics.NewVoidRegistry())

			entryPointsHandlers, _ := factory.CreateRouters(runtime.NewConfig(dynamic.Configuration{HTTP: test.config(testServer.URL)}))

			responseRecorder := &httptest.ResponseRecorder{}
			request := httptest.NewRequest(http.MethodGet, testServer.URL+requestPath, nil)

			entryPointsHandlers["web"].GetHTTPHandler().ServeHTTP(responseRecorder, request)

			assert.Equal(t, test.expectedStatusCode, responseRecorder.Result().StatusCode, "status code")
		})
	}
}

func TestInternalServices(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusOK)
	}))
	defer testServer.Close()

	staticConfig := static.Configuration{
		API: &static.API{},
		EntryPoints: map[string]*static.EntryPoint{
			"web": {},
		},
	}

	dynamicConfigs := th.BuildConfiguration(
		th.WithRouters(
			th.WithRouter("foo",
				th.WithEntryPoints("web"),
				th.WithServiceName("api@internal"),
				th.WithRule("PathPrefix(`/api`)")),
		),
	)

	roundTripperManager := service.NewRoundTripperManager()
	roundTripperManager.Update(map[string]*dynamic.ServersTransport{"default@internal": {}})
	managerFactory := service.NewManagerFactory(staticConfig, nil, metrics.NewVoidRegistry(), roundTripperManager, nil)
	tlsManager := tls.NewManager()

	voidRegistry := metrics.NewVoidRegistry()

	factory := NewRouterFactory(staticConfig, managerFactory, tlsManager, middleware.NewChainBuilder(staticConfig, voidRegistry, nil), nil, voidRegistry)

	entryPointsHandlers, _ := factory.CreateRouters(runtime.NewConfig(dynamic.Configuration{HTTP: dynamicConfigs}))

	// Test that the /ok path returns a status 200.
	responseRecorderOk := &httptest.ResponseRecorder{}
	requestOk := httptest.NewRequest(http.MethodGet, testServer.URL+"/api/rawdata", nil)
	entryPointsHandlers["web"].GetHTTPHandler().ServeHTTP(responseRecorderOk, requestOk)

	assert.Equal(t, http.StatusOK, responseRecorderOk.Result().StatusCode, "status code")
}
