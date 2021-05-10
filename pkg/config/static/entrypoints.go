package static

import (
	"fmt"
	"math"
	"strings"

	ptypes "github.com/traefik/paerser/types"
	"github.com/traefik/traefik/v2/pkg/types"
)

// EntryPoint holds the entry point configuration.
type EntryPoint struct {
	Address          string                `description:"Entry point address." json:"address,omitempty" toml:"address,omitempty" yaml:"address,omitempty"`
	Transport        *EntryPointsTransport `description:"Configures communication between clients and Traefik." json:"transport,omitempty" toml:"transport,omitempty" yaml:"transport,omitempty" export:"true"`
	ProxyProtocol    *ProxyProtocol        `description:"Proxy-Protocol configuration." json:"proxyProtocol,omitempty" toml:"proxyProtocol,omitempty" yaml:"proxyProtocol,omitempty" label:"allowEmpty" file:"allowEmpty" export:"true"`
	ForwardedHeaders *ForwardedHeaders     `description:"Trust client forwarding headers." json:"forwardedHeaders,omitempty" toml:"forwardedHeaders,omitempty" yaml:"forwardedHeaders,omitempty" export:"true"`
	HTTP             HTTPConfig            `description:"HTTP configuration." json:"http,omitempty" toml:"http,omitempty" yaml:"http,omitempty" export:"true"`
	HTTP3            *HTTP3Config          `description:"HTTP3 configuration." json:"http3,omitempty" toml:"http3,omitempty" yaml:"http3,omitempty" export:"true"`
	UDP              *UDPConfig            `description:"UDP configuration." json:"udp,omitempty" toml:"udp,omitempty" yaml:"udp,omitempty"`
}

// GetAddress strips any potential protocol part of the address field of the
// entry point, in order to return the actual address.
func (ep EntryPoint) GetAddress() string {
	splitN := strings.SplitN(ep.Address, "/", 2)
	return splitN[0]
}

// GetProtocol returns the protocol part of the address field of the entry point.
// If none is specified, it defaults to "tcp".
func (ep EntryPoint) GetProtocol() (string, error) {
	splitN := strings.SplitN(ep.Address, "/", 2)
	if len(splitN) < 2 {
		return "tcp", nil
	}

	protocol := strings.ToLower(splitN[1])
	if protocol == "tcp" || protocol == "udp" {
		return protocol, nil
	}

	return "", fmt.Errorf("invalid protocol: %s", splitN[1])
}

// SetDefaults sets the default values.
func (ep *EntryPoint) SetDefaults() {
	ep.Transport = &EntryPointsTransport{}
	ep.Transport.SetDefaults()
	ep.ForwardedHeaders = &ForwardedHeaders{}
	ep.UDP = &UDPConfig{}
	ep.UDP.SetDefaults()
}

// HTTPConfig is the HTTP configuration of an entry point.
type HTTPConfig struct {
	Redirections *Redirections `description:"Set of redirection" json:"redirections,omitempty" toml:"redirections,omitempty" yaml:"redirections,omitempty" export:"true"`
	Middlewares  []string      `description:"Default middlewares for the routers linked to the entry point." json:"middlewares,omitempty" toml:"middlewares,omitempty" yaml:"middlewares,omitempty"  export:"true"`
	TLS          *TLSConfig    `description:"Default TLS configuration for the routers linked to the entry point." json:"tls,omitempty" toml:"tls,omitempty" yaml:"tls,omitempty" label:"allowEmpty" file:"allowEmpty"  export:"true"`
}

// Redirections is a set of redirection for an entry point.
type Redirections struct {
	EntryPoint *RedirectEntryPoint `description:"Set of redirection for an entry point." json:"entryPoint,omitempty" toml:"entryPoint,omitempty" yaml:"entryPoint,omitempty" export:"true"`
}

