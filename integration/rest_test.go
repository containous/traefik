package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-check/check"
	"github.com/traefik/traefik/v2/integration/try"
	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	checker "github.com/vdemeester/shakers"
)

type RestSuite struct{ BaseSuite }

func (s *RestSuite) SetUpSuite(c *check.C) {
	s.createComposeProject(c, "rest")

	s.composeProject.Start(c)
}

func (s *RestSuite) TestSimpleConfigurationInsecure(c *check.C) {
	cmd, display := s.traefikCmd(withConfigFile("fixtures/rest/simple.toml"))

	defer display(c)
	err := cmd.Start()
	c.Assert(err, checker.IsNil)
	defer s.killCmd(cmd)

	// wait for Traefik
	err = try.GetRequest("http://127.0.0.1:8080/api/rawdata", 1000*time.Millisecond, try.BodyContains("rest@internal"))
	c.Assert(err, checker.IsNil)

	// Expected a 404 as we did not configure anything.
	err = try.GetRequest("http://127.0.0.1:8000/", 1000*time.Millisecond, try.StatusCodeIs(http.StatusNotFound))
	c.Assert(err, checker.IsNil)

	testCase := []struct {
		desc      string
		config    *dynamic.Configuration
		ruleMatch string
	}{
		{
			desc: "deploy http configuration",
			config: &dynamic.Configuration{
				HTTP: &dynamic.HTTPConfiguration{
					Routers: map[string]*dynamic.Router{
						"routerHTTP": {
							EntryPoints: []string{"web"},
							Middlewares: []string{},
							Service:     "serviceHTTP",
							Rule:        "PathPrefix(`/`)",
						},
					},
					Services: map[string]*dynamic.Service{
						"serviceHTTP": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								Servers: []dynamic.Server{
									{
										URL: "http://" + s.composeProject.Container(c, "whoami1").NetworkSettings.IPAddress + ":80",
									},
								},
							},
						},
					},
				},
			},
			ruleMatch: "PathPrefix(`/`)",
		},
		{
			desc: "deploy tcp configuration",
			config: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{
					Routers: map[string]*dynamic.TCPRouter{
						"routerTCP": {
							EntryPoints: []string{"web"},
							Service:     "serviceTCP",
							Rule:        "HostSNI(`*`)",
						},
					},
					Services: map[string]*dynamic.TCPService{
						"serviceTCP": {
							LoadBalancer: &dynamic.TCPServersLoadBalancer{
								Servers: []dynamic.TCPServer{
									{
										Address: s.composeProject.Container(c, "whoami1").NetworkSettings.IPAddress + ":80",
									},
								},
							},
						},
					},
				},
			},
			ruleMatch: "HostSNI(`*`)",
		},
	}

	for _, test := range testCase {
		data, err := json.Marshal(test.config)
		c.Assert(err, checker.IsNil)

		request, err := http.NewRequest(http.MethodPut, "http://127.0.0.1:8080/api/providers/rest", bytes.NewReader(data))
		c.Assert(err, checker.IsNil)

		response, err := http.DefaultClient.Do(request)
		c.Assert(err, checker.IsNil)
		c.Assert(response.StatusCode, checker.Equals, http.StatusOK)

		err = try.GetRequest("http://127.0.0.1:8080/api/rawdata", 3*time.Second, try.BodyContains(test.ruleMatch))
		c.Assert(err, checker.IsNil)

		err = try.GetRequest("http://127.0.0.1:8000/", 1000*time.Millisecond, try.StatusCodeIs(http.StatusOK))
		c.Assert(err, checker.IsNil)
	}
}

func (s *RestSuite) TestSimpleConfiguration(c *check.C) {
	file := s.adaptFile(c, "fixtures/rest/simple_secure.toml", struct{}{})
	defer os.Remove(file)

	cmd, display := s.traefikCmd(withConfigFile(file))

	defer display(c)
	err := cmd.Start()
	c.Assert(err, checker.IsNil)
	defer s.killCmd(cmd)

	// Expected a 404 as we did not configure anything.
	err = try.GetRequest("http://127.0.0.1:8000/", 1000*time.Millisecond, try.StatusCodeIs(http.StatusNotFound))
	c.Assert(err, checker.IsNil)

	err = try.GetRequest("http://127.0.0.1:8080/api/rawdata", 2000*time.Millisecond, try.BodyContains("PathPrefix(`/secure`)"))
	c.Assert(err, checker.IsNil)

	request, err := http.NewRequest(http.MethodPut, "http://127.0.0.1:8080/api/providers/rest", strings.NewReader("{}"))
	c.Assert(err, checker.IsNil)

	response, err := http.DefaultClient.Do(request)
	c.Assert(err, checker.IsNil)
	c.Assert(response.StatusCode, checker.Equals, http.StatusNotFound)

	testCase := []struct {
		desc      string
		config    *dynamic.Configuration
		ruleMatch string
	}{
		{
			desc: "deploy http configuration",
			config: &dynamic.Configuration{
				HTTP: &dynamic.HTTPConfiguration{
					Routers: map[string]*dynamic.Router{
						"router1": {
							EntryPoints: []string{"web"},
							Middlewares: []string{},
							Service:     "service1",
							Rule:        "PathPrefix(`/`)",
						},
					},
					Services: map[string]*dynamic.Service{
						"service1": {
							LoadBalancer: &dynamic.ServersLoadBalancer{
								Servers: []dynamic.Server{
									{
										URL: "http://" + s.composeProject.Container(c, "whoami1").NetworkSettings.IPAddress + ":80",
									},
								},
							},
						},
					},
				},
			},
			ruleMatch: "PathPrefix(`/`)",
		},
		{
			desc: "deploy tcp configuration",
			config: &dynamic.Configuration{
				TCP: &dynamic.TCPConfiguration{
					Routers: map[string]*dynamic.TCPRouter{
						"router1": {
							EntryPoints: []string{"web"},
							Service:     "service1",
							Rule:        "HostSNI(`*`)",
						},
					},
					Services: map[string]*dynamic.TCPService{
						"service1": {
							LoadBalancer: &dynamic.TCPServersLoadBalancer{
								Servers: []dynamic.TCPServer{
									{
										Address: s.composeProject.Container(c, "whoami1").NetworkSettings.IPAddress + ":80",
									},
								},
							},
						},
					},
				},
			},
			ruleMatch: "HostSNI(`*`)",
		},
	}

	for _, test := range testCase {
		data, err := json.Marshal(test.config)
		c.Assert(err, checker.IsNil)

		request, err := http.NewRequest(http.MethodPut, "http://127.0.0.1:8000/secure/api/providers/rest", bytes.NewReader(data))
		c.Assert(err, checker.IsNil)

		response, err := http.DefaultClient.Do(request)
		c.Assert(err, checker.IsNil)
		c.Assert(response.StatusCode, checker.Equals, http.StatusOK)

		err = try.GetRequest("http://127.0.0.1:8080/api/rawdata", 1000*time.Millisecond, try.BodyContains(test.ruleMatch))
		c.Assert(err, checker.IsNil)

		err = try.GetRequest("http://127.0.0.1:8000/", 1000*time.Millisecond, try.StatusCodeIs(http.StatusOK))
		c.Assert(err, checker.IsNil)
	}
}
