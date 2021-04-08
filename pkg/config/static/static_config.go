package static

import (
	"fmt"
	stdlog "log"
	"strings"
	"time"

	assetfs "github.com/elazarl/go-bindata-assetfs"
	legolog "github.com/go-acme/lego/v4/log"
	"github.com/sirupsen/logrus"
	ptypes "github.com/traefik/paerser/types"
	"github.com/traefik/traefik/v2/pkg/log"
	"github.com/traefik/traefik/v2/pkg/ping"
	acmeprovider "github.com/traefik/traefik/v2/pkg/provider/acme"
	"github.com/traefik/traefik/v2/pkg/provider/consulcatalog"
	"github.com/traefik/traefik/v2/pkg/provider/docker"
	"github.com/traefik/traefik/v2/pkg/provider/ecs"
	"github.com/traefik/traefik/v2/pkg/provider/file"
	"github.com/traefik/traefik/v2/pkg/provider/http"
	"github.com/traefik/traefik/v2/pkg/provider/kubernetes/crd"
	"github.com/traefik/traefik/v2/pkg/provider/kubernetes/gateway"
	"github.com/traefik/traefik/v2/pkg/provider/kubernetes/ingress"
	"github.com/traefik/traefik/v2/pkg/provider/kv/consul"
	"github.com/traefik/traefik/v2/pkg/provider/kv/etcd"
	"github.com/traefik/traefik/v2/pkg/provider/kv/redis"
	"github.com/traefik/traefik/v2/pkg/provider/kv/zk"
	"github.com/traefik/traefik/v2/pkg/provider/marathon"
	"github.com/traefik/traefik/v2/pkg/provider/rancher"
	"github.com/traefik/traefik/v2/pkg/provider/rest"
	"github.com/traefik/traefik/v2/pkg/tls"
	"github.com/traefik/traefik/v2/pkg/tracing/datadog"
	"github.com/traefik/traefik/v2/pkg/tracing/elastic"
	"github.com/traefik/traefik/v2/pkg/tracing/haystack"
	"github.com/traefik/traefik/v2/pkg/tracing/instana"
	"github.com/traefik/traefik/v2/pkg/tracing/jaeger"
	"github.com/traefik/traefik/v2/pkg/tracing/zipkin"
	"github.com/traefik/traefik/v2/pkg/types"
)

const (
	// DefaultInternalEntryPointName the name of the default internal entry point.
	DefaultInternalEntryPointName = "traefik"

	// DefaultGraceTimeout controls how long Traefik serves pending requests
	// prior to shutting down.
	DefaultGraceTimeout = 10 * time.Second

	// DefaultIdleTimeout before closing an idle connection.
	DefaultIdleTimeout = 180 * time.Second

	// DefaultAcmeCAServer is the default ACME API endpoint.
	DefaultAcmeCAServer = "https://acme-v02.api.letsencrypt.org/directory"

	// DefaultUDPTimeout defines how long to wait by default on an idle session,
	// before releasing all resources related to that session.
	DefaultUDPTimeout = 3 * time.Second
)

