package metrics

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/stvp/go-udp-testing"
	ptypes "github.com/traefik/paerser/types"
	"github.com/traefik/traefik/v2/pkg/types"
)

func TestInfluxDB(t *testing.T) {
	udp.SetAddr(":8089")
	// This is needed to make sure that UDP Listener listens for data a bit longer, otherwise it will quit after a millisecond
	udp.Timeout = 5 * time.Second

	influxDBClient = nil
	influxDBRegistry := RegisterInfluxDB(context.Background(),
		&types.InfluxDB{
			Address:              ":8089",
			PushInterval:         ptypes.Duration(time.Second),
			AddEntryPointsLabels: true,
			AddRoutersLabels:     true,
			AddServicesLabels:    true,
			AdditionalLabels:     map[string]string{"tag1": "val1"},
		})
	defer StopInfluxDB()

	if !influxDBRegistry.IsEpEnabled() || !influxDBRegistry.IsRouterEnabled() || !influxDBRegistry.IsSvcEnabled() {
		t.Fatalf("InfluxDBRegistry  should return true for IsEnabled(), IsRouterEnabled() and IsSvcEnabled()")
	}

	expectedServer := []string{
		`(traefik\.config\.reload\.total,tag1=val1 count=1) [\d]{19}`,
		`(traefik\.config\.reload\.total\.failure,tag1=val1 count=1) [\d]{19}`,
		`(traefik\.config\.reload\.lastSuccessTimestamp,tag1=val1 value=1) [\d]{19}`,
		`(traefik\.config\.reload\.lastFailureTimestamp,tag1=val1 value=1) [\d]{19}`,
	}

	msgServer := udp.ReceiveString(t, func() {
		influxDBRegistry.ConfigReloadsCounter().Add(1)
		influxDBRegistry.ConfigReloadsFailureCounter().Add(1)
		influxDBRegistry.LastConfigReloadSuccessGauge().Set(1)
		influxDBRegistry.LastConfigReloadFailureGauge().Set(1)
	})

	assertMessage(t, msgServer, expectedServer)

	expectedTLS := []string{
		`(traefik\.tls\.certs\.notAfterTimestamp,key=value,tag1=val1 value=1) [\d]{19}`,
	}

	msgTLS := udp.ReceiveString(t, func() {
		influxDBRegistry.TLSCertsNotAfterTimestampGauge().With("key", "value").Set(1)
	})

	assertMessage(t, msgTLS, expectedTLS)

	expectedEntrypoint := []string{
		`(traefik\.entrypoint\.requests\.total,code=200,entrypoint=test,method=GET,tag1=val1 count=1) [\d]{19}`,
		`(traefik\.entrypoint\.requests\.tls\.total,entrypoint=test,tag1=val1,tls_cipher=bar,tls_version=foo count=1) [\d]{19}`,
		`(traefik\.entrypoint\.request\.duration(?:,code=[\d]{3})?,entrypoint=test,tag1=val1 p50=10000,p90=10000,p95=10000,p99=10000) [\d]{19}`,
		`(traefik\.entrypoint\.connections\.open,entrypoint=test,tag1=val1 value=1) [\d]{19}`,
	}

	msgEntrypoint := udp.ReceiveString(t, func() {
		influxDBRegistry.EntryPointReqsCounter().With("entrypoint", "test", "code", strconv.Itoa(http.StatusOK), "method", http.MethodGet).Add(1)
		influxDBRegistry.EntryPointReqsTLSCounter().With("entrypoint", "test", "tls_version", "foo", "tls_cipher", "bar").Add(1)
		influxDBRegistry.EntryPointReqDurationHistogram().With("entrypoint", "test").Observe(10000)
		influxDBRegistry.EntryPointOpenConnsGauge().With("entrypoint", "test").Set(1)
	})

	assertMessage(t, msgEntrypoint, expectedEntrypoint)

	expectedRouter := []string{
		`(traefik\.router\.requests\.total,code=200,method=GET,router=demo,service=test,tag1=val1 count=1) [\d]{19}`,
		`(traefik\.router\.requests\.total,code=404,method=GET,router=demo,service=test,tag1=val1 count=1) [\d]{19}`,
		`(traefik\.router\.requests\.tls\.total,router=demo,service=test,tag1=val1,tls_cipher=bar,tls_version=foo count=1) [\d]{19}`,
		`(traefik\.router\.request\.duration,code=200,router=demo,service=test,tag1=val1 p50=10000,p90=10000,p95=10000,p99=10000) [\d]{19}`,
		`(traefik\.router\.connections\.open,router=demo,service=test,tag1=val1 value=1) [\d]{19}`,
	}

	msgRouter := udp.ReceiveString(t, func() {
		influxDBRegistry.RouterReqsCounter().With("router", "demo", "service", "test", "code", strconv.Itoa(http.StatusNotFound), "method", http.MethodGet).Add(1)
		influxDBRegistry.RouterReqsCounter().With("router", "demo", "service", "test", "code", strconv.Itoa(http.StatusOK), "method", http.MethodGet).Add(1)
		influxDBRegistry.RouterReqsTLSCounter().With("router", "demo", "service", "test", "tls_version", "foo", "tls_cipher", "bar").Add(1)
		influxDBRegistry.RouterReqDurationHistogram().With("router", "demo", "service", "test", "code", strconv.Itoa(http.StatusOK)).Observe(10000)
		influxDBRegistry.RouterOpenConnsGauge().With("router", "demo", "service", "test").Set(1)
	})

	assertMessage(t, msgRouter, expectedRouter)

	expectedService := []string{
		`(traefik\.service\.requests\.total,code=200,method=GET,service=test,tag1=val1 count=1) [\d]{19}`,
		`(traefik\.service\.requests\.total,code=404,method=GET,service=test,tag1=val1 count=1) [\d]{19}`,
		`(traefik\.service\.requests\.tls\.total,service=test,tag1=val1,tls_cipher=bar,tls_version=foo count=1) [\d]{19}`,
		`(traefik\.service\.request\.duration,code=200,service=test,tag1=val1 p50=10000,p90=10000,p95=10000,p99=10000) [\d]{19}`,
		`(traefik\.service\.retries\.total(?:,code=[\d]{3},method=GET)?,service=test,tag1=val1 count=2) [\d]{19}`,
		`(traefik\.service\.server\.up,service=test,tag1=val1,url=http://127.0.0.1 value=1) [\d]{19}`,
		`(traefik\.service\.connections\.open,service=test,tag1=val1 value=1) [\d]{19}`,
	}

	msgService := udp.ReceiveString(t, func() {
		influxDBRegistry.ServiceReqsCounter().With("service", "test", "code", strconv.Itoa(http.StatusOK), "method", http.MethodGet).Add(1)
		influxDBRegistry.ServiceReqsCounter().With("service", "test", "code", strconv.Itoa(http.StatusNotFound), "method", http.MethodGet).Add(1)
		influxDBRegistry.ServiceReqsTLSCounter().With("service", "test", "tls_version", "foo", "tls_cipher", "bar").Add(1)
		influxDBRegistry.ServiceReqDurationHistogram().With("service", "test", "code", strconv.Itoa(http.StatusOK)).Observe(10000)
		influxDBRegistry.ServiceOpenConnsGauge().With("service", "test").Set(1)
		influxDBRegistry.ServiceRetriesCounter().With("service", "test").Add(1)
		influxDBRegistry.ServiceRetriesCounter().With("service", "test").Add(1)
		influxDBRegistry.ServiceServerUpGauge().With("service", "test", "url", "http://127.0.0.1").Set(1)
	})

	assertMessage(t, msgService, expectedService)
}

