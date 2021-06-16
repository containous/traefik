package metrics

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-kit/kit/metrics"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/log"
	"github.com/traefik/traefik/v2/pkg/safe"
	"github.com/traefik/traefik/v2/pkg/types"
)

const (
	// MetricNamePrefix prefix of all metric names.
	MetricNamePrefix = "traefik_"

	// server meta information.
	metricConfigPrefix             = MetricNamePrefix + "config_"
	configReloadsTotalName         = metricConfigPrefix + "reloads_total"
	configReloadsFailuresTotalName = metricConfigPrefix + "reloads_failure_total"
	configLastReloadSuccessName    = metricConfigPrefix + "last_reload_success"
	configLastReloadFailureName    = metricConfigPrefix + "last_reload_failure"

	// TLS.
	metricsTLSPrefix          = MetricNamePrefix + "tls_"
	tlsCertsNotAfterTimestamp = metricsTLSPrefix + "certs_not_after"

	// entry point.
	metricEntryPointPrefix     = MetricNamePrefix + "entrypoint_"
	entryPointReqsTotalName    = metricEntryPointPrefix + "requests_total"
	entryPointReqsTLSTotalName = metricEntryPointPrefix + "requests_tls_total"
	entryPointReqDurationName  = metricEntryPointPrefix + "request_duration_seconds"
	entryPointOpenConnsName    = metricEntryPointPrefix + "open_connections"

	// router level.
	metricRouterPrefix     = MetricNamePrefix + "router_"
	routerReqsTotalName    = metricRouterPrefix + "requests_total"
	routerReqsTLSTotalName = metricRouterPrefix + "requests_tls_total"
	routerReqDurationName  = metricRouterPrefix + "request_duration_seconds"
	routerOpenConnsName    = metricRouterPrefix + "open_connections"

	// service level.
	metricServicePrefix     = MetricNamePrefix + "service_"
	serviceReqsTotalName    = metricServicePrefix + "requests_total"
	serviceReqsTLSTotalName = metricServicePrefix + "requests_tls_total"
	serviceReqDurationName  = metricServicePrefix + "request_duration_seconds"
	serviceOpenConnsName    = metricServicePrefix + "open_connections"
	serviceRetriesTotalName = metricServicePrefix + "retries_total"
	serviceServerUpName     = metricServicePrefix + "server_up"
)

// promState holds all metric state internally and acts as the only Collector we register for Prometheus.
//
// This enables control to remove metrics that belong to outdated configuration.
// As an example why this is required, consider Traefik learns about a new service.
// It populates the 'traefik_server_service_up' metric for it with a value of 1 (alive).
// When the service is undeployed now the metric is still there in the client library
// and will be returned on the metrics endpoint until Traefik would be restarted.
//
// To solve this problem promState keeps track of Traefik's dynamic configuration.
// Metrics that "belong" to a dynamic configuration part like services or entryPoints
// are removed after they were scraped at least once when the corresponding object
// doesn't exist anymore.
var promState = newPrometheusState()

var promRegistry = stdprometheus.NewRegistry()

// PrometheusHandler exposes Prometheus routes.
func PrometheusHandler() http.Handler {
	return promhttp.HandlerFor(promRegistry, promhttp.HandlerOpts{})
}

// RegisterPrometheus registers all Prometheus metrics.
// It must be called only once and failing to register the metrics will lead to a panic.
func RegisterPrometheus(ctx context.Context, config *types.Prometheus) Registry {
	standardRegistry := initStandardRegistry(config)

	if err := promRegistry.Register(stdprometheus.NewProcessCollector(stdprometheus.ProcessCollectorOpts{})); err != nil {
		var arErr stdprometheus.AlreadyRegisteredError
		if !errors.As(err, &arErr) {
			log.FromContext(ctx).Warn("ProcessCollector is already registered")
		}
	}

	if err := promRegistry.Register(stdprometheus.NewGoCollector()); err != nil {
		var arErr stdprometheus.AlreadyRegisteredError
		if !errors.As(err, &arErr) {
			log.FromContext(ctx).Warn("GoCollector is already registered")
		}
	}

	if !registerPromState(ctx) {
		return nil
	}

	return standardRegistry
}

