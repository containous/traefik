package ingress

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/types"
)

func Test_parseRouterConfig(t *testing.T) {
	testCases := []struct {
		desc        string
		annotations map[string]string
		expected    *RouterConfig
	}{
		{
			desc: "router annotations",
			annotations: map[string]string{
				"ingress.kubernetes.io/foo":                               "bar",
				"traefik.ingress.kubernetes.io/foo":                       "bar",
				"traefik.ingress.kubernetes.io/router.pathmatcher":        "foobar",
				"traefik.ingress.kubernetes.io/router.entrypoints":        "foobar,foobar",
				"traefik.ingress.kubernetes.io/router.middlewares":        "foobar,foobar",
				"traefik.ingress.kubernetes.io/router.priority":           "42",
				"traefik.ingress.kubernetes.io/router.tls":                "true",
				"traefik.ingress.kubernetes.io/router.tls.certresolver":   "foobar",
				"traefik.ingress.kubernetes.io/router.tls.domains.0.main": "foobar",
				"traefik.ingress.kubernetes.io/router.tls.domains.0.sans": "foobar,foobar",
				"traefik.ingress.kubernetes.io/router.tls.domains.1.main": "foobar",
				"traefik.ingress.kubernetes.io/router.tls.domains.1.sans": "foobar,foobar",
				"traefik.ingress.kubernetes.io/router.tls.options":        "foobar",
			},
			expected: &RouterConfig{
				Router: &RouterIng{
					PathMatcher: "foobar",
					EntryPoints: []string{"foobar", "foobar"},
					Middlewares: []string{"foobar", "foobar"},
					Priority:    42,
					TLS: &dynamic.RouterTLSConfig{
						CertResolver: "foobar",
						Domains: []types.Domain{
							{
								Main: "foobar",
								SANs: []string{"foobar", "foobar"},
							},
							{
								Main: "foobar",
								SANs: []string{"foobar", "foobar"},
							},
						},
						Options: "foobar",
					},
				},
			},
		},
		{
			desc: "simple TLS annotation",
			annotations: map[string]string{
				"traefik.ingress.kubernetes.io/router.tls": "true",
			},
			expected: &RouterConfig{
				Router: &RouterIng{
					PathMatcher: "PathPrefix",
					TLS:         &dynamic.RouterTLSConfig{},
				},
			},
		},
		{
			desc:        "empty map",
			annotations: nil,
			expected:    nil,
		},
		{
			desc:        "nil map",
			annotations: nil,
			expected:    nil,
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			cfg, err := parseRouterConfig(test.annotations)
			require.NoError(t, err)

			assert.Equal(t, test.expected, cfg)
		})
	}
}

func Test_parseServiceConfig(t *testing.T) {
	testCases := []struct {
		desc        string
		annotations map[string]string
		expected    *ServiceConfig
	}{
		{
			desc: "service annotations",
			annotations: map[string]string{
				"ingress.kubernetes.io/foo":                                    "bar",
				"traefik.ingress.kubernetes.io/foo":                            "bar",
				"traefik.ingress.kubernetes.io/service.serversscheme":          "protocol",
				"traefik.ingress.kubernetes.io/service.serverstransport":       "foobar@file",
				"traefik.ingress.kubernetes.io/service.passhostheader":         "true",
				"traefik.ingress.kubernetes.io/service.sticky.cookie":          "true",
				"traefik.ingress.kubernetes.io/service.sticky.cookie.httponly": "true",
				"traefik.ingress.kubernetes.io/service.sticky.cookie.name":     "foobar",
				"traefik.ingress.kubernetes.io/service.sticky.cookie.secure":   "true",
				"traefik.ingress.kubernetes.io/service.sticky.cookie.samesite": "none",
			},
			expected: &ServiceConfig{
				Service: &ServiceIng{
					Sticky: &dynamic.Sticky{
						Cookie: &dynamic.Cookie{
							Name:     "foobar",
							Secure:   true,
							HTTPOnly: true,
							SameSite: "none",
						},
					},
					ServersScheme:    "protocol",
					ServersTransport: "foobar@file",
					PassHostHeader:   Bool(true),
				},
			},
		},
		{
			desc: "simple sticky annotation",
			annotations: map[string]string{
				"traefik.ingress.kubernetes.io/service.sticky.cookie": "true",
			},
			expected: &ServiceConfig{
				Service: &ServiceIng{
					Sticky:         &dynamic.Sticky{Cookie: &dynamic.Cookie{}},
					PassHostHeader: Bool(true),
				},
			},
		},
		{
			desc:        "empty map",
			annotations: map[string]string{},
			expected:    nil,
		},
		{
			desc:        "nil map",
			annotations: nil,
			expected:    nil,
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			cfg, err := parseServiceConfig(test.annotations)
			require.NoError(t, err)

			assert.Equal(t, test.expected, cfg)
		})
	}
}

