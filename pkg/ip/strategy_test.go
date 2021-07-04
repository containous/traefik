package ip

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemoteAddrStrategy_GetIP(t *testing.T) {
	testCases := []struct {
		desc     string
		expected string
	}{
		{
			desc:     "Use RemoteAddr",
			expected: "192.0.2.1",
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			strategy := RemoteAddrStrategy{}
			req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1", nil)
			actual := strategy.GetIP(req)
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestDepthStrategy_GetIP(t *testing.T) {
	testCases := []struct {
		desc          string
		depth         int
		xForwardedFor string
		expected      string
	}{
		{
			desc:          "Use depth",
			depth:         3,
			xForwardedFor: "10.0.0.4,10.0.0.3,10.0.0.2,10.0.0.1",
			expected:      "10.0.0.3",
		},
		{
			desc:          "Use non existing depth in XForwardedFor",
			depth:         2,
			xForwardedFor: "",
			expected:      "",
		},
		{
			desc:          "Use depth that match the first IP in XForwardedFor",
			depth:         2,
			xForwardedFor: "10.0.0.2,10.0.0.1",
			expected:      "10.0.0.2",
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			strategy := DepthStrategy{Depth: test.depth}
			req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1", nil)
			req.Header.Set(xForwardedFor, test.xForwardedFor)
			actual := strategy.GetIP(req)
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestTrustedIPsStrategy_GetIP(t *testing.T) {
	testCases := []struct {
		desc          string
		trustedIPs    []string
		xForwardedFor string
		expected      string
		useRemote     bool
	}{
		{
			desc:          "Trust all IPs",
			trustedIPs:    []string{"10.0.0.4", "10.0.0.3", "10.0.0.2", "10.0.0.1"},
			xForwardedFor: "10.0.0.4,10.0.0.3,10.0.0.2,10.0.0.1",
			expected:      "",
		},
		{
			desc:          "Do not trust all IPs",
			trustedIPs:    []string{"10.0.0.2", "10.0.0.1"},
			xForwardedFor: "10.0.0.4,10.0.0.3,10.0.0.2,10.0.0.1",
			expected:      "10.0.0.3",
		},
		{
			desc:          "Do not trust all IPs with CIDR",
			trustedIPs:    []string{"10.0.0.1/24"},
			xForwardedFor: "127.0.0.1,10.0.0.4,10.0.0.3,10.0.0.2,10.0.0.1",
			expected:      "127.0.0.1",
		},
		{
			desc:          "Trust all IPs with CIDR",
			trustedIPs:    []string{"10.0.0.1/24"},
			xForwardedFor: "10.0.0.4,10.0.0.3,10.0.0.2,10.0.0.1",
			expected:      "",
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			checker, err := NewChecker(test.trustedIPs)
			require.NoError(t, err)

			strategy := PoolStrategy{Checker: checker}
			req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1", nil)
			req.Header.Set(xForwardedFor, test.xForwardedFor)
			actual := strategy.GetIP(req)
			assert.Equal(t, test.expected, actual)
		})
	}
}