func initStandardRegistry(config *types.Prometheus) Registry {
	buckets := []float64{0.1, 0.3, 1.2, 5.0}
	if config.Buckets != nil {
		buckets = config.Buckets
	}

	safe.Go(func() {
		promState.ListenValueUpdates()
	})

	configReloads := newCounterFrom(promState.collectors, stdprometheus.CounterOpts{
		Name: configReloadsTotalName,
		Help: "Config reloads",
	}, []string{})
	configReloadsFailures := newCounterFrom(promState.collectors, stdprometheus.CounterOpts{
		Name: configReloadsFailuresTotalName,
		Help: "Config failure reloads",
	}, []string{})
	lastConfigReloadSuccess := newGaugeFrom(promState.collectors, stdprometheus.GaugeOpts{
		Name: configLastReloadSuccessName,
		Help: "Last config reload success",
	}, []string{})
	lastConfigReloadFailure := newGaugeFrom(promState.collectors, stdprometheus.GaugeOpts{
		Name: configLastReloadFailureName,
		Help: "Last config reload failure",
	}, []string{})
	tlsCertsNotAfterTimesptamp := newGaugeFrom(promState.collectors, stdprometheus.GaugeOpts{
		Name: tlsCertsNotAfterTimestamp,
		Help: "Certificate expiration timestamp",
	}, []string{"cn", "serial", "sans"})

	promState.describers = []func(chan<- *stdprometheus.Desc){
		configReloads.cv.Describe,
		configReloadsFailures.cv.Describe,
		lastConfigReloadSuccess.gv.Describe,
		lastConfigReloadFailure.gv.Describe,
		tlsCertsNotAfterTimesptamp.gv.Describe,
	}

	reg := &standardRegistry{
		epEnabled:                      config.AddEntryPointsLabels,
		routerEnabled:                  config.AddRoutersLabels,
		svcEnabled:                     config.AddServicesLabels,
		configReloadsCounter:           configReloads,
		configReloadsFailureCounter:    configReloadsFailures,
		lastConfigReloadSuccessGauge:   lastConfigReloadSuccess,
		lastConfigReloadFailureGauge:   lastConfigReloadFailure,
		tlsCertsNotAfterTimestampGauge: tlsCertsNotAfterTimesptamp,
	}

	if config.AddEntryPointsLabels {
		entryPointReqs := newCounterFrom(promState.collectors, stdprometheus.CounterOpts{
			Name: entryPointReqsTotalName,
			Help: "How many HTTP requests processed on an entrypoint, partitioned by status code, protocol, and method.",
		}, []string{"code", "method", "protocol", "entrypoint"})
		entryPointReqsTLS := newCounterFrom(promState.collectors, stdprometheus.CounterOpts{
			Name: entryPointReqsTLSTotalName,
			Help: "How many HTTP requests with TLS processed on an entrypoint, partitioned by TLS Version and TLS cipher Used.",
		}, []string{"tls_version", "tls_cipher", "entrypoint"})
		entryPointReqDurations := newHistogramFrom(promState.collectors, stdprometheus.HistogramOpts{
			Name:    entryPointReqDurationName,
			Help:    "How long it took to process the request on an entrypoint, partitioned by status code, protocol, and method.",
			Buckets: buckets,
		}, []string{"code", "method", "protocol", "entrypoint"})
		entryPointOpenConns := newGaugeFrom(promState.collectors, stdprometheus.GaugeOpts{
			Name: entryPointOpenConnsName,
			Help: "How many open connections exist on an entrypoint, partitioned by method and protocol.",
		}, []string{"method", "protocol", "entrypoint"})

		promState.describers = append(promState.describers, []func(chan<- *stdprometheus.Desc){
			entryPointReqs.cv.Describe,
			entryPointReqsTLS.cv.Describe,
			entryPointReqDurations.hv.Describe,
			entryPointOpenConns.gv.Describe,
		}...)

		reg.entryPointReqsCounter = entryPointReqs
		reg.entryPointReqsTLSCounter = entryPointReqsTLS
		reg.entryPointReqDurationHistogram, _ = NewHistogramWithScale(entryPointReqDurations, time.Second)
		reg.entryPointOpenConnsGauge = entryPointOpenConns
	}

	if config.AddRoutersLabels {
		routerReqs := newCounterFrom(promState.collectors, stdprometheus.CounterOpts{
			Name: routerReqsTotalName,
			Help: "How many HTTP requests are processed on a router, partitioned by service, status code, protocol, and method.",
		}, []string{"code", "method", "protocol", "router", "service"})
		routerReqsTLS := newCounterFrom(promState.collectors, stdprometheus.CounterOpts{
			Name: routerReqsTLSTotalName,
			Help: "How many HTTP requests with TLS are processed on a router, partitioned by service, TLS Version, and TLS cipher Used.",
		}, []string{"tls_version", "tls_cipher", "router", "service"})
		routerReqDurations := newHistogramFrom(promState.collectors, stdprometheus.HistogramOpts{
			Name:    routerReqDurationName,
			Help:    "How long it took to process the request on a router, partitioned by service, status code, protocol, and method.",
			Buckets: buckets,
		}, []string{"code", "method", "protocol", "router", "service"})
		routerOpenConns := newGaugeFrom(promState.collectors, stdprometheus.GaugeOpts{
			Name: routerOpenConnsName,
			Help: "How many open connections exist on a router, partitioned by service, method, and protocol.",
		}, []string{"method", "protocol", "router", "service"})

		promState.describers = append(promState.describers, []func(chan<- *stdprometheus.Desc){
			routerReqs.cv.Describe,
			routerReqsTLS.cv.Describe,
			routerReqDurations.hv.Describe,
			routerOpenConns.gv.Describe,
		}...)
		reg.routerReqsCounter = routerReqs
		reg.routerReqsTLSCounter = routerReqsTLS
		reg.routerReqDurationHistogram, _ = NewHistogramWithScale(routerReqDurations, time.Second)
		reg.routerOpenConnsGauge = routerOpenConns
	}

	if config.AddServicesLabels {
		serviceReqs := newCounterFrom(promState.collectors, stdprometheus.CounterOpts{
			Name: serviceReqsTotalName,
			Help: "How many HTTP requests processed on a service, partitioned by status code, protocol, and method.",
		}, []string{"code", "method", "protocol", "service"})
		serviceReqsTLS := newCounterFrom(promState.collectors, stdprometheus.CounterOpts{
			Name: serviceReqsTLSTotalName,
			Help: "How many HTTP requests with TLS processed on a service, partitioned by TLS version and TLS cipher.",
		}, []string{"tls_version", "tls_cipher", "service"})
		serviceReqDurations := newHistogramFrom(promState.collectors, stdprometheus.HistogramOpts{
			Name:    serviceReqDurationName,
			Help:    "How long it took to process the request on a service, partitioned by status code, protocol, and method.",
			Buckets: buckets,
		}, []string{"code", "method", "protocol", "service"})
		serviceOpenConns := newGaugeFrom(promState.collectors, stdprometheus.GaugeOpts{
			Name: serviceOpenConnsName,
			Help: "How many open connections exist on a service, partitioned by method and protocol.",
		}, []string{"method", "protocol", "service"})
		serviceRetries := newCounterFrom(promState.collectors, stdprometheus.CounterOpts{
			Name: serviceRetriesTotalName,
			Help: "How many request retries happened on a service.",
		}, []string{"service"})
		serviceServerUp := newGaugeFrom(promState.collectors, stdprometheus.GaugeOpts{
			Name: serviceServerUpName,
			Help: "service server is up, described by gauge value of 0 or 1.",
		}, []string{"service", "url"})

		promState.describers = append(promState.describers, []func(chan<- *stdprometheus.Desc){
			serviceReqs.cv.Describe,
			serviceReqsTLS.cv.Describe,
			serviceReqDurations.hv.Describe,
			serviceOpenConns.gv.Describe,
			serviceRetries.cv.Describe,
			serviceServerUp.gv.Describe,
		}...)

		reg.serviceReqsCounter = serviceReqs
		reg.serviceReqsTLSCounter = serviceReqsTLS
		reg.serviceReqDurationHistogram, _ = NewHistogramWithScale(serviceReqDurations, time.Second)
		reg.serviceOpenConnsGauge = serviceOpenConns
		reg.serviceRetriesCounter = serviceRetries
		reg.serviceServerUpGauge = serviceServerUp
	}

	return reg
}

