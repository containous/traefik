package metrics

import (
	"context"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stvp/go-udp-testing"
	ptypes "github.com/traefik/paerser/types"
	"github.com/traefik/traefik/v2/pkg/types"
)

func TestStatsD(t *testing.T) {
	t.Cleanup(func() {
		StopStatsd()
	})

	udp.SetAddr(":18125")
	// This is needed to make sure that UDP Listener listens for data a bit longer, otherwise it will quit after a millisecond
	udp.Timeout = 5 * time.Second

	statsdRegistry := RegisterStatsd(context.Background(), &types.Statsd{Address: ":18125", PushInterval: ptypes.Duration(time.Second), AddEntryPointsLabels: true, AddRoutersLabels: true, AddServicesLabels: true})

	testRegistry(t, "", statsdRegistry)
}

func TestStatsDWithPrefix(t *testing.T) {
	t.Cleanup(func() {
		StopStatsd()
	})

	udp.SetAddr(":18125")
	// This is needed to make sure that UDP Listener listens for data a bit longer, otherwise it will quit after a millisecond
	udp.Timeout = 5 * time.Second

	statsdRegistry := RegisterStatsd(context.Background(), &types.Statsd{Address: ":18125", PushInterval: ptypes.Duration(time.Second), AddEntryPointsLabels: true, AddRoutersLabels: true, AddServicesLabels: true, Prefix: "testPrefix"})

	testRegistry(t, "testPrefix", statsdRegistry)
}

func testRegistry(t *testing.T, metricsPrefix string, registry Registry) {
	t.Helper()

	if !registry.IsEpEnabled() || !registry.IsRouterEnabled() || !registry.IsSvcEnabled() {
		t.Errorf("Statsd registry should return true for IsEnabled(), IsRouterEnabled() and IsSvcEnabled()")
	}

	if metricsPrefix == "" {
		metricsPrefix = "traefik"
	}

	expected := []string{
		metricsPrefix + ".config.reload.total:1.000000|c\n",
		metricsPrefix + ".config.reload.total.failure:1.000000|c\n",
		metricsPrefix + ".config.reload.lastSuccessTimestamp:1.000000|g\n",
		metricsPrefix + ".config.reload.lastFailureTimestamp:1.000000|g\n",

		metricsPrefix + ".tls.certs.notAfterTimestamp:1.000000|g\n",

		metricsPrefix + ".entrypoint.request.total:1.000000|c\n",
		metricsPrefix + ".entrypoint.request.tls.total:1.000000|c\n",
		metricsPrefix + ".entrypoint.request.duration:10000.000000|ms",
		metricsPrefix + ".entrypoint.connections.open:1.000000|g\n",

		metricsPrefix + ".router.request.total:2.000000|c\n",
		metricsPrefix + ".router.request.tls.total:1.000000|c\n",
		metricsPrefix + ".router.request.duration:10000.000000|ms",
		metricsPrefix + ".router.connections.open:1.000000|g\n",

		metricsPrefix + ".service.request.total:2.000000|c\n",
		metricsPrefix + ".service.request.tls.total:1.000000|c\n",
		metricsPrefix + ".service.request.duration:10000.000000|ms",
		metricsPrefix + ".service.connections.open:1.000000|g\n",
		metricsPrefix + ".service.retries.total:2.000000|c\n",
		metricsPrefix + ".service.server.up:1.000000|g\n",
	}

	udp.ShouldReceiveAll(t, expected, func() {
		registry.ConfigReloadsCounter().Add(1)
		registry.ConfigReloadsFailureCounter().Add(1)
		registry.LastConfigReloadSuccessGauge().Set(1)
		registry.LastConfigReloadFailureGauge().Set(1)

		registry.TLSCertsNotAfterTimestampGauge().With("key", "value").Set(1)

		registry.EntryPointReqsCounter().With("entrypoint", "test", "code", strconv.Itoa(http.StatusOK), "method", http.MethodGet).Add(1)
		registry.EntryPointReqsTLSCounter().With("entrypoint", "test", "tls_version", "foo", "tls_cipher", "bar").Add(1)
		registry.EntryPointReqDurationHistogram().With("entrypoint", "test").Observe(10000)
		registry.EntryPointOpenConnsGauge().With("entrypoint", "test").Set(1)

		registry.RouterReqsCounter().With("router", "demo", "service", "test", "code", strconv.Itoa(http.StatusNotFound), "method", http.MethodGet).Add(1)
		registry.RouterReqsCounter().With("router", "demo", "service", "test", "code", strconv.Itoa(http.StatusOK), "method", http.MethodGet).Add(1)
		registry.RouterReqsTLSCounter().With("router", "demo", "service", "test", "tls_version", "foo", "tls_cipher", "bar").Add(1)
		registry.RouterReqDurationHistogram().With("router", "demo", "service", "test", "code", strconv.Itoa(http.StatusOK)).Observe(10000)
		registry.RouterOpenConnsGauge().With("router", "demo", "service", "test").Set(1)

		registry.ServiceReqsCounter().With("service", "test", "code", strconv.Itoa(http.StatusOK), "method", http.MethodGet).Add(1)
		registry.ServiceReqsCounter().With("service", "test", "code", strconv.Itoa(http.StatusNotFound), "method", http.MethodGet).Add(1)
		registry.ServiceReqsTLSCounter().With("service", "test", "tls_version", "foo", "tls_cipher", "bar").Add(1)
		registry.ServiceReqDurationHistogram().With("service", "test", "code", strconv.Itoa(http.StatusOK)).Observe(10000)
		registry.ServiceOpenConnsGauge().With("service", "test").Set(1)
		registry.ServiceRetriesCounter().With("service", "test").Add(1)
		registry.ServiceRetriesCounter().With("service", "test").Add(1)
		registry.ServiceServerUpGauge().With("service:test", "url", "http://127.0.0.1").Set(1)
	})
}
