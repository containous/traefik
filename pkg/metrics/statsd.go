package metrics

import (
	"context"
	"time"

	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics/statsd"
	"github.com/traefik/traefik/v2/pkg/log"
	"github.com/traefik/traefik/v2/pkg/safe"
	"github.com/traefik/traefik/v2/pkg/types"
)

var (
	statsdClient *statsd.Statsd
	statsdTicker *time.Ticker
)

const (
	statsdConfigReloadsName           = "config.reload.total"
	statsdConfigReloadsFailureName    = statsdConfigReloadsName + ".failure"
	statsdLastConfigReloadSuccessName = "config.reload.lastSuccessTimestamp"
	statsdLastConfigReloadFailureName = "config.reload.lastFailureTimestamp"

	statsdTLSCertsNotAfterTimestampName = "tls.certs.notAfterTimestamp"

	statsdEntryPointReqsName        = "entrypoint.request.total"
	statsdEntryPointReqsTLSName     = "entrypoint.request.tls.total"
	statsdEntryPointReqDurationName = "entrypoint.request.duration"
	statsdEntryPointOpenConnsName   = "entrypoint.connections.open"

	statsdRouterReqsName         = "router.request.total"
	statsdRouterReqsTLSName      = "router.request.tls.total"
	statsdRouterReqsDurationName = "router.request.duration"
	statsdRouterOpenConnsName    = "router.connections.open"

	statsdServiceReqsName         = "service.request.total"
	statsdServiceReqsTLSName      = "service.request.tls.total"
	statsdServiceReqsDurationName = "service.request.duration"
	statsdServiceRetriesTotalName = "service.retries.total"
	statsdServiceServerUpName     = "service.server.up"
	statsdServiceOpenConnsName    = "service.connections.open"
)

// RegisterStatsd registers the metrics pusher if this didn't happen yet and creates a statsd Registry instance.
func RegisterStatsd(ctx context.Context, config *types.Statsd) Registry {
	// just to be sure there is a prefix defined
	if config.Prefix == "" {
		config.Prefix = "traefik"
	}

	statsdClient = statsd.New(config.Prefix+".", kitlog.LoggerFunc(func(keyvals ...interface{}) error {
		log.WithoutContext().WithField(log.MetricsProviderName, "statsd").Info(keyvals)
		return nil
	}))

	if statsdTicker == nil {
		statsdTicker = initStatsdTicker(ctx, config)
	}

	registry := &standardRegistry{
		configReloadsCounter:           statsdClient.NewCounter(statsdConfigReloadsName, 1.0),
		configReloadsFailureCounter:    statsdClient.NewCounter(statsdConfigReloadsFailureName, 1.0),
		lastConfigReloadSuccessGauge:   statsdClient.NewGauge(statsdLastConfigReloadSuccessName),
		lastConfigReloadFailureGauge:   statsdClient.NewGauge(statsdLastConfigReloadFailureName),
		tlsCertsNotAfterTimestampGauge: statsdClient.NewGauge(statsdTLSCertsNotAfterTimestampName),
	}

	if config.AddEntryPointsLabels {
		registry.epEnabled = config.AddEntryPointsLabels
		registry.entryPointReqsCounter = statsdClient.NewCounter(statsdEntryPointReqsName, 1.0)
		registry.entryPointReqsTLSCounter = statsdClient.NewCounter(statsdEntryPointReqsTLSName, 1.0)
		registry.entryPointReqDurationHistogram, _ = NewHistogramWithScale(statsdClient.NewTiming(statsdEntryPointReqDurationName, 1.0), time.Millisecond)
		registry.entryPointOpenConnsGauge = statsdClient.NewGauge(statsdEntryPointOpenConnsName)
	}

	if config.AddRoutersLabels {
		registry.routerEnabled = config.AddRoutersLabels
		registry.routerReqsCounter = statsdClient.NewCounter(statsdRouterReqsName, 1.0)
		registry.routerReqsTLSCounter = statsdClient.NewCounter(statsdRouterReqsTLSName, 1.0)
		registry.routerReqDurationHistogram, _ = NewHistogramWithScale(statsdClient.NewTiming(statsdRouterReqsDurationName, 1.0), time.Millisecond)
		registry.routerOpenConnsGauge = statsdClient.NewGauge(statsdRouterOpenConnsName)
	}

	if config.AddServicesLabels {
		registry.svcEnabled = config.AddServicesLabels
		registry.serviceReqsCounter = statsdClient.NewCounter(statsdServiceReqsName, 1.0)
		registry.serviceReqsTLSCounter = statsdClient.NewCounter(statsdServiceReqsTLSName, 1.0)
		registry.serviceReqDurationHistogram, _ = NewHistogramWithScale(statsdClient.NewTiming(statsdServiceReqsDurationName, 1.0), time.Millisecond)
		registry.serviceRetriesCounter = statsdClient.NewCounter(statsdServiceRetriesTotalName, 1.0)
		registry.serviceOpenConnsGauge = statsdClient.NewGauge(statsdServiceOpenConnsName)
		registry.serviceServerUpGauge = statsdClient.NewGauge(statsdServiceServerUpName)
	}

	return registry
}

// initStatsdTicker initializes metrics pusher and creates a statsdClient if not created already.
func initStatsdTicker(ctx context.Context, config *types.Statsd) *time.Ticker {
	address := config.Address
	if len(address) == 0 {
		address = "localhost:8125"
	}

	report := time.NewTicker(time.Duration(config.PushInterval))

	safe.Go(func() {
		statsdClient.SendLoop(ctx, report.C, "udp", address)
	})

	return report
}

// StopStatsd stops internal statsdTicker which controls the pushing of metrics to StatsD Agent and resets it to `nil`.
func StopStatsd() {
	if statsdTicker != nil {
		statsdTicker.Stop()
	}
	statsdTicker = nil
}
