package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v2/pkg/config/runtime"
	"github.com/traefik/traefik/v2/pkg/config/static"
)

func TestHandler_EntryPoints(t *testing.T) {
	type expected struct {
		statusCode int
		nextPage   string
		jsonFile   string
	}

	testCases := []struct {
		desc     string
		path     string
		conf     static.Configuration
		expected expected
	}{
		{
			desc: "all entry points, but no config",
			path: "/api/entrypoints",
			conf: static.Configuration{API: &static.API{}, Global: &static.Global{}},
			expected: expected{
				statusCode: http.StatusOK,
				nextPage:   "1",
				jsonFile:   "testdata/entrypoints-empty.json",
			},
		},
		{
			desc: "all entry points",
			path: "/api/entrypoints",
			conf: static.Configuration{
				Global: &static.Global{},
				API:    &static.API{},
				EntryPoints: map[string]*static.EntryPoint{
					"web": {
						Address: ":80",
						Transport: &static.EntryPointsTransport{
							LifeCycle: &static.LifeCycle{
								RequestAcceptGraceTimeout: 1,
								GraceTimeOut:              2,
							},
							RespondingTimeouts: &static.RespondingTimeouts{
								ReadTimeout:  3,
								WriteTimeout: 4,
								IdleTimeout:  5,
							},
						},
						ProxyProtocol: &static.ProxyProtocol{
							Insecure:   true,
							TrustedIPs: []string{"192.168.1.1", "192.168.1.2"},
						},
						ForwardedHeaders: &static.ForwardedHeaders{
							Insecure:   true,
							TrustedIPs: []string{"192.168.1.3", "192.168.1.4"},
						},
					},
					"websecure": {
						Address: ":443",
						Transport: &static.EntryPointsTransport{
							LifeCycle: &static.LifeCycle{
								RequestAcceptGraceTimeout: 10,
								GraceTimeOut:              20,
							},
							RespondingTimeouts: &static.RespondingTimeouts{
								ReadTimeout:  30,
								WriteTimeout: 40,
								IdleTimeout:  50,
							},
						},
						ProxyProtocol: &static.ProxyProtocol{
							Insecure:   true,
							TrustedIPs: []string{"192.168.1.10", "192.168.1.20"},
						},
						ForwardedHeaders: &static.ForwardedHeaders{
							Insecure:   true,
							TrustedIPs: []string{"192.168.1.30", "192.168.1.40"},
						},
					},
				},
			},
			expected: expected{
				statusCode: http.StatusOK,
				nextPage:   "1",
				jsonFile:   "testdata/entrypoints.json",
			},
		},
		{
			desc: "all entry points, pagination, 1 res per page, want page 2",
			path: "/api/entrypoints?page=2&per_page=1",
			conf: static.Configuration{
				Global: &static.Global{},
				API:    &static.API{},
				EntryPoints: map[string]*static.EntryPoint{
					"web1": {Address: ":81"},
					"web2": {Address: ":82"},
					"web3": {Address: ":83"},
				},
			},
			expected: expected{
				statusCode: http.StatusOK,
				nextPage:   "3",
				jsonFile:   "testdata/entrypoints-page2.json",
			},
		},
		{
			desc: "all entry points, pagination, 19 results overall, 7 res per page, want page 3",
			path: "/api/entrypoints?page=3&per_page=7",
			conf: static.Configuration{
				Global:      &static.Global{},
				API:         &static.API{},
				EntryPoints: generateEntryPoints(19),
			},
			expected: expected{
				statusCode: http.StatusOK,
				nextPage:   "1",
				jsonFile:   "testdata/entrypoints-many-lastpage.json",
			},
		},
		{
			desc: "all entry points, pagination, 5 results overall, 10 res per page, want page 2",
			path: "/api/entrypoints?page=2&per_page=10",
			conf: static.Configuration{
				Global:      &static.Global{},
				API:         &static.API{},
				EntryPoints: generateEntryPoints(5),
			},
			expected: expected{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			desc: "all entry points, pagination, 10 results overall, 10 res per page, want page 2",
			path: "/api/entrypoints?page=2&per_page=10",
			conf: static.Configuration{
				Global:      &static.Global{},
				API:         &static.API{},
				EntryPoints: generateEntryPoints(10),
			},
			expected: expected{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			desc: "one entry point by id",
			path: "/api/entrypoints/bar",
			conf: static.Configuration{
				Global: &static.Global{},
				API:    &static.API{},
				EntryPoints: map[string]*static.EntryPoint{
					"bar": {Address: ":81"},
				},
			},
			expected: expected{
				statusCode: http.StatusOK,
				jsonFile:   "testdata/entrypoint-bar.json",
			},
		},
		{
			desc: "one entry point by id, that does not exist",
			path: "/api/entrypoints/foo",
			conf: static.Configuration{
				Global: &static.Global{},
				API:    &static.API{},
				EntryPoints: map[string]*static.EntryPoint{
					"bar": {Address: ":81"},
				},
			},
			expected: expected{
				statusCode: http.StatusNotFound,
			},
		},
		{
			desc: "one entry point by id, but no config",
			path: "/api/entrypoints/foo",
			conf: static.Configuration{API: &static.API{}, Global: &static.Global{}},
			expected: expected{
				statusCode: http.StatusNotFound,
			},
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			handler := New(test.conf, &runtime.Configuration{})
			server := httptest.NewServer(handler.createRouter())

			resp, err := http.DefaultClient.Get(server.URL + test.path)
			require.NoError(t, err)

			require.Equal(t, test.expected.statusCode, resp.StatusCode)

			assert.Equal(t, test.expected.nextPage, resp.Header.Get(nextPageHeader))

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

func generateEntryPoints(nb int) map[string]*static.EntryPoint {
	eps := make(map[string]*static.EntryPoint, nb)
	for i := 0; i < nb; i++ {
		eps[fmt.Sprintf("ep%2d", i)] = &static.EntryPoint{
			Address: ":" + strconv.Itoa(i),
		}
	}

	return eps
}