func Test_convertAnnotations(t *testing.T) {
	testCases := []struct {
		desc        string
		annotations map[string]string
		expected    map[string]string
	}{
		{
			desc: "router annotations",
			annotations: map[string]string{
				"ingress.kubernetes.io/foo":                               "bar",
				"traefik.ingress.kubernetes.io/foo":                       "bar",
				"traefik.ingress.kubernetes.io/router.pathmatcher":        "foobar",
				"traefik.ingress.kubernetes.io/router.entrypoints":        "foobar,foobar",
				"traefik.ingress.kubernetes.io/router.middlewares":        "foobar,foobar",
				"traefik.ingress.kubernetes.io/router.priority":           "42",
				"traefik.ingress.kubernetes.io/router.tls":                "true",
				"traefik.ingress.kubernetes.io/router.tls.certresolver":   "foobar",
				"traefik.ingress.kubernetes.io/router.tls.domains.0.main": "foobar",
				"traefik.ingress.kubernetes.io/router.tls.domains.0.sans": "foobar,foobar",
				"traefik.ingress.kubernetes.io/router.tls.domains.1.main": "foobar",
				"traefik.ingress.kubernetes.io/router.tls.domains.1.sans": "foobar,foobar",
				"traefik.ingress.kubernetes.io/router.tls.options":        "foobar",
			},
			expected: map[string]string{
				"traefik.foo":                        "bar",
				"traefik.router.pathmatcher":         "foobar",
				"traefik.router.entrypoints":         "foobar,foobar",
				"traefik.router.middlewares":         "foobar,foobar",
				"traefik.router.priority":            "42",
				"traefik.router.tls":                 "true",
				"traefik.router.tls.certresolver":    "foobar",
				"traefik.router.tls.domains[0].main": "foobar",
				"traefik.router.tls.domains[0].sans": "foobar,foobar",
				"traefik.router.tls.domains[1].main": "foobar",
				"traefik.router.tls.domains[1].sans": "foobar,foobar",
				"traefik.router.tls.options":         "foobar",
			},
		},
		{
			desc: "service annotations",
			annotations: map[string]string{
				"traefik.ingress.kubernetes.io/service.serversscheme":          "protocol",
				"traefik.ingress.kubernetes.io/service.serverstransport":       "foobar@file",
				"traefik.ingress.kubernetes.io/service.passhostheader":         "true",
				"traefik.ingress.kubernetes.io/service.sticky.cookie":          "true",
				"traefik.ingress.kubernetes.io/service.sticky.cookie.httponly": "true",
				"traefik.ingress.kubernetes.io/service.sticky.cookie.name":     "foobar",
				"traefik.ingress.kubernetes.io/service.sticky.cookie.secure":   "true",
			},
			expected: map[string]string{
				"traefik.service.passhostheader":         "true",
				"traefik.service.serversscheme":          "protocol",
				"traefik.service.serverstransport":       "foobar@file",
				"traefik.service.sticky.cookie":          "true",
				"traefik.service.sticky.cookie.httponly": "true",
				"traefik.service.sticky.cookie.name":     "foobar",
				"traefik.service.sticky.cookie.secure":   "true",
			},
		},
		{
			desc:        "empty map",
			annotations: map[string]string{},
			expected:    nil,
		},
		{
			desc:        "nil map",
			annotations: nil,
			expected:    nil,
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			labels := convertAnnotations(test.annotations)

			assert.Equal(t, test.expected, labels)
		})
	}
}
