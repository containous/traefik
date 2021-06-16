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

func TestDatadog(t *testing.T) {
	udp.SetAddr(":18125")
	// This is needed to make sure that UDP Listener listens for data a bit longer, otherwise it will quit after a millisecond
	udp.Timeout = 5 * time.Second

	datadogRegistry := RegisterDatadog(context.Background(), &types.Datadog{Address: ":18125", PushInterval: ptypes.Duration(time.Second), AddEntryPointsLabels: true, AddRoutersLabels: true, AddServicesLabels: true})
	defer StopDatadog()

	if !datadogRegistry.IsEpEnabled() || !datadogRegistry.IsRouterEnabled() || !datadogRegistry.IsSvcEnabled() {
		t.Errorf("DatadogRegistry should return true for IsEnabled(), IsRouterEnabled() and IsSvcEnabled()")
	}

	expected := []string{
		"traefik.config.reload.total:1.000000|c\n",
		"traefik.config.reload.total:1.000000|c|#failure:true\n",
		"traefik.config.reload.lastSuccessTimestamp:1.000000|g\n",
		"traefik.config.reload.lastFailureTimestamp:1.000000|g\n",

		"traefik.tls.certs.notAfterTimestamp:1.000000|g|#key:value\n",

		"traefik.entrypoint.request.total:1.000000|c|#entrypoint:test\n",
		"traefik.entrypoint.request.tls.total:1.000000|c|#entrypoint:test,tls_version:foo,tls_cipher:bar\n",
		"traefik.entrypoint.request.duration:10000.000000|h|#entrypoint:test\n",
		"traefik.entrypoint.connections.open:1.000000|g|#entrypoint:test\n",

		"traefik.router.request.total:1.000000|c|#router:demo,service:test,code:404,method:GET\n",
		"traefik.router.request.total:1.000000|c|#router:demo,service:test,code:200,method:GET\n",
		"traefik.router.request.tls.total:1.000000|c|#router:demo,service:test,tls_version:foo,tls_cipher:bar\n",
		"traefik.router.request.duration:10000.000000|h|#router:demo,service:test,code:200\n",
		"traefik.router.connections.open:1.000000|g|#router:demo,service:test\n",

		"traefik.service.request.total:1.000000|c|#service:test,code:404,method:GET\n",
		"traefik.service.request.total:1.000000|c|#service:test,code:200,method:GET\n",
		"traefik.service.request.tls.total:1.000000|c|#service:test,tls_version:foo,tls_cipher:bar\n",
		"traefik.service.request.duration:10000.000000|h|#service:test,code:200\n",
		"traefik.service.connections.open:1.000000|g|#service:test\n",
		"traefik.service.retries.total:2.000000|c|#service:test\n",
		"traefik.service.request.duration:10000.000000|h|#service:test,code:200\n",
		"traefik.service.server.up:1.000000|g|#service:test,url:http://127.0.0.1,one:two\n",
	}

	udp.ShouldReceiveAll(t, expected, func() {
		datadogRegistry.ConfigReloadsCounter().Add(1)
		datadogRegistry.ConfigReloadsFailureCounter().Add(1)
		datadogRegistry.LastConfigReloadSuccessGauge().Add(1)
		datadogRegistry.LastConfigReloadFailureGauge().Add(1)

		datadogRegistry.TLSCertsNotAfterTimestampGauge().With("key", "value").Set(1)

		datadogRegistry.EntryPointReqsCounter().With("entrypoint", "test").Add(1)
		datadogRegistry.EntryPointReqsTLSCounter().With("entrypoint", "test", "tls_version", "foo", "tls_cipher", "bar").Add(1)
		datadogRegistry.EntryPointReqDurationHistogram().With("entrypoint", "test").Observe(10000)
		datadogRegistry.EntryPointOpenConnsGauge().With("entrypoint", "test").Set(1)

		datadogRegistry.RouterReqsCounter().With("router", "demo", "service", "test", "code", strconv.Itoa(http.StatusOK), "method", http.MethodGet).Add(1)
		datadogRegistry.RouterReqsCounter().With("router", "demo", "service", "test", "code", strconv.Itoa(http.StatusNotFound), "method", http.MethodGet).Add(1)
		datadogRegistry.RouterReqsTLSCounter().With("router", "demo", "service", "test", "tls_version", "foo", "tls_cipher", "bar").Add(1)
		datadogRegistry.RouterReqDurationHistogram().With("router", "demo", "service", "test", "code", strconv.Itoa(http.StatusOK)).Observe(10000)
		datadogRegistry.RouterOpenConnsGauge().With("router", "demo", "service", "test").Set(1)

		datadogRegistry.ServiceReqsCounter().With("service", "test", "code", strconv.Itoa(http.StatusOK), "method", http.MethodGet).Add(1)
		datadogRegistry.ServiceReqsCounter().With("service", "test", "code", strconv.Itoa(http.StatusNotFound), "method", http.MethodGet).Add(1)
		datadogRegistry.ServiceReqsTLSCounter().With("service", "test", "tls_version", "foo", "tls_cipher", "bar").Add(1)
		datadogRegistry.ServiceReqDurationHistogram().With("service", "test", "code", strconv.Itoa(http.StatusOK)).Observe(10000)
		datadogRegistry.ServiceOpenConnsGauge().With("service", "test").Set(1)
		datadogRegistry.ServiceRetriesCounter().With("service", "test").Add(1)
		datadogRegistry.ServiceRetriesCounter().With("service", "test").Add(1)
		datadogRegistry.ServiceServerUpGauge().With("service", "test", "url", "http://127.0.0.1", "one", "two").Set(1)
	})
}
