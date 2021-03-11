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

	influxDBRegistry := RegisterInfluxDB(context.Background(), &types.InfluxDB{Address: ":8089", PushInterval: ptypes.Duration(time.Second), AddEntryPointsLabels: true, AddServicesLabels: true})
	defer StopInfluxDB()

	if !influxDBRegistry.IsEpEnabled() || !influxDBRegistry.IsSvcEnabled() {
		t.Fatalf("InfluxDB registry must be epEnabled")
	}

	expectedService := []string{
		`(traefik\.service\.requests\.total,code=200,method=GET,service=test count=1) [\d]{19}`,
		`(traefik\.service\.requests\.total,code=404,method=GET,service=test count=1) [\d]{19}`,
		`(traefik\.service\.request\.duration,code=200,service=test p50=10000,p90=10000,p95=10000,p99=10000) [\d]{19}`,
		`(traefik\.service\.retries\.total(?:,code=[\d]{3},method=GET)?,service=test count=2) [\d]{19}`,
		`(traefik\.config\.reload\.total(?:[a-z=0-9A-Z,]+)? count=1) [\d]{19}`,
		`(traefik\.config\.reload\.total\.failure(?:[a-z=0-9A-Z,]+)? count=1) [\d]{19}`,
		`(traefik\.service\.server\.up,service=test(?:[a-z=0-9A-Z,]+)?,url=http://127.0.0.1 value=1) [\d]{19}`,
	}

	msgService := udp.ReceiveString(t, func() {
		influxDBRegistry.ServiceReqsCounter().With("service", "test", "code", strconv.Itoa(http.StatusOK), "method", http.MethodGet).Add(1)
		influxDBRegistry.ServiceReqsCounter().With("service", "test", "code", strconv.Itoa(http.StatusNotFound), "method", http.MethodGet).Add(1)
		influxDBRegistry.ServiceRetriesCounter().With("service", "test").Add(1)
		influxDBRegistry.ServiceRetriesCounter().With("service", "test").Add(1)
		influxDBRegistry.ServiceReqDurationHistogram().With("service", "test", "code", strconv.Itoa(http.StatusOK)).Observe(10000)
		influxDBRegistry.ConfigReloadsCounter().Add(1)
		influxDBRegistry.ConfigReloadsFailureCounter().Add(1)
		influxDBRegistry.ServiceServerUpGauge().With("service", "test", "url", "http://127.0.0.1").Set(1)
	})

	assertMessage(t, msgService, expectedService)

	expectedEntrypoint := []string{
		`(traefik\.entrypoint\.requests\.total,entrypoint=test(?:[a-z=0-9A-Z,:/.]+)? count=1) [\d]{19}`,
		`(traefik\.entrypoint\.request\.duration(?:,code=[\d]{3})?,entrypoint=test(?:[a-z=0-9A-Z,:/.]+)? p50=10000,p90=10000,p95=10000,p99=10000) [\d]{19}`,
		`(traefik\.entrypoint\.connections\.open,entrypoint=test value=1) [\d]{19}`,
	}

	msgEntrypoint := udp.ReceiveString(t, func() {
		influxDBRegistry.EntryPointReqsCounter().With("entrypoint", "test").Add(1)
		influxDBRegistry.EntryPointReqDurationHistogram().With("entrypoint", "test").Observe(10000)
		influxDBRegistry.EntryPointOpenConnsGauge().With("entrypoint", "test").Set(1)
	})

	assertMessage(t, msgEntrypoint, expectedEntrypoint)

	expectedTLS := []string{
		`(traefik\.tls\.certs\.notAfterTimestamp,key=value value=1) [\d]{19}`,
	}

	msgTLS := udp.ReceiveString(t, func() {
		influxDBRegistry.TLSCertsNotAfterTimestampGauge().With("key", "value").Set(1)
	})

	assertMessage(t, msgTLS, expectedTLS)
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

	influxDBRegistry := RegisterInfluxDB(context.Background(), &types.InfluxDB{Address: ts.URL, Protocol: "http", PushInterval: ptypes.Duration(time.Second), Database: "test", RetentionPolicy: "autogen", AddEntryPointsLabels: true, AddServicesLabels: true})
	defer StopInfluxDB()

	if !influxDBRegistry.IsEpEnabled() || !influxDBRegistry.IsSvcEnabled() {
		t.Fatalf("InfluxDB registry must be epEnabled")
	}

	expectedService := []string{
		`(traefik\.service\.requests\.total,code=200,method=GET,service=test count=1) [\d]{19}`,
		`(traefik\.service\.requests\.total,code=404,method=GET,service=test count=1) [\d]{19}`,
		`(traefik\.service\.request\.duration,code=200,service=test p50=10000,p90=10000,p95=10000,p99=10000) [\d]{19}`,
		`(traefik\.service\.retries\.total(?:,code=[\d]{3},method=GET)?,service=test count=2) [\d]{19}`,
		`(traefik\.config\.reload\.total(?:[a-z=0-9A-Z,]+)? count=1) [\d]{19}`,
		`(traefik\.config\.reload\.total\.failure(?:[a-z=0-9A-Z,]+)? count=1) [\d]{19}`,
		`(traefik\.service\.server\.up,service=test(?:[a-z=0-9A-Z,]+)?,url=http://127.0.0.1 value=1) [\d]{19}`,
	}

	influxDBRegistry.ServiceReqsCounter().With("service", "test", "code", strconv.Itoa(http.StatusOK), "method", http.MethodGet).Add(1)
	influxDBRegistry.ServiceReqsCounter().With("service", "test", "code", strconv.Itoa(http.StatusNotFound), "method", http.MethodGet).Add(1)
	influxDBRegistry.ServiceRetriesCounter().With("service", "test").Add(1)
	influxDBRegistry.ServiceRetriesCounter().With("service", "test").Add(1)
	influxDBRegistry.ServiceReqDurationHistogram().With("service", "test", "code", strconv.Itoa(http.StatusOK)).Observe(10000)
	influxDBRegistry.ConfigReloadsCounter().Add(1)
	influxDBRegistry.ConfigReloadsFailureCounter().Add(1)
	influxDBRegistry.ServiceServerUpGauge().With("service", "test", "url", "http://127.0.0.1").Set(1)
	msgService := <-c

	assertMessage(t, *msgService, expectedService)

	expectedEntrypoint := []string{
		`(traefik\.entrypoint\.requests\.total,entrypoint=test(?:[a-z=0-9A-Z,:/.]+)? count=1) [\d]{19}`,
		`(traefik\.entrypoint\.request\.duration(?:,code=[\d]{3})?,entrypoint=test(?:[a-z=0-9A-Z,:/.]+)? p50=10000,p90=10000,p95=10000,p99=10000) [\d]{19}`,
		`(traefik\.entrypoint\.connections\.open,entrypoint=test value=1) [\d]{19}`,
	}

	influxDBRegistry.EntryPointReqsCounter().With("entrypoint", "test").Add(1)
	influxDBRegistry.EntryPointReqDurationHistogram().With("entrypoint", "test").Observe(10000)
	influxDBRegistry.EntryPointOpenConnsGauge().With("entrypoint", "test").Set(1)
	msgEntrypoint := <-c

	assertMessage(t, *msgEntrypoint, expectedEntrypoint)

	expectedTLS := []string{
		`(traefik\.tls\.certs\.notAfterTimestamp,key=value value=1) [\d]{19}`,
	}

	influxDBRegistry.TLSCertsNotAfterTimestampGauge().With("key", "value").Set(1)
	msgTLS := <-c

	assertMessage(t, *msgTLS, expectedTLS)
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