// Configuration is the static configuration.
type Configuration struct {
	Global *Global `description:"Global configuration options" json:"global,omitempty" toml:"global,omitempty" yaml:"global,omitempty" export:"true"`

	ServersTransport *ServersTransport `description:"Servers default transport." json:"serversTransport,omitempty" toml:"serversTransport,omitempty" yaml:"serversTransport,omitempty" export:"true"`
	EntryPoints      EntryPoints       `description:"Entry points definition." json:"entryPoints,omitempty" toml:"entryPoints,omitempty" yaml:"entryPoints,omitempty" export:"true"`
	Providers        *Providers        `description:"Providers configuration." json:"providers,omitempty" toml:"providers,omitempty" yaml:"providers,omitempty" export:"true"`

	API     *API           `description:"Enable api/dashboard." json:"api,omitempty" toml:"api,omitempty" yaml:"api,omitempty" label:"allowEmpty" file:"allowEmpty" export:"true"`
	Metrics *types.Metrics `description:"Enable a metrics exporter." json:"metrics,omitempty" toml:"metrics,omitempty" yaml:"metrics,omitempty" export:"true"`
	Ping    *ping.Handler  `description:"Enable ping." json:"ping,omitempty" toml:"ping,omitempty" yaml:"ping,omitempty" label:"allowEmpty" file:"allowEmpty" export:"true"`

	Log       *types.TraefikLog `description:"Traefik log settings." json:"log,omitempty" toml:"log,omitempty" yaml:"log,omitempty" label:"allowEmpty" file:"allowEmpty" export:"true"`
	AccessLog *types.AccessLog  `description:"Access log settings." json:"accessLog,omitempty" toml:"accessLog,omitempty" yaml:"accessLog,omitempty" label:"allowEmpty" file:"allowEmpty" export:"true"`
	Tracing   *Tracing          `description:"OpenTracing configuration." json:"tracing,omitempty" toml:"tracing,omitempty" yaml:"tracing,omitempty" label:"allowEmpty" file:"allowEmpty" export:"true"`

	HostResolver *types.HostResolverConfig `description:"Enable CNAME Flattening." json:"hostResolver,omitempty" toml:"hostResolver,omitempty" yaml:"hostResolver,omitempty" label:"allowEmpty" file:"allowEmpty" export:"true"`

	CertificatesResolvers map[string]CertificateResolver `description:"Certificates resolvers configuration." json:"certificatesResolvers,omitempty" toml:"certificatesResolvers,omitempty" yaml:"certificatesResolvers,omitempty" export:"true"`

	Pilot *Pilot `description:"Traefik Pilot configuration." json:"pilot,omitempty" toml:"pilot,omitempty" yaml:"pilot,omitempty" export:"true"`

	Experimental *Experimental `description:"experimental features." json:"experimental,omitempty" toml:"experimental,omitempty" yaml:"experimental,omitempty" export:"true"`
}

// CertificateResolver contains the configuration for the different types of certificates resolver.
type CertificateResolver struct {
	ACME *acmeprovider.Configuration `description:"Enable ACME (Let's Encrypt): automatic SSL." json:"acme,omitempty" toml:"acme,omitempty" yaml:"acme,omitempty" export:"true"`
}

// Global holds the global configuration.
type Global struct {
	CheckNewVersion    bool `description:"Periodically check if a new version has been released." json:"checkNewVersion,omitempty" toml:"checkNewVersion,omitempty" yaml:"checkNewVersion,omitempty" label:"allowEmpty" file:"allowEmpty" export:"true"`
	SendAnonymousUsage bool `description:"Periodically send anonymous usage statistics. If the option is not specified, it will be enabled by default." json:"sendAnonymousUsage,omitempty" toml:"sendAnonymousUsage,omitempty" yaml:"sendAnonymousUsage,omitempty" label:"allowEmpty" file:"allowEmpty" export:"true"`
}

// ServersTransport options to configure communication between Traefik and the servers.
type ServersTransport struct {
	InsecureSkipVerify  bool                `description:"Disable SSL certificate verification." json:"insecureSkipVerify,omitempty" toml:"insecureSkipVerify,omitempty" yaml:"insecureSkipVerify,omitempty" export:"true"`
	RootCAs             []tls.FileOrContent `description:"Add cert file for self-signed certificate." json:"rootCAs,omitempty" toml:"rootCAs,omitempty" yaml:"rootCAs,omitempty"`
	MaxIdleConnsPerHost int                 `description:"If non-zero, controls the maximum idle (keep-alive) to keep per-host. If zero, DefaultMaxIdleConnsPerHost is used" json:"maxIdleConnsPerHost,omitempty" toml:"maxIdleConnsPerHost,omitempty" yaml:"maxIdleConnsPerHost,omitempty" export:"true"`
	ForwardingTimeouts  *ForwardingTimeouts `description:"Timeouts for requests forwarded to the backend servers." json:"forwardingTimeouts,omitempty" toml:"forwardingTimeouts,omitempty" yaml:"forwardingTimeouts,omitempty" export:"true"`
}