func TestInfluxDBHTTP(t *testing.T) {
	c := make(chan *string)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "can't read body "+err.Error(), http.StatusBadRequest)
			return
		}
		bodyStr := string(body)
		c <- &bodyStr
		_, _ = fmt.Fprintln(w, "ok")
	}))
	defer ts.Close()

	influxDBClient = nil
	influxDBRegistry := RegisterInfluxDB(context.Background(),
		&types.InfluxDB{
			Address:              ts.URL,
			Protocol:             "http",
			PushInterval:         ptypes.Duration(time.Second),
			Database:             "test",
			RetentionPolicy:      "autogen",
			AddEntryPointsLabels: true,
			AddServicesLabels:    true,
			AddRoutersLabels:     true,
		})
	defer StopInfluxDB()

	if !influxDBRegistry.IsEpEnabled() || !influxDBRegistry.IsRouterEnabled() || !influxDBRegistry.IsSvcEnabled() {
		t.Fatalf("InfluxDB registry must be epEnabled")
	}

	expectedServer := []string{
		`(traefik\.config\.reload\.total count=1) [\d]{19}`,
		`(traefik\.config\.reload\.total\.failure count=1) [\d]{19}`,
		`(traefik\.config\.reload\.lastSuccessTimestamp value=1) [\d]{19}`,
		`(traefik\.config\.reload\.lastFailureTimestamp value=1) [\d]{19}`,
	}

	influxDBRegistry.ConfigReloadsCounter().Add(1)
	influxDBRegistry.ConfigReloadsFailureCounter().Add(1)
	influxDBRegistry.LastConfigReloadSuccessGauge().Set(1)
	influxDBRegistry.LastConfigReloadFailureGauge().Set(1)
	msgServer := <-c

	assertMessage(t, *msgServer, expectedServer)

	expectedTLS := []string{
		`(traefik\.tls\.certs\.notAfterTimestamp,key=value value=1) [\d]{19}`,
	}

	influxDBRegistry.TLSCertsNotAfterTimestampGauge().With("key", "value").Set(1)
	msgTLS := <-c

	assertMessage(t, *msgTLS, expectedTLS)

	expectedEntrypoint := []string{
		`(traefik\.entrypoint\.requests\.total,code=200,entrypoint=test,method=GET count=1) [\d]{19}`,
		`(traefik\.entrypoint\.requests\.tls\.total,entrypoint=test,tls_cipher=bar,tls_version=foo count=1) [\d]{19}`,
		`(traefik\.entrypoint\.request\.duration(?:,code=[\d]{3})?,entrypoint=test p50=10000,p90=10000,p95=10000,p99=10000) [\d]{19}`,
		`(traefik\.entrypoint\.connections\.open,entrypoint=test value=1) [\d]{19}`,
	}

	influxDBRegistry.EntryPointReqsCounter().With("entrypoint", "test", "code", strconv.Itoa(http.StatusOK), "method", http.MethodGet).Add(1)
	influxDBRegistry.EntryPointReqsTLSCounter().With("entrypoint", "test", "tls_version", "foo", "tls_cipher", "bar").Add(1)
	influxDBRegistry.EntryPointReqDurationHistogram().With("entrypoint", "test").Observe(10000)
	influxDBRegistry.EntryPointOpenConnsGauge().With("entrypoint", "test").Set(1)
	msgEntrypoint := <-c

	assertMessage(t, *msgEntrypoint, expectedEntrypoint)

	expectedRouter := []string{
		`(traefik\.router\.requests\.total,code=200,method=GET,router=demo,service=test count=1) [\d]{19}`,
		`(traefik\.router\.requests\.total,code=404,method=GET,router=demo,service=test count=1) [\d]{19}`,
		`(traefik\.router\.requests\.tls\.total,router=demo,service=test,tls_cipher=bar,tls_version=foo count=1) [\d]{19}`,
		`(traefik\.router\.request\.duration,code=200,router=demo,service=test p50=10000,p90=10000,p95=10000,p99=10000) [\d]{19}`,
		`(traefik\.router\.connections\.open,router=demo,service=test value=1) [\d]{19}`,
	}

	influxDBRegistry.RouterReqsCounter().With("router", "demo", "service", "test", "code", strconv.Itoa(http.StatusNotFound), "method", http.MethodGet).Add(1)
	influxDBRegistry.RouterReqsCounter().With("router", "demo", "service", "test", "code", strconv.Itoa(http.StatusOK), "method", http.MethodGet).Add(1)
	influxDBRegistry.RouterReqsTLSCounter().With("router", "demo", "service", "test", "tls_version", "foo", "tls_cipher", "bar").Add(1)
	influxDBRegistry.RouterReqDurationHistogram().With("router", "demo", "service", "test", "code", strconv.Itoa(http.StatusOK)).Observe(10000)
	influxDBRegistry.RouterOpenConnsGauge().With("router", "demo", "service", "test").Set(1)
	msgRouter := <-c

	assertMessage(t, *msgRouter, expectedRouter)

	expectedService := []string{
		`(traefik\.service\.requests\.total,code=200,method=GET,service=test count=1) [\d]{19}`,
		`(traefik\.service\.requests\.total,code=404,method=GET,service=test count=1) [\d]{19}`,
		`(traefik\.service\.requests\.tls\.total,service=test,tls_cipher=bar,tls_version=foo count=1) [\d]{19}`,
		`(traefik\.service\.request\.duration,code=200,service=test p50=10000,p90=10000,p95=10000,p99=10000) [\d]{19}`,
		`(traefik\.service\.retries\.total(?:,code=[\d]{3},method=GET)?,service=test count=2) [\d]{19}`,
		`(traefik\.service\.server\.up,service=test,url=http://127.0.0.1 value=1) [\d]{19}`,
		`(traefik\.service\.connections\.open,service=test value=1) [\d]{19}`,
	}

	influxDBRegistry.ServiceReqsCounter().With("service", "test", "code", strconv.Itoa(http.StatusOK), "method", http.MethodGet).Add(1)
	influxDBRegistry.ServiceReqsCounter().With("service", "test", "code", strconv.Itoa(http.StatusNotFound), "method", http.MethodGet).Add(1)
	influxDBRegistry.ServiceReqsTLSCounter().With("service", "test", "tls_version", "foo", "tls_cipher", "bar").Add(1)
	influxDBRegistry.ServiceReqDurationHistogram().With("service", "test", "code", strconv.Itoa(http.StatusOK)).Observe(10000)
	influxDBRegistry.ServiceOpenConnsGauge().With("service", "test").Set(1)
	influxDBRegistry.ServiceRetriesCounter().With("service", "test").Add(1)
	influxDBRegistry.ServiceRetriesCounter().With("service", "test").Add(1)
	influxDBRegistry.ServiceServerUpGauge().With("service", "test", "url", "http://127.0.0.1").Set(1)
	msgService := <-c

	assertMessage(t, *msgService, expectedService)
}

func assertMessage(t *testing.T, msg string, patterns []string) {
	t.Helper()
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		match := re.FindStringSubmatch(msg)
		if len(match) != 2 {
			t.Errorf("Got %q %v, want %q", msg, match, pattern)
		}
	}
}
