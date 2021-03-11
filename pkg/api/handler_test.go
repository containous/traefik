package api

import (
	"encoding/json"
	"flag"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/config/runtime"
	"github.com/traefik/traefik/v2/pkg/config/static"
)

var updateExpected = flag.Bool("update_expected", false, "Update expected files in testdata")

func TestHandler_RawData(t *testing.T) {
	type expected struct {
		statusCode int
		json       string
	}

	testCases := []struct {
		desc     string
		path     string
		conf     runtime.Configuration
		expected expected
	}{
		{
			desc: "Get rawdata",
			path: "/api/rawdata",
			conf: runtime.Configuration{
				Services: map[string]*runtime.ServiceInfo{
					"foo-service@myprovider": {
						Service: &dynamic.Service{
							LoadBalancer: &dynamic.ServersLoadBalancer{
								PassHostHeader: Bool(true),
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
				TCPServices: map[string]*runtime.TCPServiceInfo{
					"tcpfoo-service@myprovider": {
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
					"tcpbar@myprovider": {
						TCPRouter: &dynamic.TCPRouter{
							EntryPoints: []string{"web"},
							Service:     "tcpfoo-service@myprovider",
							Rule:        "HostSNI(`foo.bar`)",
						},
					},
					"tcptest@myprovider": {
						TCPRouter: &dynamic.TCPRouter{
							EntryPoints: []string{"web"},
							Service:     "tcpfoo-service@myprovider",
							Rule:        "HostSNI(`foo.bar.other`)",
						},
					},
				},
			},
			expected: expected{
				statusCode: http.StatusOK,
				json:       "testdata/getrawdata.json",
			},
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			// TODO: server status

			rtConf := &test.conf

			rtConf.PopulateUsedBy()
			handler := New(static.Configuration{API: &static.API{}, Global: &static.Global{}}, rtConf)
			server := httptest.NewServer(handler.createRouter())

			resp, err := http.DefaultClient.Get(server.URL + test.path)
			require.NoError(t, err)

			assert.Equal(t, test.expected.statusCode, resp.StatusCode)
			assert.Equal(t, resp.Header.Get("Content-Type"), "application/json")

			contents, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			err = resp.Body.Close()
			require.NoError(t, err)

			if test.expected.json == "" {
				return
			}
			if *updateExpected {
				var rtRepr RunTimeRepresentation
				err := json.Unmarshal(contents, &rtRepr)
				require.NoError(t, err)

				newJSON, err := json.MarshalIndent(rtRepr, "", "\t")
				require.NoError(t, err)

				err = os.WriteFile(test.expected.json, newJSON, 0o644)
				require.NoError(t, err)
			}

			data, err := os.ReadFile(test.expected.json)
			require.NoError(t, err)
			assert.JSONEq(t, string(data), string(contents))
		})
	}
}

func TestHandler_GetMiddleware(t *testing.T) {
	testCases := []struct {
		desc           string
		middlewareName string
		conf           runtime.Configuration
		expectedStatus int
		expected       interface{}
	}{
		{
			desc:           "Middleware not found",
			middlewareName: "auth@myprovider",
			conf: runtime.Configuration{
				Middlewares: map[string]*runtime.MiddlewareInfo{},
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			desc:           "Get middleware",
			middlewareName: "auth@myprovider",
			conf: runtime.Configuration{
				Middlewares: map[string]*runtime.MiddlewareInfo{
					"auth@myprovider": {
						Middleware: &dynamic.Middleware{
							BasicAuth: &dynamic.BasicAuth{
								Users: []string{"admin:admin"},
							},
						},
					},
				},
			},
			expectedStatus: http.StatusOK,
			expected: middlewareRepresentation{
				MiddlewareInfo: &runtime.MiddlewareInfo{
					Middleware: &dynamic.Middleware{
						BasicAuth: &dynamic.BasicAuth{
							Users: []string{"admin:admin"},
						},
					},
				},
				Name:     "auth@myprovider",
				Provider: "myprovider",
				Type:     "basicauth",
			},
		},
		{
			desc:           "Get plugin middleware",
			middlewareName: "myplugin@myprovider",
			conf: runtime.Configuration{
				Middlewares: map[string]*runtime.MiddlewareInfo{
					"myplugin@myprovider": {
						Middleware: &dynamic.Middleware{
							Plugin: map[string]dynamic.PluginConf{
								"mysuperplugin": {
									"foo": "bar",
								},
							},
						},
					},
				},
			},
			expectedStatus: http.StatusOK,
			expected: middlewareRepresentation{
				MiddlewareInfo: &runtime.MiddlewareInfo{
					Middleware: &dynamic.Middleware{
						Plugin: map[string]dynamic.PluginConf{
							"mysuperplugin": {
								"foo": "bar",
							},
						},
					},
				},
				Name:     "myplugin@myprovider",
				Provider: "myprovider",
				Type:     "mysuperplugin",
			},
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			handler := New(static.Configuration{API: &static.API{}, Global: &static.Global{}}, &test.conf)
			server := httptest.NewServer(handler.createRouter())

			resp, err := http.DefaultClient.Get(server.URL + "/api/http/middlewares/" + test.middlewareName)
			require.NoError(t, err)

			assert.Equal(t, test.expectedStatus, resp.StatusCode)

			if test.expected == nil {
				return
			}

			data, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			err = resp.Body.Close()
			require.NoError(t, err)

			expected, err := json.Marshal(test.expected)
			require.NoError(t, err)

			assert.JSONEq(t, string(expected), string(data))
		})
	}
}
