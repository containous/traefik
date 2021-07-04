package integration

import (
	"net/http"
	"os"
	"time"

	"github.com/go-check/check"
	"github.com/traefik/traefik/v2/integration/try"
	checker "github.com/vdemeester/shakers"
)

// Headers tests suite.
type HeadersSuite struct{ BaseSuite }

func (s *HeadersSuite) TestSimpleConfiguration(c *check.C) {
	cmd, display := s.traefikCmd(withConfigFile("fixtures/headers/basic.toml"))
	defer display(c)
	err := cmd.Start()
	c.Assert(err, checker.IsNil)
	defer s.killCmd(cmd)

	// Expected a 404 as we did not configure anything
	err = try.GetRequest("http://127.0.0.1:8000/", 1000*time.Millisecond, try.StatusCodeIs(http.StatusNotFound))
	c.Assert(err, checker.IsNil)
}

func (s *HeadersSuite) TestCorsResponses(c *check.C) {
	file := s.adaptFile(c, "fixtures/headers/cors.toml", struct{}{})
	defer os.Remove(file)
	cmd, display := s.traefikCmd(withConfigFile(file))
	defer display(c)

	err := cmd.Start()
	c.Assert(err, checker.IsNil)
	defer s.killCmd(cmd)

	backend := startTestServer("9000", http.StatusOK, "")
	defer backend.Close()

	err = try.GetRequest(backend.URL, 500*time.Millisecond, try.StatusCodeIs(http.StatusOK))
	c.Assert(err, checker.IsNil)

	testCase := []struct {
		desc           string
		requestHeaders http.Header
		expected       http.Header
		reqHost        string
		method         string
	}{
		{
			desc: "simple access control allow origin",
			requestHeaders: http.Header{
				"Origin": {"https://foo.bar.org"},
			},
			expected: http.Header{
				"Access-Control-Allow-Origin": {"https://foo.bar.org"},
				"Vary":                        {"Origin"},
			},
			reqHost: "test.localhost",
			method:  http.MethodGet,
		},
		{
			desc: "simple preflight request",
			requestHeaders: http.Header{
				"Access-Control-Request-Headers": {"origin"},
				"Access-Control-Request-Method":  {"GET", "OPTIONS"},
				"Origin":                         {"https://foo.bar.org"},
			},
			expected: http.Header{
				"Access-Control-Allow-Origin":  {"https://foo.bar.org"},
				"Access-Control-Max-Age":       {"100"},
				"Access-Control-Allow-Methods": {"GET,OPTIONS,PUT"},
			},
			reqHost: "test.localhost",
			method:  http.MethodOptions,
		},
		{
			desc: "preflight Options request with no cors configured",
			requestHeaders: http.Header{
				"Access-Control-Request-Headers": {"origin"},
				"Access-Control-Request-Method":  {"GET", "OPTIONS"},
				"Origin":                         {"https://foo.bar.org"},
			},
			expected: http.Header{
				"X-Custom-Response-Header": {"True"},
			},
			reqHost: "test2.localhost",
			method:  http.MethodOptions,
		},
		{
			desc: "preflight Get request with no cors configured",
			requestHeaders: http.Header{
				"Access-Control-Request-Headers": {"origin"},
				"Access-Control-Request-Method":  {"GET", "OPTIONS"},
				"Origin":                         {"https://foo.bar.org"},
			},
			expected: http.Header{
				"X-Custom-Response-Header": {"True"},
			},
			reqHost: "test2.localhost",
			method:  http.MethodGet,
		},
	}

	for _, test := range testCase {
		req, err := http.NewRequest(test.method, "http://127.0.0.1:8000/", nil)
		c.Assert(err, checker.IsNil)
		req.Host = test.reqHost
		req.Header = test.requestHeaders

		err = try.Request(req, 500*time.Millisecond, try.HasHeaderStruct(test.expected))
		c.Assert(err, checker.IsNil)
	}
}

func (s *HeadersSuite) TestSecureHeadersResponses(c *check.C) {
	file := s.adaptFile(c, "fixtures/headers/secure.toml", struct{}{})
	defer os.Remove(file)
	cmd, display := s.traefikCmd(withConfigFile(file))
	defer display(c)

	err := cmd.Start()
	c.Assert(err, checker.IsNil)
	defer s.killCmd(cmd)

	backend := startTestServer("9000", http.StatusOK, "")
	defer backend.Close()

	err = try.GetRequest(backend.URL, 500*time.Millisecond, try.StatusCodeIs(http.StatusOK))
	c.Assert(err, checker.IsNil)

	testCase := []struct {
		desc            string
		expected        http.Header
		reqHost         string
		internalReqHost string
	}{
		{
			desc: "Permissions-Policy Set",
			expected: http.Header{
				"Permissions-Policy": {"microphone=(),"},
			},
			reqHost:         "test.localhost",
			internalReqHost: "internal.localhost",
		},
	}

	for _, test := range testCase {
		req, err := http.NewRequest(http.MethodGet, "http://127.0.0.1:8000/", nil)
		c.Assert(err, checker.IsNil)
		req.Host = test.reqHost

		err = try.Request(req, 500*time.Millisecond, try.StatusCodeIs(http.StatusOK), try.HasHeaderStruct(test.expected))
		c.Assert(err, checker.IsNil)

		req, err = http.NewRequest(http.MethodGet, "http://127.0.0.1:8000/api/rawdata", nil)
		c.Assert(err, checker.IsNil)
		req.Host = test.internalReqHost

		err = try.Request(req, 500*time.Millisecond, try.StatusCodeIs(http.StatusOK), try.HasHeaderStruct(test.expected))
		c.Assert(err, checker.IsNil)
	}
}

func (s *HeadersSuite) TestMultipleSecureHeadersResponses(c *check.C) {
	file := s.adaptFile(c, "fixtures/headers/secure_multiple.toml", struct{}{})
	defer os.Remove(file)
	cmd, display := s.traefikCmd(withConfigFile(file))
	defer display(c)

	err := cmd.Start()
	c.Assert(err, checker.IsNil)
	defer s.killCmd(cmd)

	backend := startTestServer("9000", http.StatusOK, "")
	defer backend.Close()

	err = try.GetRequest(backend.URL, 500*time.Millisecond, try.StatusCodeIs(http.StatusOK))
	c.Assert(err, checker.IsNil)

	testCase := []struct {
		desc     string
		expected http.Header
		reqHost  string
	}{
		{
			desc: "Multiple Secure Headers Set",
			expected: http.Header{
				"X-Frame-Options":        {"DENY"},
				"X-Content-Type-Options": {"nosniff"},
			},
			reqHost: "test.localhost",
		},
	}

	for _, test := range testCase {
		req, err := http.NewRequest(http.MethodGet, "http://127.0.0.1:8000/", nil)
		c.Assert(err, checker.IsNil)
		req.Host = test.reqHost

		err = try.Request(req, 500*time.Millisecond, try.HasHeaderStruct(test.expected))
		c.Assert(err, checker.IsNil)
	}
}
