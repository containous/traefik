package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"reflect"
	"time"

	"github.com/containous/alice"
	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/config/runtime"
	"github.com/traefik/traefik/v2/pkg/healthcheck"
	"github.com/traefik/traefik/v2/pkg/log"
	"github.com/traefik/traefik/v2/pkg/metrics"
	"github.com/traefik/traefik/v2/pkg/middlewares/accesslog"
	"github.com/traefik/traefik/v2/pkg/middlewares/emptybackendhandler"
	metricsMiddle "github.com/traefik/traefik/v2/pkg/middlewares/metrics"
	"github.com/traefik/traefik/v2/pkg/middlewares/pipelining"
	"github.com/traefik/traefik/v2/pkg/safe"
	"github.com/traefik/traefik/v2/pkg/server/cookie"
	"github.com/traefik/traefik/v2/pkg/server/provider"
	"github.com/traefik/traefik/v2/pkg/server/service/loadbalancer/mirror"
	"github.com/traefik/traefik/v2/pkg/server/service/loadbalancer/wrr"
	"github.com/vulcand/oxy/roundrobin"
	"github.com/vulcand/oxy/roundrobin/stickycookie"
)

const (
	defaultHealthCheckInterval = 30 * time.Second
	defaultHealthCheckTimeout  = 5 * time.Second
)

const defaultMaxBodySize int64 = -1

// RoundTripperGetter is a roundtripper getter interface.
type RoundTripperGetter interface {
	Get(name string) (http.RoundTripper, error)
}

// NewManager creates a new Manager.
func NewManager(configs map[string]*runtime.ServiceInfo, metricsRegistry metrics.Registry, routinePool *safe.Pool, roundTripperManager RoundTripperGetter) *Manager {
	return &Manager{
		routinePool:         routinePool,
		metricsRegistry:     metricsRegistry,
		bufferPool:          newBufferPool(),
		roundTripperManager: roundTripperManager,
		balancers:           make(map[string]healthcheck.Balancers),
		configs:             configs,
	}
}

// Manager The service manager.
type Manager struct {
	routinePool         *safe.Pool
	metricsRegistry     metrics.Registry
	bufferPool          httputil.BufferPool
	roundTripperManager RoundTripperGetter
	// balancers is the map of all Balancers, keyed by service name.
	// There is one Balancer per service handler, and there is one service handler per reference to a service
	// (e.g. if 2 routers refer to the same service name, 2 service handlers are created),
	// which is why there is not just one Balancer per service name.
	balancers map[string]healthcheck.Balancers
	configs   map[string]*runtime.ServiceInfo
}

// BuildHTTP Creates a http.Handler for a service configuration.
func (m *Manager) BuildHTTP(rootCtx context.Context, serviceName string) (http.Handler, error) {
	ctx := log.With(rootCtx, log.Str(log.ServiceName, serviceName))

	serviceName = provider.GetQualifiedName(ctx, serviceName)
	ctx = provider.AddInContext(ctx, serviceName)

	conf, ok := m.configs[serviceName]
	if !ok {
		return nil, fmt.Errorf("the service %q does not exist", serviceName)
	}

	value := reflect.ValueOf(*conf.Service)
	var count int
	for i := 0; i < value.NumField(); i++ {
		if !value.Field(i).IsNil() {
			count++
		}
	}
	if count > 1 {
		err := errors.New("cannot create service: multi-types service not supported, consider declaring two different pieces of service instead")
		conf.AddError(err, true)
		return nil, err
	}

	var lb http.Handler

	switch {
	case conf.LoadBalancer != nil:
		var err error
		lb, err = m.getLoadBalancerServiceHandler(ctx, serviceName, conf.LoadBalancer)
		if err != nil {
			conf.AddError(err, true)
			return nil, err
		}
	case conf.Weighted != nil:
		var err error
		lb, err = m.getWRRServiceHandler(ctx, serviceName, conf.Weighted)
		if err != nil {
			conf.AddError(err, true)
			return nil, err
		}
	case conf.Mirroring != nil:
		var err error
		lb, err = m.getMirrorServiceHandler(ctx, conf.Mirroring)
		if err != nil {
			conf.AddError(err, true)
			return nil, err
		}
	default:
		sErr := fmt.Errorf("the service %q does not have any type defined", serviceName)
		conf.AddError(sErr, true)
		return nil, sErr
	}

	return lb, nil
}

