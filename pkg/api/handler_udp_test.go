package api

import (
	"context"
	"encoding/json"
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

func TestHandler_UDP(t *testing.T) {
	type expected struct {
		statusCode int
		nextPage   string
		jsonFile   string
	}

	testCases := []struct {
		desc     string
		path     string
		conf     runtime.Configuration
		expected expected
	}{
		{
			desc: "all UDP routers, but no config",
			path: "/api/udp/routers",
			conf: runtime.Configuration{},
			expected: expected{
				statusCode: http.StatusOK,
				nextPage:   "1",
				jsonFile:   "testdata/udprouters-empty.json",
			},
		},
		{
			desc: "all UDP routers",
			path: "/api/udp/routers",
			conf: runtime.Configuration{
				UDPRouters: map[string]*runtime.UDPRouterInfo{
					"test@myprovider": {
						UDPRouter: &dynamic.UDPRouter{
							EntryPoints: []string{"web"},
							Service:     "foo-service@myprovider",
						},
						Status: runtime.StatusEnabled,
					},
					"bar@myprovider": {
						UDPRouter: &dynamic.UDPRouter{
							EntryPoints: []string{"web"},
							Service:     "foo-service@myprovider",
						},
						Status: runtime.StatusWarning,
					},
					"foo@myprovider": {
						UDPRouter: &dynamic.UDPRouter{
							EntryPoints: []string{"web"},
							Service:     "foo-service@myprovider",
						},
						Status: runtime.StatusDisabled,
					},
				},
			},
			expected: expected{
				statusCode: http.StatusOK,
				nextPage:   "1",
				jsonFile:   "testdata/udprouters.json",
			},
		},
		{
			desc: "all UDP routers, pagination, 1 res per page, want page 2",
			path: "/api/udp/routers?page=2&per_page=1",
			conf: runtime.Configuration{
				UDPRouters: map[string]*runtime.UDPRouterInfo{
					"bar@myprovider": {
						UDPRouter: &dynamic.UDPRouter{
							EntryPoints: []string{"web"},
							Service:     "foo-service@myprovider",
						},
					},
					"baz@myprovider": {
						UDPRouter: &dynamic.UDPRouter{
							EntryPoints: []string{"web"},
							Service:     "foo-service@myprovider",
						},
					},
					"test@myprovider": {
						UDPRouter: &dynamic.UDPRouter{
							EntryPoints: []string{"web"},
							Service:     "foo-service@myprovider",
						},
					},
				},
			},
			expected: expected{
				statusCode: http.StatusOK,
				nextPage:   "3",
				jsonFile:   "testdata/udprouters-page2.json",
			},
		},
		{
			desc: "UDP routers filtered by status",
			path: "/api/udp/routers?status=enabled",
			conf: runtime.Configuration{
				UDPRouters: map[string]*runtime.UDPRouterInfo{
					"test@myprovider": {
						UDPRouter: &dynamic.UDPRouter{
							EntryPoints: []string{"web"},
							Service:     "foo-service@myprovider",
						},
						Status: runtime.StatusEnabled,
					},
					"bar@myprovider": {
						UDPRouter: &dynamic.UDPRouter{
							EntryPoints: []string{"web"},
							Service:     "foo-service@myprovider",
						},
						Status: runtime.StatusWarning,
					},
					"foo@myprovider": {
						UDPRouter: &dynamic.UDPRouter{
							EntryPoints: []string{"web"},
							Service:     "foo-service@myprovider",
						},
						Status: runtime.StatusDisabled,
					},
				},
			},
			expected: expected{
				statusCode: http.StatusOK,
				nextPage:   "1",
				jsonFile:   "testdata/udprouters-filtered-status.json",
			},
		},
		{
			desc: "UDP routers filtered by search",
			path: "/api/udp/routers?search=bar@my",
			conf: runtime.Configuration{
				UDPRouters: map[string]*runtime.UDPRouterInfo{
					"test@myprovider": {
						UDPRouter: &dynamic.UDPRouter{
							EntryPoints: []string{"web"},
							Service:     "foo-service@myprovider",
						},
						Status: runtime.StatusEnabled,
					},
					"bar@myprovider": {
						UDPRouter: &dynamic.UDPRouter{
							EntryPoints: []string{"web"},
							Service:     "foo-service@myprovider",
						},
						Status: runtime.StatusWarning,
					},
					"foo@myprovider": {
						UDPRouter: &dynamic.UDPRouter{
							EntryPoints: []string{"web"},
							Service:     "foo-service@myprovider",
						},
						Status: runtime.StatusDisabled,
					},
				},
			},
			expected: expected{
				statusCode: http.StatusOK,
				nextPage:   "1",
				jsonFile:   "testdata/udprouters-filtered-search.json",
			},
		},
		{
			desc: "one UDP router by id",
			path: "/api/udp/routers/bar@myprovider",
			conf: runtime.Configuration{
				UDPRouters: map[string]*runtime.UDPRouterInfo{
					"bar@myprovider": {
						UDPRouter: &dynamic.UDPRouter{
							EntryPoints: []string{"web"},
							Service:     "foo-service@myprovider",
						},
					},
				},
			},
			expected: expected{
				statusCode: http.StatusOK,
				jsonFile:   "testdata/udprouter-bar.json",
			},
		},
		{
			desc: "one UDP router by id, that does not exist",
			path: "/api/udp/routers/foo@myprovider",
			conf: runtime.Configuration{
				UDPRouters: map[string]*runtime.UDPRouterInfo{
					"bar@myprovider": {
						UDPRouter: &dynamic.UDPRouter{
							EntryPoints: []string{"web"},
							Service:     "foo-service@myprovider",
						},
					},
				},
			},
			expected: expected{
				statusCode: http.StatusNotFound,
			},
		},
		{
			desc: "one UDP router by id, but no config",
			path: "/api/udp/routers/bar@myprovider",
			conf: runtime.Configuration{},
			expected: expected{
				statusCode: http.StatusNotFound,
			},
		},
		{
			desc: "all udp services, but no config",
			path: "/api/udp/services",
			conf: runtime.Configuration{},
			expected: expected{
				statusCode: http.StatusOK,
				nextPage:   "1",
				jsonFile:   "testdata/udpservices-empty.json",
			},
		},
		{
			desc: "all udp services",
			path: "/api/udp/services",
			conf: runtime.Configuration{
				UDPServices: map[string]*runtime.UDPServiceInfo{
					"bar@myprovider": {
						UDPService: &dynamic.UDPService{
							LoadBalancer: &dynamic.UDPServersLoadBalancer{
								Servers: []dynamic.UDPServer{
									{
										Address: "127.0.0.1:2345",
									},
								},
							},
						},
						UsedBy: []string{"foo@myprovider", "test@myprovider"},
						Status: runtime.StatusEnabled,
					},
					"baz@myprovider": {
						UDPService: &dynamic.UDPService{
							LoadBalancer: &dynamic.UDPServersLoadBalancer{
								Servers: []dynamic.UDPServer{
									{
										Address: "127.0.0.2:2345",
									},
								},
							},
						},
						UsedBy: []string{"foo@myprovider"},
						Status: runtime.StatusWarning,
					},
					"foz@myprovider": {
						UDPService: &dynamic.UDPService{
							LoadBalancer: &dynamic.UDPServersLoadBalancer{
								Servers: []dynamic.UDPServer{
									{
										Address: "127.0.0.2:2345",
									},
								},
							},
						},
						UsedBy: []string{"foo@myprovider"},
						Status: runtime.StatusDisabled,
					},
				},
			},
			expected: expected{
				statusCode: http.StatusOK,
				nextPage:   "1",
				jsonFile:   "testdata/udpservices.json",
			},
		},
		{
			desc: "udp services filtered by status",
			path: "/api/udp/services?status=enabled",
			conf: runtime.Configuration{
				UDPServices: map[string]*runtime.UDPServiceInfo{
					"bar@myprovider": {
						UDPService: &dynamic.UDPService{
							LoadBalancer: &dynamic.UDPServersLoadBalancer{
								Servers: []dynamic.UDPServer{
									{
										Address: "127.0.0.1:2345",
									},
								},
							},
						},
						UsedBy: []string{"foo@myprovider", "test@myprovider"},
						Status: runtime.StatusEnabled,
					},
					"baz@myprovider": {
						UDPService: &dynamic.UDPService{
							LoadBalancer: &dynamic.UDPServersLoadBalancer{
								Servers: []dynamic.UDPServer{
									{
										Address: "127.0.0.2:2345",
									},
								},
							},
						},
						UsedBy: []string{"foo@myprovider"},
						Status: runtime.StatusWarning,
					},
					"foz@myprovider": {
						UDPService: &dynamic.UDPService{
							LoadBalancer: &dynamic.UDPServersLoadBalancer{
								Servers: []dynamic.UDPServer{
									{
										Address: "127.0.0.2:2345",
									},
								},
							},
						},
						UsedBy: []string{"foo@myprovider"},
						Status: runtime.StatusDisabled,
					},
				},
			},
			expected: expected{
				statusCode: http.StatusOK,
				nextPage:   "1",
				jsonFile:   "testdata/udpservices-filtered-status.json",
			},
		},
		{
			desc: "udp services filtered by search",
			path: "/api/udp/services?search=baz@my",
			conf: runtime.Configuration{
				UDPServices: map[string]*runtime.UDPServiceInfo{
					"bar@myprovider": {
						UDPService: &dynamic.UDPService{
							LoadBalancer: &dynamic.UDPServersLoadBalancer{
								Servers: []dynamic.UDPServer{
									{
										Address: "127.0.0.1:2345",
									},
								},
							},
						},
						UsedBy: []string{"foo@myprovider", "test@myprovider"},
						Status: runtime.StatusEnabled,
					},
					"baz@myprovider": {
						UDPService: &dynamic.UDPService{
							LoadBalancer: &dynamic.UDPServersLoadBalancer{
								Servers: []dynamic.UDPServer{
									{
										Address: "127.0.0.2:2345",
									},
								},
							},
						},
						UsedBy: []string{"foo@myprovider"},
						Status: runtime.StatusWarning,
					},
					"foz@myprovider": {
						UDPService: &dynamic.UDPService{
							LoadBalancer: &dynamic.UDPServersLoadBalancer{
								Servers: []dynamic.UDPServer{
									{
										Address: "127.0.0.2:2345",
									},
								},
							},
						},
						UsedBy: []string{"foo@myprovider"},
						Status: runtime.StatusDisabled,
					},
				},
			},
			expected: expected{
				statusCode: http.StatusOK,
				nextPage:   "1",
				jsonFile:   "testdata/udpservices-filtered-search.json",
			},
		},
		{
			desc: "all udp services, 1 res per page, want page 2",
			path: "/api/udp/services?page=2&per_page=1",
			conf: runtime.Configuration{
				UDPServices: map[string]*runtime.UDPServiceInfo{
					"bar@myprovider": {
						UDPService: &dynamic.UDPService{
							LoadBalancer: &dynamic.UDPServersLoadBalancer{
								Servers: []dynamic.UDPServer{
									{
										Address: "127.0.0.1:2345",
									},
								},
							},
						},
						UsedBy: []string{"foo@myprovider", "test@myprovider"},
					},
					"baz@myprovider": {
						UDPService: &dynamic.UDPService{
							LoadBalancer: &dynamic.UDPServersLoadBalancer{
								Servers: []dynamic.UDPServer{
									{
										Address: "127.0.0.2:2345",
									},
								},
							},
						},
						UsedBy: []string{"foo@myprovider"},
					},
					"test@myprovider": {
						UDPService: &dynamic.UDPService{
							LoadBalancer: &dynamic.UDPServersLoadBalancer{
								Servers: []dynamic.UDPServer{
									{
										Address: "127.0.0.3:2345",
									},
								},
							},
						},
					},
				},
			},
			expected: expected{
				statusCode: http.StatusOK,
				nextPage:   "3",
				jsonFile:   "testdata/udpservices-page2.json",
			},
		},
		{
			desc: "one udp service by id",
			path: "/api/udp/services/bar@myprovider",
			conf: runtime.Configuration{
				UDPServices: map[string]*runtime.UDPServiceInfo{
					"bar@myprovider": {
						UDPService: &dynamic.UDPService{
							LoadBalancer: &dynamic.UDPServersLoadBalancer{
								Servers: []dynamic.UDPServer{
									{
										Address: "127.0.0.1:2345",
									},
								},
							},
						},
						UsedBy: []string{"foo@myprovider", "test@myprovider"},
					},
				},
			},
			expected: expected{
				statusCode: http.StatusOK,
				jsonFile:   "testdata/udpservice-bar.json",
			},
		},
		{
			desc: "one udp service by id, that does not exist",
			path: "/api/udp/services/nono@myprovider",
			conf: runtime.Configuration{
				UDPServices: map[string]*runtime.UDPServiceInfo{
					"bar@myprovider": {
						UDPService: &dynamic.UDPService{
							LoadBalancer: &dynamic.UDPServersLoadBalancer{
								Servers: []dynamic.UDPServer{
									{
										Address: "127.0.0.1:2345",
									},
								},
							},
						},
						UsedBy: []string{"foo@myprovider", "test@myprovider"},
					},
				},
			},
			expected: expected{
				statusCode: http.StatusNotFound,
			},
		},
		{
			desc: "one udp service by id, but no config",
			path: "/api/udp/services/foo@myprovider",
			conf: runtime.Configuration{},
			expected: expected{
				statusCode: http.StatusNotFound,
			},
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			rtConf := &test.conf
			// To lazily initialize the Statuses.
			rtConf.PopulateUsedBy()
			rtConf.GetUDPRoutersByEntryPoints(context.Background(), []string{"web"})

			handler := New(static.Configuration{API: &static.API{}, Global: &static.Global{}}, rtConf)
			server := httptest.NewServer(handler.createRouter())

			resp, err := http.DefaultClient.Get(server.URL + test.path)
			require.NoError(t, err)

			assert.Equal(t, test.expected.nextPage, resp.Header.Get(nextPageHeader))

			require.Equal(t, test.expected.statusCode, resp.StatusCode)

			if test.expected.jsonFile == "" {
				return
			}

			assert.Equal(t, resp.Header.Get("Content-Type"), "application/json")

			contents, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			err = resp.Body.Close()
			require.NoError(t, err)

			if *updateExpected {
				var results interface{}
				err := json.Unmarshal(contents, &results)
				require.NoError(t, err)

				newJSON, err := json.MarshalIndent(results, "", "\t")
				require.NoError(t, err)

				err = os.WriteFile(test.expected.jsonFile, newJSON, 0o644)
				require.NoError(t, err)
			}

			data, err := os.ReadFile(test.expected.jsonFile)
			require.NoError(t, err)
			assert.JSONEq(t, string(data), string(contents))
		})
	}
}