// API holds the API configuration.
type API struct {
	Insecure  bool `description:"Activate API directly on the entryPoint named traefik." json:"insecure,omitempty" toml:"insecure,omitempty" yaml:"insecure,omitempty" export:"true"`
	Dashboard bool `description:"Activate dashboard." json:"dashboard,omitempty" toml:"dashboard,omitempty" yaml:"dashboard,omitempty" export:"true"`
	Debug     bool `description:"Enable additional endpoints for debugging and profiling." json:"debug,omitempty" toml:"debug,omitempty" yaml:"debug,omitempty" export:"true"`
	// TODO: Re-enable statistics
	// Statistics      *types.Statistics `description:"Enable more detailed statistics." json:"statistics,omitempty" toml:"statistics,omitempty" yaml:"statistics,omitempty" export:"true" label:"allowEmpty" file:"allowEmpty"`
	DashboardAssets *assetfs.AssetFS `json:"-" toml:"-" yaml:"-" label:"-" file:"-"`
}

// SetDefaults sets the default values.
func (a *API) SetDefaults() {
	a.Dashboard = true
}

// RespondingTimeouts contains timeout configurations for incoming requests to the Traefik instance.
type RespondingTimeouts struct {
	ReadTimeout  ptypes.Duration `description:"ReadTimeout is the maximum duration for reading the entire request, including the body. If zero, no timeout is set." json:"readTimeout,omitempty" toml:"readTimeout,omitempty" yaml:"readTimeout,omitempty" export:"true"`
	WriteTimeout ptypes.Duration `description:"WriteTimeout is the maximum duration before timing out writes of the response. If zero, no timeout is set." json:"writeTimeout,omitempty" toml:"writeTimeout,omitempty" yaml:"writeTimeout,omitempty" export:"true"`
	IdleTimeout  ptypes.Duration `description:"IdleTimeout is the maximum amount duration an idle (keep-alive) connection will remain idle before closing itself. If zero, no timeout is set." json:"idleTimeout,omitempty" toml:"idleTimeout,omitempty" yaml:"idleTimeout,omitempty" export:"true"`
}

// SetDefaults sets the default values.
func (a *RespondingTimeouts) SetDefaults() {
	a.IdleTimeout = ptypes.Duration(DefaultIdleTimeout)
}

// ForwardingTimeouts contains timeout configurations for forwarding requests to the backend servers.
type ForwardingTimeouts struct {
	DialTimeout           ptypes.Duration `description:"The amount of time to wait until a connection to a backend server can be established. If zero, no timeout exists." json:"dialTimeout,omitempty" toml:"dialTimeout,omitempty" yaml:"dialTimeout,omitempty" export:"true"`
	ResponseHeaderTimeout ptypes.Duration `description:"The amount of time to wait for a server's response headers after fully writing the request (including its body, if any). If zero, no timeout exists." json:"responseHeaderTimeout,omitempty" toml:"responseHeaderTimeout,omitempty" yaml:"responseHeaderTimeout,omitempty" export:"true"`
	IdleConnTimeout       ptypes.Duration `description:"The maximum period for which an idle HTTP keep-alive connection will remain open before closing itself" json:"idleConnTimeout,omitempty" toml:"idleConnTimeout,omitempty" yaml:"idleConnTimeout,omitempty" export:"true"`
}

// SetDefaults sets the default values.
func (f *ForwardingTimeouts) SetDefaults() {
	f.DialTimeout = ptypes.Duration(30 * time.Second)
	f.IdleConnTimeout = ptypes.Duration(90 * time.Second)
}

// LifeCycle contains configurations relevant to the lifecycle (such as the shutdown phase) of Traefik.
type LifeCycle struct {
	RequestAcceptGraceTimeout ptypes.Duration `description:"Duration to keep accepting requests before Traefik initiates the graceful shutdown procedure." json:"requestAcceptGraceTimeout,omitempty" toml:"requestAcceptGraceTimeout,omitempty" yaml:"requestAcceptGraceTimeout,omitempty" export:"true"`
	GraceTimeOut              ptypes.Duration `description:"Duration to give active requests a chance to finish before Traefik stops." json:"graceTimeOut,omitempty" toml:"graceTimeOut,omitempty" yaml:"graceTimeOut,omitempty" export:"true"`
}