func (m *Manager) getMirrorServiceHandler(ctx context.Context, config *dynamic.Mirroring) (http.Handler, error) {
	serviceHandler, err := m.BuildHTTP(ctx, config.Service)
	if err != nil {
		return nil, err
	}

	maxBodySize := defaultMaxBodySize
	if config.MaxBodySize != nil {
		maxBodySize = *config.MaxBodySize
	}
	handler := mirror.New(serviceHandler, m.routinePool, maxBodySize)
	for _, mirrorConfig := range config.Mirrors {
		mirrorHandler, err := m.BuildHTTP(ctx, mirrorConfig.Name)
		if err != nil {
			return nil, err
		}

		err = handler.AddMirror(mirrorHandler, mirrorConfig.Percent)
		if err != nil {
			return nil, err
		}
	}
	return handler, nil
}

func (m *Manager) getWRRServiceHandler(ctx context.Context, serviceName string, config *dynamic.WeightedRoundRobin) (http.Handler, error) {
	// TODO Handle accesslog and metrics with multiple service name
	if config.Sticky != nil && config.Sticky.Cookie != nil {
		config.Sticky.Cookie.Name = cookie.GetName(config.Sticky.Cookie.Name, serviceName)
	}

	balancer := wrr.New(config.Sticky)
	for _, service := range config.Services {
		serviceHandler, err := m.BuildHTTP(ctx, service.Name)
		if err != nil {
			return nil, err
		}

		balancer.AddService(service.Name, serviceHandler, service.Weight)
	}
	return balancer, nil
}

func (m *Manager) getLoadBalancerServiceHandler(ctx context.Context, serviceName string, service *dynamic.ServersLoadBalancer) (http.Handler, error) {
	if service.PassHostHeader == nil {
		defaultPassHostHeader := true
		service.PassHostHeader = &defaultPassHostHeader
	}

	if len(service.ServersTransport) > 0 {
		service.ServersTransport = provider.GetQualifiedName(ctx, service.ServersTransport)
	}

	roundTripper, err := m.roundTripperManager.Get(service.ServersTransport)
	if err != nil {
		return nil, err
	}

	fwd, err := buildProxy(service.PassHostHeader, service.ResponseForwarding, roundTripper, m.bufferPool)
	if err != nil {
		return nil, err
	}

	alHandler := func(next http.Handler) (http.Handler, error) {
		return accesslog.NewFieldHandler(next, accesslog.ServiceName, serviceName, accesslog.AddServiceFields), nil
	}
	chain := alice.New()
	if m.metricsRegistry != nil && m.metricsRegistry.IsSvcEnabled() {
		chain = chain.Append(metricsMiddle.WrapServiceHandler(ctx, m.metricsRegistry, serviceName))
	}

	handler, err := chain.Append(alHandler).Then(pipelining.New(ctx, fwd, "pipelining"))
	if err != nil {
		return nil, err
	}

	balancer, err := m.getLoadBalancer(ctx, serviceName, service, handler)
	if err != nil {
		return nil, err
	}

	// TODO rename and checks
	m.balancers[serviceName] = append(m.balancers[serviceName], balancer)

	// Empty (backend with no servers)
	return emptybackendhandler.New(balancer), nil
}

// LaunchHealthCheck Launches the health checks.
func (m *Manager) LaunchHealthCheck() {
	backendConfigs := make(map[string]*healthcheck.BackendConfig)

	for serviceName, balancers := range m.balancers {
		ctx := log.With(context.Background(), log.Str(log.ServiceName, serviceName))

		// TODO Should all the services handle healthcheck? Handle different types
		service := m.configs[serviceName].LoadBalancer

		// Health Check
		var backendHealthCheck *healthcheck.BackendConfig
		if hcOpts := buildHealthCheckOptions(ctx, balancers, serviceName, service.HealthCheck); hcOpts != nil {
			log.FromContext(ctx).Debugf("Setting up healthcheck for service %s with %s", serviceName, *hcOpts)

			hcOpts.Transport, _ = m.roundTripperManager.Get(service.ServersTransport)
			backendHealthCheck = healthcheck.NewBackendConfig(*hcOpts, serviceName)
		}

		if backendHealthCheck != nil {
			backendConfigs[serviceName] = backendHealthCheck
		}
	}

	healthcheck.GetHealthCheck(m.metricsRegistry).SetBackendsConfiguration(context.Background(), backendConfigs)
}