func registerPromState(ctx context.Context) bool {
	err := promRegistry.Register(promState)
	if err == nil {
		return true
	}

	logger := log.FromContext(ctx)

	var arErr stdprometheus.AlreadyRegisteredError
	if errors.As(err, &arErr) {
		logger.Debug("Prometheus collector already registered.")
		return true
	}

	logger.Errorf("Unable to register Traefik to Prometheus: %v", err)
	return false
}

// OnConfigurationUpdate receives the current configuration from Traefik.
// It then converts the configuration to the optimized package internal format
// and sets it to the promState.
func OnConfigurationUpdate(conf dynamic.Configuration, entryPoints []string) {
	dynamicConfig := newDynamicConfig()

	for _, value := range entryPoints {
		dynamicConfig.entryPoints[value] = true
	}

	for name := range conf.HTTP.Routers {
		dynamicConfig.routers[name] = true
	}

	for serviceName, service := range conf.HTTP.Services {
		dynamicConfig.services[serviceName] = make(map[string]bool)
		if service.LoadBalancer != nil {
			for _, server := range service.LoadBalancer.Servers {
				dynamicConfig.services[serviceName][server.URL] = true
			}
		}
	}

	promState.SetDynamicConfig(dynamicConfig)
}

func newPrometheusState() *prometheusState {
	return &prometheusState{
		collectors:    make(chan *collector),
		dynamicConfig: newDynamicConfig(),
		state:         make(map[string]*collector),
	}
}