// SetDefaults sets the default values.
func (a *LifeCycle) SetDefaults() {
	a.GraceTimeOut = ptypes.Duration(DefaultGraceTimeout)
}

// Tracing holds the tracing configuration.
type Tracing struct {
	ServiceName   string           `description:"Set the name for this service." json:"serviceName,omitempty" toml:"serviceName,omitempty" yaml:"serviceName,omitempty" export:"true"`
	SpanNameLimit int              `description:"Set the maximum character limit for Span names (default 0 = no limit)." json:"spanNameLimit,omitempty" toml:"spanNameLimit,omitempty" yaml:"spanNameLimit,omitempty" export:"true"`
	Jaeger        *jaeger.Config   `description:"Settings for Jaeger." json:"jaeger,omitempty" toml:"jaeger,omitempty" yaml:"jaeger,omitempty" export:"true" label:"allowEmpty" file:"allowEmpty"`
	Zipkin        *zipkin.Config   `description:"Settings for Zipkin." json:"zipkin,omitempty" toml:"zipkin,omitempty" yaml:"zipkin,omitempty" export:"true" label:"allowEmpty" file:"allowEmpty"`
	Datadog       *datadog.Config  `description:"Settings for Datadog." json:"datadog,omitempty" toml:"datadog,omitempty" yaml:"datadog,omitempty" export:"true" label:"allowEmpty" file:"allowEmpty"`
	Instana       *instana.Config  `description:"Settings for Instana." json:"instana,omitempty" toml:"instana,omitempty" yaml:"instana,omitempty" export:"true" label:"allowEmpty" file:"allowEmpty"`
	Haystack      *haystack.Config `description:"Settings for Haystack." json:"haystack,omitempty" toml:"haystack,omitempty" yaml:"haystack,omitempty" export:"true" label:"allowEmpty" file:"allowEmpty"`
	Elastic       *elastic.Config  `description:"Settings for Elastic." json:"elastic,omitempty" toml:"elastic,omitempty" yaml:"elastic,omitempty" export:"true" label:"allowEmpty" file:"allowEmpty"`
}

// SetDefaults sets the default values.
func (t *Tracing) SetDefaults() {
	t.ServiceName = "traefik"
	t.SpanNameLimit = 0
}