func buildHealthCheckOptions(ctx context.Context, lb healthcheck.Balancer, backend string, hc *dynamic.HealthCheck) *healthcheck.Options {
	if hc == nil || hc.Path == "" {
		return nil
	}

	logger := log.FromContext(ctx)

	interval := defaultHealthCheckInterval
	if hc.Interval != "" {
		intervalOverride, err := time.ParseDuration(hc.Interval)
		switch {
		case err != nil:
			logger.Errorf("Illegal health check interval for '%s': %s", backend, err)
		case intervalOverride <= 0:
			logger.Errorf("Health check interval smaller than zero for service '%s'", backend)
		default:
			interval = intervalOverride
		}
	}

	timeout := defaultHealthCheckTimeout
	if hc.Timeout != "" {
		timeoutOverride, err := time.ParseDuration(hc.Timeout)
		switch {
		case err != nil:
			logger.Errorf("Illegal health check timeout for backend '%s': %s", backend, err)
		case timeoutOverride <= 0:
			logger.Errorf("Health check timeout smaller than zero for backend '%s', backend", backend)
		default:
			timeout = timeoutOverride
		}
	}

	if timeout >= interval {
		logger.Warnf("Health check timeout for backend '%s' should be lower than the health check interval. Interval set to timeout + 1 second (%s).", backend, interval)
	}

	followRedirects := true
	if hc.FollowRedirects != nil {
		followRedirects = *hc.FollowRedirects
	}

	return &healthcheck.Options{
		Scheme:          hc.Scheme,
		Path:            hc.Path,
		Port:            hc.Port,
		Interval:        interval,
		Timeout:         timeout,
		LB:              lb,
		Hostname:        hc.Hostname,
		Headers:         hc.Headers,
		FollowRedirects: followRedirects,
	}
}

func (m *Manager) getLoadBalancer(ctx context.Context, serviceName string, service *dynamic.ServersLoadBalancer, fwd http.Handler) (healthcheck.BalancerHandler, error) {
	logger := log.FromContext(ctx)
	logger.Debug("Creating load-balancer")

	var options []roundrobin.LBOption

	var cookieName string
	if service.Sticky != nil && service.Sticky.Cookie != nil {
		cookieName = cookie.GetName(service.Sticky.Cookie.Name, serviceName)

		opts := roundrobin.CookieOptions{
			HTTPOnly: service.Sticky.Cookie.HTTPOnly,
			Secure:   service.Sticky.Cookie.Secure,
			SameSite: convertSameSite(service.Sticky.Cookie.SameSite),
		}

		// Sticky Cookie Value
		cv, err := stickycookie.NewFallbackValue(&stickycookie.RawValue{}, &stickycookie.HashValue{})
		if err != nil {
			return nil, err
		}

		options = append(options, roundrobin.EnableStickySession(roundrobin.NewStickySessionWithOptions(cookieName, opts).SetCookieValue(cv)))

		logger.Debugf("Sticky session cookie name: %v", cookieName)
	}

	lb, err := roundrobin.New(fwd, options...)
	if err != nil {
		return nil, err
	}

	lbsu := healthcheck.NewLBStatusUpdater(lb, m.configs[serviceName])
	if err := m.upsertServers(ctx, lbsu, service.Servers); err != nil {
		return nil, fmt.Errorf("error configuring load balancer for service %s: %w", serviceName, err)
	}

	return lbsu, nil
}

func (m *Manager) upsertServers(ctx context.Context, lb healthcheck.BalancerHandler, servers []dynamic.Server) error {
	logger := log.FromContext(ctx)

	for name, srv := range servers {
		u, err := url.Parse(srv.URL)
		if err != nil {
			return fmt.Errorf("error parsing server URL %s: %w", srv.URL, err)
		}

		logger.WithField(log.ServerName, name).Debugf("Creating server %d %s", name, u)

		if err := lb.UpsertServer(u, roundrobin.Weight(1)); err != nil {
			return fmt.Errorf("error adding server %s to load balancer: %w", srv.URL, err)
		}

		// FIXME Handle Metrics
	}
	return nil
}

func convertSameSite(sameSite string) http.SameSite {
	switch sameSite {
	case "none":
		return http.SameSiteNoneMode
	case "lax":
		return http.SameSiteLaxMode
	case "strict":
		return http.SameSiteStrictMode
	default:
		return 0
	}
}