type prometheusState struct {
	collectors chan *collector
	describers []func(ch chan<- *stdprometheus.Desc)

	mtx           sync.Mutex
	dynamicConfig *dynamicConfig
	state         map[string]*collector
}

func (ps *prometheusState) SetDynamicConfig(dynamicConfig *dynamicConfig) {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()
	ps.dynamicConfig = dynamicConfig
}

func (ps *prometheusState) ListenValueUpdates() {
	for collector := range ps.collectors {
		ps.mtx.Lock()
		ps.state[collector.id] = collector
		ps.mtx.Unlock()
	}
}

// Describe implements prometheus.Collector and simply calls
// the registered describer functions.
func (ps *prometheusState) Describe(ch chan<- *stdprometheus.Desc) {
	for _, desc := range ps.describers {
		desc(ch)
	}
}

// Collect implements prometheus.Collector. It calls the Collect
// method of all metrics it received on the collectors channel.
// It's also responsible to remove metrics that belong to an outdated configuration.
// The removal happens only after their Collect method was called to ensure that
// also those metrics will be exported on the current scrape.
func (ps *prometheusState) Collect(ch chan<- stdprometheus.Metric) {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()

	var outdatedKeys []string
	for key, cs := range ps.state {
		cs.collector.Collect(ch)

		if ps.isOutdated(cs) {
			outdatedKeys = append(outdatedKeys, key)
		}
	}

	for _, key := range outdatedKeys {
		ps.state[key].delete()
		delete(ps.state, key)
	}
}

// isOutdated checks whether the passed collector has labels that mark
// it as belonging to an outdated configuration of Traefik.
func (ps *prometheusState) isOutdated(collector *collector) bool {
	labels := collector.labels

	if entrypointName, ok := labels["entrypoint"]; ok && !ps.dynamicConfig.hasEntryPoint(entrypointName) {
		return true
	}

	if serviceName, ok := labels["service"]; ok {
		if !ps.dynamicConfig.hasService(serviceName) {
			return true
		}
		if url, ok := labels["url"]; ok && !ps.dynamicConfig.hasServerURL(serviceName, url) {
			return true
		}
	}

	return false
}

func newDynamicConfig() *dynamicConfig {
	return &dynamicConfig{
		entryPoints: make(map[string]bool),
		routers:     make(map[string]bool),
		services:    make(map[string]map[string]bool),
	}
}

// dynamicConfig holds the current configuration for entryPoints, services,
// and server URLs in an optimized way to check for existence. This provides
// a performant way to check whether the collected metrics belong to the
// current configuration or to an outdated one.
type dynamicConfig struct {
	entryPoints map[string]bool
	routers     map[string]bool
	services    map[string]map[string]bool
}

func (d *dynamicConfig) hasEntryPoint(entrypointName string) bool {
	_, ok := d.entryPoints[entrypointName]
	return ok
}

func (d *dynamicConfig) hasService(serviceName string) bool {
	_, ok := d.services[serviceName]
	return ok
}

func (d *dynamicConfig) hasServerURL(serviceName, serverURL string) bool {
	if service, hasService := d.services[serviceName]; hasService {
		_, ok := service[serverURL]
		return ok
	}
	return false
}

func newCollector(metricName string, labels stdprometheus.Labels, c stdprometheus.Collector, deleteFn func()) *collector {
	return &collector{
		id:        buildMetricID(metricName, labels),
		labels:    labels,
		collector: c,
		delete:    deleteFn,
	}
}

// collector wraps a Collector object from the Prometheus client library.
// It adds information on how many generations this metric should be present
// in the /metrics output, relative to the time it was last tracked.
type collector struct {
	id        string
	labels    stdprometheus.Labels
	collector stdprometheus.Collector
	delete    func()
}

func buildMetricID(metricName string, labels stdprometheus.Labels) string {
	var labelNamesValues []string
	for name, value := range labels {
		labelNamesValues = append(labelNamesValues, name, value)
	}
	sort.Strings(labelNamesValues)
	return metricName + ":" + strings.Join(labelNamesValues, "|")
}

