package dynamic

import (
	"reflect"

	"github.com/traefik/traefik/v2/pkg/types"
)

// +k8s:deepcopy-gen=true

// TCPConfiguration contains all the TCP configuration parameters.
type TCPConfiguration struct {
	Routers     map[string]*TCPRouter     `json:"routers,omitempty" toml:"routers,omitempty" yaml:"routers,omitempty" export:"true"`
	Services    map[string]*TCPService    `json:"services,omitempty" toml:"services,omitempty" yaml:"services,omitempty" export:"true"`
	Middlewares map[string]*TCPMiddleware `json:"middlewares,omitempty" toml:"middlewares,omitempty" yaml:"middlewares,omitempty" export:"true"`
}

// +k8s:deepcopy-gen=true

// TCPService holds a tcp service configuration (can only be of one type at the same time).
type TCPService struct {
	LoadBalancer *TCPServersLoadBalancer `json:"loadBalancer,omitempty" toml:"loadBalancer,omitempty" yaml:"loadBalancer,omitempty" export:"true"`
	Weighted     *TCPWeightedRoundRobin  `json:"weighted,omitempty" toml:"weighted,omitempty" yaml:"weighted,omitempty" label:"-" export:"true"`
}

// +k8s:deepcopy-gen=true

// TCPWeightedRoundRobin is a weighted round robin tcp load-balancer of services.
type TCPWeightedRoundRobin struct {
	Services []TCPWRRService `json:"services,omitempty" toml:"services,omitempty" yaml:"services,omitempty" export:"true"`
}

// +k8s:deepcopy-gen=true

// TCPWRRService is a reference to a tcp service load-balanced with weighted round robin.
type TCPWRRService struct {
	Name   string `json:"name,omitempty" toml:"name,omitempty" yaml:"name,omitempty" export:"true"`
	Weight *int   `json:"weight,omitempty" toml:"weight,omitempty" yaml:"weight,omitempty" export:"true"`
}

// SetDefaults Default values for a TCPWRRService.
func (w *TCPWRRService) SetDefaults() {
	defaultWeight := 1
	w.Weight = &defaultWeight
}

// +k8s:deepcopy-gen=true

// TCPRouter holds the router configuration.
type TCPRouter struct {
	EntryPoints []string            `json:"entryPoints,omitempty" toml:"entryPoints,omitempty" yaml:"entryPoints,omitempty" export:"true"`
	Middlewares []string            `json:"middlewares,omitempty" toml:"middlewares,omitempty" yaml:"middlewares,omitempty" export:"true"`
	Service     string              `json:"service,omitempty" toml:"service,omitempty" yaml:"service,omitempty" export:"true"`
	Rule        string              `json:"rule,omitempty" toml:"rule,omitempty" yaml:"rule,omitempty"`
	TLS         *RouterTCPTLSConfig `json:"tls,omitempty" toml:"tls,omitempty" yaml:"tls,omitempty" label:"allowEmpty" file:"allowEmpty" export:"true"`
}

// +k8s:deepcopy-gen=true

// RouterTCPTLSConfig holds the TLS configuration for a router.
type RouterTCPTLSConfig struct {
	Passthrough  bool           `json:"passthrough" toml:"passthrough" yaml:"passthrough" export:"true"`
	Options      string         `json:"options,omitempty" toml:"options,omitempty" yaml:"options,omitempty" export:"true"`
	CertResolver string         `json:"certResolver,omitempty" toml:"certResolver,omitempty" yaml:"certResolver,omitempty" export:"true"`
	Domains      []types.Domain `json:"domains,omitempty" toml:"domains,omitempty" yaml:"domains,omitempty" export:"true"`
}

// +k8s:deepcopy-gen=true

// TCPServersLoadBalancer holds the LoadBalancerService configuration.
type TCPServersLoadBalancer struct {
	// TerminationDelay, corresponds to the deadline that the proxy sets, after one
	// of its connected peers indicates it has closed the writing capability of its
	// connection, to close the reading capability as well, hence fully terminating the
	// connection. It is a duration in milliseconds, defaulting to 100. A negative value
	// means an infinite deadline (i.e. the reading capability is never closed).
	TerminationDelay *int           `json:"terminationDelay,omitempty" toml:"terminationDelay,omitempty" yaml:"terminationDelay,omitempty" export:"true"`
	ProxyProtocol    *ProxyProtocol `json:"proxyProtocol,omitempty" toml:"proxyProtocol,omitempty" yaml:"proxyProtocol,omitempty" label:"allowEmpty" file:"allowEmpty" export:"true"`
	Servers          []TCPServer    `json:"servers,omitempty" toml:"servers,omitempty" yaml:"servers,omitempty" label-slice-as-struct:"server" export:"true"`
}

// SetDefaults Default values for a TCPServersLoadBalancer.
func (l *TCPServersLoadBalancer) SetDefaults() {
	defaultTerminationDelay := 100 // in milliseconds
	l.TerminationDelay = &defaultTerminationDelay
}

// Mergeable tells if the given service is mergeable.
func (l *TCPServersLoadBalancer) Mergeable(loadBalancer *TCPServersLoadBalancer) bool {
	savedServers := l.Servers
	defer func() {
		l.Servers = savedServers
	}()
	l.Servers = nil

	savedServersLB := loadBalancer.Servers
	defer func() {
		loadBalancer.Servers = savedServersLB
	}()
	loadBalancer.Servers = nil

	return reflect.DeepEqual(l, loadBalancer)
}

// +k8s:deepcopy-gen=true

// TCPServer holds a TCP Server configuration.
type TCPServer struct {
	Address string `json:"address,omitempty" toml:"address,omitempty" yaml:"address,omitempty" label:"-"`
	Port    string `toml:"-" json:"-" yaml:"-"`
}

// +k8s:deepcopy-gen=true

// ProxyProtocol holds the ProxyProtocol configuration.
type ProxyProtocol struct {
	Version int `json:"version,omitempty" toml:"version,omitempty" yaml:"version,omitempty" export:"true"`
}

// SetDefaults Default values for a ProxyProtocol.
func (p *ProxyProtocol) SetDefaults() {
	p.Version = 2
}
