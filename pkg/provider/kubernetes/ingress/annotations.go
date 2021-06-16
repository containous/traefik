package ingress

import (
	"regexp"
	"strings"

	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/config/label"
)

const (
	// https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/#syntax-and-character-set
	annotationsPrefix = "traefik.ingress.kubernetes.io/"
)

var annotationsRegex = regexp.MustCompile(`(.+)\.(\w+)\.(\d+)\.(.+)`)

// RouterConfig is the router's root configuration from annotations.
type RouterConfig struct {
	Router *RouterIng `json:"router,omitempty"`
}

// RouterIng is the router's configuration from annotations.
type RouterIng struct {
	PathMatcher string                   `json:"pathMatcher,omitempty"`
	EntryPoints []string                 `json:"entryPoints,omitempty"`
	Middlewares []string                 `json:"middlewares,omitempty"`
	Priority    int                      `json:"priority,omitempty"`
	TLS         *dynamic.RouterTLSConfig `json:"tls,omitempty" label:"allowEmpty"`
}

// SetDefaults sets the default values.
func (r *RouterIng) SetDefaults() {
	r.PathMatcher = defaultPathMatcher
}

// ServiceConfig is the service's root configuration from annotations.
type ServiceConfig struct {
	Service *ServiceIng `json:"service,omitempty"`
}

// ServiceIng is the service's configuration from annotations.
type ServiceIng struct {
	ServersScheme    string          `json:"serversScheme,omitempty"`
	ServersTransport string          `json:"serversTransport,omitempty"`
	PassHostHeader   *bool           `json:"passHostHeader"`
	Sticky           *dynamic.Sticky `json:"sticky,omitempty" label:"allowEmpty"`
}

// SetDefaults sets the default values.
func (s *ServiceIng) SetDefaults() {
	s.PassHostHeader = func(v bool) *bool { return &v }(true)
}

func parseRouterConfig(annotations map[string]string) (*RouterConfig, error) {
	labels := convertAnnotations(annotations)
	if len(labels) == 0 {
		return nil, nil
	}

	cfg := &RouterConfig{}

	err := label.Decode(labels, cfg, "traefik.router.")
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func parseServiceConfig(annotations map[string]string) (*ServiceConfig, error) {
	labels := convertAnnotations(annotations)
	if len(labels) == 0 {
		return nil, nil
	}

	cfg := &ServiceConfig{}

	err := label.Decode(labels, cfg, "traefik.service.")
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func convertAnnotations(annotations map[string]string) map[string]string {
	if len(annotations) == 0 {
		return nil
	}

	result := make(map[string]string)

	for key, value := range annotations {
		if !strings.HasPrefix(key, annotationsPrefix) {
			continue
		}

		newKey := strings.ReplaceAll(key, "ingress.kubernetes.io/", "")

		if annotationsRegex.MatchString(newKey) {
			newKey = annotationsRegex.ReplaceAllString(newKey, "$1.$2[$3].$4")
		}

		result[newKey] = value
	}

	return result
}