// Providers contains providers configuration.
type Providers struct {
	ProvidersThrottleDuration ptypes.Duration `description:"Backends throttle duration: minimum duration between 2 events from providers before applying a new configuration. It avoids unnecessary reloads if multiples events are sent in a short amount of time." json:"providersThrottleDuration,omitempty" toml:"providersThrottleDuration,omitempty" yaml:"providersThrottleDuration,omitempty" export:"true"`

	Docker            *docker.Provider        `description:"Enable Docker backend with default settings." json:"docker,omitempty" toml:"docker,omitempty" yaml:"docker,omitempty" export:"true" label:"allowEmpty" file:"allowEmpty"`
	File              *file.Provider          `description:"Enable File backend with default settings." json:"file,omitempty" toml:"file,omitempty" yaml:"file,omitempty" export:"true"`
	Marathon          *marathon.Provider      `description:"Enable Marathon backend with default settings." json:"marathon,omitempty" toml:"marathon,omitempty" yaml:"marathon,omitempty" export:"true" label:"allowEmpty" file:"allowEmpty"`
	KubernetesIngress *ingress.Provider       `description:"Enable Kubernetes backend with default settings." json:"kubernetesIngress,omitempty" toml:"kubernetesIngress,omitempty" yaml:"kubernetesIngress,omitempty" export:"true" label:"allowEmpty" file:"allowEmpty"`
	KubernetesCRD     *crd.Provider           `description:"Enable Kubernetes backend with default settings." json:"kubernetesCRD,omitempty" toml:"kubernetesCRD,omitempty" yaml:"kubernetesCRD,omitempty" export:"true" label:"allowEmpty" file:"allowEmpty"`
	KubernetesGateway *gateway.Provider       `description:"Enable Kubernetes gateway api provider with default settings." json:"kubernetesGateway,omitempty" toml:"kubernetesGateway,omitempty" yaml:"kubernetesGateway,omitempty" export:"true" label:"allowEmpty" file:"allowEmpty"`
	Rest              *rest.Provider          `description:"Enable Rest backend with default settings." json:"rest,omitempty" toml:"rest,omitempty" yaml:"rest,omitempty" export:"true" label:"allowEmpty" file:"allowEmpty"`
	Rancher           *rancher.Provider       `description:"Enable Rancher backend with default settings." json:"rancher,omitempty" toml:"rancher,omitempty" yaml:"rancher,omitempty" export:"true" label:"allowEmpty" file:"allowEmpty"`
	ConsulCatalog     *consulcatalog.Provider `description:"Enable ConsulCatalog backend with default settings." json:"consulCatalog,omitempty" toml:"consulCatalog,omitempty" yaml:"consulCatalog,omitempty" label:"allowEmpty" file:"allowEmpty" export:"true"`
	Ecs               *ecs.Provider           `description:"Enable AWS ECS backend with default settings." json:"ecs,omitempty" toml:"ecs,omitempty" yaml:"ecs,omitempty" label:"allowEmpty" file:"allowEmpty" export:"true"`

	Consul    *consul.Provider `description:"Enable Consul backend with default settings." json:"consul,omitempty" toml:"consul,omitempty" yaml:"consul,omitempty" label:"allowEmpty" file:"allowEmpty"  export:"true"`
	Etcd      *etcd.Provider   `description:"Enable Etcd backend with default settings." json:"etcd,omitempty" toml:"etcd,omitempty" yaml:"etcd,omitempty" label:"allowEmpty" file:"allowEmpty" export:"true"`
	ZooKeeper *zk.Provider     `description:"Enable ZooKeeper backend with default settings." json:"zooKeeper,omitempty" toml:"zooKeeper,omitempty" yaml:"zooKeeper,omitempty" label:"allowEmpty" file:"allowEmpty" export:"true"`
	Redis     *redis.Provider  `description:"Enable Redis backend with default settings." json:"redis,omitempty" toml:"redis,omitempty" yaml:"redis,omitempty" label:"allowEmpty" file:"allowEmpty" export:"true"`
	HTTP      *http.Provider   `description:"Enable HTTP backend with default settings." json:"http,omitempty" toml:"http,omitempty" yaml:"http,omitempty" label:"allowEmpty" file:"allowEmpty" export:"true"`
}