// RedirectEntryPoint is the definition of an entry point redirection.
type RedirectEntryPoint struct {
	To        string `description:"Targeted entry point of the redirection." json:"to,omitempty" toml:"to,omitempty" yaml:"to,omitempty" export:"true"`
	Scheme    string `description:"Scheme used for the redirection." json:"scheme,omitempty" toml:"scheme,omitempty" yaml:"scheme,omitempty" export:"true"`
	Permanent bool   `description:"Applies a permanent redirection." json:"permanent,omitempty" toml:"permanent,omitempty" yaml:"permanent,omitempty" export:"true"`
	Priority  int    `description:"Priority of the generated router." json:"priority,omitempty" toml:"priority,omitempty" yaml:"priority,omitempty" export:"true"`
}

// HTTP3Config is the HTTP3 configuration of an entry point.
type HTTP3Config struct {
	Enabled      bool   `description:"Enable HTTP3." json:"enabled,omitempty" toml:"enabled,omitempty" yaml:"enabled,omitempty" export:"true"`
}

// SetDefaults sets the default values.
func (r *RedirectEntryPoint) SetDefaults() {
	r.Scheme = "https"
	r.Permanent = true
	r.Priority = math.MaxInt32 - 1
}

// TLSConfig is the default TLS configuration for all the routers associated to the concerned entry point.
type TLSConfig struct {
	Options      string         `description:"Default TLS options for the routers linked to the entry point." json:"options,omitempty" toml:"options,omitempty" yaml:"options,omitempty" export:"true"`
	CertResolver string         `description:"Default certificate resolver for the routers linked to the entry point." json:"certResolver,omitempty" toml:"certResolver,omitempty" yaml:"certResolver,omitempty" export:"true"`
	Domains      []types.Domain `description:"Default TLS domains for the routers linked to the entry point." json:"domains,omitempty" toml:"domains,omitempty" yaml:"domains,omitempty" export:"true"`
}

// ForwardedHeaders Trust client forwarding headers.
type ForwardedHeaders struct {
	Insecure   bool     `description:"Trust all forwarded headers." json:"insecure,omitempty" toml:"insecure,omitempty" yaml:"insecure,omitempty" export:"true"`
	TrustedIPs []string `description:"Trust only forwarded headers from selected IPs." json:"trustedIPs,omitempty" toml:"trustedIPs,omitempty" yaml:"trustedIPs,omitempty"`
}

// ProxyProtocol contains Proxy-Protocol configuration.
type ProxyProtocol struct {
	Insecure   bool     `description:"Trust all." json:"insecure,omitempty" toml:"insecure,omitempty" yaml:"insecure,omitempty" export:"true"`
	TrustedIPs []string `description:"Trust only selected IPs." json:"trustedIPs,omitempty" toml:"trustedIPs,omitempty" yaml:"trustedIPs,omitempty"`
}

// EntryPoints holds the HTTP entry point list.
type EntryPoints map[string]*EntryPoint

// EntryPointsTransport configures communication between clients and Traefik.
type EntryPointsTransport struct {
	LifeCycle          *LifeCycle          `description:"Timeouts influencing the server life cycle." json:"lifeCycle,omitempty" toml:"lifeCycle,omitempty" yaml:"lifeCycle,omitempty" export:"true"`
	RespondingTimeouts *RespondingTimeouts `description:"Timeouts for incoming requests to the Traefik instance." json:"respondingTimeouts,omitempty" toml:"respondingTimeouts,omitempty" yaml:"respondingTimeouts,omitempty" export:"true"`
}

// SetDefaults sets the default values.
func (t *EntryPointsTransport) SetDefaults() {
	t.LifeCycle = &LifeCycle{}
	t.LifeCycle.SetDefaults()
	t.RespondingTimeouts = &RespondingTimeouts{}
	t.RespondingTimeouts.SetDefaults()
}

// UDPConfig is the UDP configuration of an entry point.
type UDPConfig struct {
	Timeout ptypes.Duration `description:"Timeout defines how long to wait on an idle session before releasing the related resources." json:"timeout,omitempty" toml:"timeout,omitempty" yaml:"timeout,omitempty"`
}

// SetDefaults sets the default values.
func (u *UDPConfig) SetDefaults() {
	u.Timeout = ptypes.Duration(DefaultUDPTimeout)
}