func newCounterFrom(collectors chan<- *collector, opts stdprometheus.CounterOpts, labelNames []string) *counter {
	cv := stdprometheus.NewCounterVec(opts, labelNames)
	c := &counter{
		name:       opts.Name,
		cv:         cv,
		collectors: collectors,
	}
	if len(labelNames) == 0 {
		c.Add(0)
	}
	return c
}

type counter struct {
	name             string
	cv               *stdprometheus.CounterVec
	labelNamesValues labelNamesValues
	collectors       chan<- *collector
}

func (c *counter) With(labelValues ...string) metrics.Counter {
	return &counter{
		name:             c.name,
		cv:               c.cv,
		labelNamesValues: c.labelNamesValues.With(labelValues...),
		collectors:       c.collectors,
	}
}

func (c *counter) Add(delta float64) {
	labels := c.labelNamesValues.ToLabels()
	collector := c.cv.With(labels)
	collector.Add(delta)
	c.collectors <- newCollector(c.name, labels, collector, func() {
		c.cv.Delete(labels)
	})
}

func (c *counter) Describe(ch chan<- *stdprometheus.Desc) {
	c.cv.Describe(ch)
}

func newGaugeFrom(collectors chan<- *collector, opts stdprometheus.GaugeOpts, labelNames []string) *gauge {
	gv := stdprometheus.NewGaugeVec(opts, labelNames)
	g := &gauge{
		name:       opts.Name,
		gv:         gv,
		collectors: collectors,
	}
	if len(labelNames) == 0 {
		g.Set(0)
	}
	return g
}

type gauge struct {
	name             string
	gv               *stdprometheus.GaugeVec
	labelNamesValues labelNamesValues
	collectors       chan<- *collector
}

func (g *gauge) With(labelValues ...string) metrics.Gauge {
	return &gauge{
		name:             g.name,
		gv:               g.gv,
		labelNamesValues: g.labelNamesValues.With(labelValues...),
		collectors:       g.collectors,
	}
}

func (g *gauge) Add(delta float64) {
	labels := g.labelNamesValues.ToLabels()
	collector := g.gv.With(labels)
	collector.Add(delta)
	g.collectors <- newCollector(g.name, labels, collector, func() {
		g.gv.Delete(labels)
	})
}

func (g *gauge) Set(value float64) {
	labels := g.labelNamesValues.ToLabels()
	collector := g.gv.With(labels)
	collector.Set(value)
	g.collectors <- newCollector(g.name, labels, collector, func() {
		g.gv.Delete(labels)
	})
}

func (g *gauge) Describe(ch chan<- *stdprometheus.Desc) {
	g.gv.Describe(ch)
}

func newHistogramFrom(collectors chan<- *collector, opts stdprometheus.HistogramOpts, labelNames []string) *histogram {
	hv := stdprometheus.NewHistogramVec(opts, labelNames)
	return &histogram{
		name:       opts.Name,
		hv:         hv,
		collectors: collectors,
	}
}

type histogram struct {
	name             string
	hv               *stdprometheus.HistogramVec
	labelNamesValues labelNamesValues
	collectors       chan<- *collector
}

func (h *histogram) With(labelValues ...string) metrics.Histogram {
	return &histogram{
		name:             h.name,
		hv:               h.hv,
		labelNamesValues: h.labelNamesValues.With(labelValues...),
		collectors:       h.collectors,
	}
}

func (h *histogram) Observe(value float64) {
	labels := h.labelNamesValues.ToLabels()
	observer := h.hv.With(labels)
	observer.Observe(value)
	// Do a type assertion to be sure that prometheus will be able to call the Collect method.
	if collector, ok := observer.(stdprometheus.Histogram); ok {
		h.collectors <- newCollector(h.name, labels, collector, func() {
			h.hv.Delete(labels)
		})
	}
}

func (h *histogram) Describe(ch chan<- *stdprometheus.Desc) {
	h.hv.Describe(ch)
}

// labelNamesValues is a type alias that provides validation on its With method.
// Metrics may include it as a member to help them satisfy With semantics and
// save some code duplication.
type labelNamesValues []string

// With validates the input, and returns a new aggregate labelNamesValues.
func (lvs labelNamesValues) With(labelValues ...string) labelNamesValues {
	if len(labelValues)%2 != 0 {
		labelValues = append(labelValues, "unknown")
	}
	return append(lvs, labelValues...)
}

// ToLabels is a convenience method to convert a labelNamesValues
// to the native prometheus.Labels.
func (lvs labelNamesValues) ToLabels() stdprometheus.Labels {
	labels := stdprometheus.Labels{}
	for i := 0; i < len(lvs); i += 2 {
		labels[lvs[i]] = lvs[i+1]
	}
	return labels
}