// SetEffectiveConfiguration adds missing configuration parameters derived from existing ones.
// It also takes care of maintaining backwards compatibility.
func (c *Configuration) SetEffectiveConfiguration() {
	// Creates the default entry point if needed
	if len(c.EntryPoints) == 0 {
		ep := &EntryPoint{Address: ":80"}
		ep.SetDefaults()
		c.EntryPoints = EntryPoints{"http": ep}
	}

	// Creates the internal traefik entry point if needed
	if (c.API != nil && c.API.Insecure) ||
		(c.Ping != nil && !c.Ping.ManualRouting && c.Ping.EntryPoint == DefaultInternalEntryPointName) ||
		(c.Metrics != nil && c.Metrics.Prometheus != nil && !c.Metrics.Prometheus.ManualRouting && c.Metrics.Prometheus.EntryPoint == DefaultInternalEntryPointName) ||
		(c.Providers != nil && c.Providers.Rest != nil && c.Providers.Rest.Insecure) {
		if _, ok := c.EntryPoints[DefaultInternalEntryPointName]; !ok {
			ep := &EntryPoint{Address: ":8080"}
			ep.SetDefaults()
			c.EntryPoints[DefaultInternalEntryPointName] = ep
		}
	}

	if c.Providers.Docker != nil {
		if c.Providers.Docker.SwarmModeRefreshSeconds <= 0 {
			c.Providers.Docker.SwarmModeRefreshSeconds = ptypes.Duration(15 * time.Second)
		}

		if c.Providers.Docker.HTTPClientTimeout < 0 {
			c.Providers.Docker.HTTPClientTimeout = 0
		}
	}

	if c.Providers.Rancher != nil {
		if c.Providers.Rancher.RefreshSeconds <= 0 {
			c.Providers.Rancher.RefreshSeconds = 15
		}
	}

	// Enable anonymous usage when pilot is enabled.
	if c.Pilot != nil && c.Pilot.Token != "" {
		c.Global.SendAnonymousUsage = true
	}

	// Create Pilot struct to apply default value on undefined configuration.
	if c.Pilot == nil {
		c.Pilot = &Pilot{}
		c.Pilot.SetDefaults()
	}

	// Disable Gateway API provider if not enabled in experimental
	if c.Experimental == nil || !c.Experimental.KubernetesGateway {
		c.Providers.KubernetesGateway = nil
	}

	if c.Experimental == nil || !c.Experimental.HTTP3 {
		for _, ep := range c.EntryPoints {
			ep.EnableHTTP3 = false
		}
	}

	// Configure Gateway API provider
	if c.Providers.KubernetesGateway != nil {
		log.WithoutContext().Debugf("Experimental Kubernetes Gateway provider has been activated")
		entryPoints := make(map[string]gateway.Entrypoint)
		for epName, entryPoint := range c.EntryPoints {
			entryPoints[epName] = gateway.Entrypoint{Address: entryPoint.GetAddress(), HasHTTPTLSConf: entryPoint.HTTP.TLS != nil}
		}

		c.Providers.KubernetesGateway.EntryPoints = entryPoints
	}

	c.initACMEProvider()
}

func (c *Configuration) initACMEProvider() {
	for _, resolver := range c.CertificatesResolvers {
		if resolver.ACME != nil {
			resolver.ACME.CAServer = getSafeACMECAServer(resolver.ACME.CAServer)
		}
	}

	legolog.Logger = stdlog.New(log.WithoutContext().WriterLevel(logrus.DebugLevel), "legolog: ", 0)
}

// ValidateConfiguration validate that configuration is coherent.
func (c *Configuration) ValidateConfiguration() error {
	var acmeEmail string
	for name, resolver := range c.CertificatesResolvers {
		if resolver.ACME == nil {
			continue
		}

		if len(resolver.ACME.Storage) == 0 {
			return fmt.Errorf("unable to initialize certificates resolver %q with no storage location for the certificates", name)
		}

		if acmeEmail != "" && resolver.ACME.Email != acmeEmail {
			return fmt.Errorf("unable to initialize certificates resolver %q, all the acme resolvers must use the same email", name)
		}
		acmeEmail = resolver.ACME.Email
	}

	return nil
}

func getSafeACMECAServer(caServerSrc string) string {
	if len(caServerSrc) == 0 {
		return DefaultAcmeCAServer
	}

	if strings.HasPrefix(caServerSrc, "https://acme-v01.api.letsencrypt.org") {
		caServer := strings.Replace(caServerSrc, "v01", "v02", 1)
		log.WithoutContext().
			Warnf("The CA server %[1]q refers to a v01 endpoint of the ACME API, please change to %[2]q. Fallback to %[2]q.", caServerSrc, caServer)
		return caServer
	}

	if strings.HasPrefix(caServerSrc, "https://acme-staging.api.letsencrypt.org") {
		caServer := strings.Replace(caServerSrc, "https://acme-staging.api.letsencrypt.org", "https://acme-staging-v02.api.letsencrypt.org", 1)
		log.WithoutContext().
			Warnf("The CA server %[1]q refers to a v01 endpoint of the ACME API, please change to %[2]q. Fallback to %[2]q.", caServerSrc, caServer)
		return caServer
	}

	return caServerSrc
}
