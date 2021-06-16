package metrics

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	th "github.com/traefik/traefik/v2/pkg/testhelpers"
	"github.com/traefik/traefik/v2/pkg/types"
)

func TestRegisterPromState(t *testing.T) {
	// Reset state of global promState.
	defer promState.reset()

	testCases := []struct {
		desc                 string
		prometheusSlice      []*types.Prometheus
		initPromState        bool
		unregisterPromState  bool
		expectedNbRegistries int
	}{
		{
			desc:                 "Register once",
			prometheusSlice:      []*types.Prometheus{{}},
			initPromState:        true,
			unregisterPromState:  false,
			expectedNbRegistries: 1,
		},
		{
			desc:                 "Register once with no promState init",
			prometheusSlice:      []*types.Prometheus{{}},
			initPromState:        false,
			unregisterPromState:  false,
			expectedNbRegistries: 1,
		},
		{
			desc:                 "Register twice",
			prometheusSlice:      []*types.Prometheus{{}, {}},
			initPromState:        true,
			unregisterPromState:  false,
			expectedNbRegistries: 2,
		},
		{
			desc:                 "Register twice with no promstate init",
			prometheusSlice:      []*types.Prometheus{{}, {}},
			initPromState:        false,
			unregisterPromState:  false,
			expectedNbRegistries: 2,
		},
		{
			desc:                 "Register twice with unregister",
			prometheusSlice:      []*types.Prometheus{{}, {}},
			initPromState:        true,
			unregisterPromState:  true,
			expectedNbRegistries: 2,
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			actualNbRegistries := 0
			for _, prom := range test.prometheusSlice {
				if test.initPromState {
					initStandardRegistry(prom)
				}
				if registerPromState(context.Background()) {
					actualNbRegistries++
				}
				if test.unregisterPromState {
					promRegistry.Unregister(promState)
				}

				promState.reset()
			}

			promRegistry.Unregister(promState)
			assert.Equal(t, test.expectedNbRegistries, actualNbRegistries)
		})
	}
}

// reset is a utility method for unit testing. It should be called after each
// test run that changes promState internally in order to avoid dependencies
// between unit tests.
func (ps *prometheusState) reset() {
	ps.collectors = make(chan *collector)
	ps.describers = []func(ch chan<- *prometheus.Desc){}
	ps.dynamicConfig = newDynamicConfig()
	ps.state = make(map[string]*collector)
}

func TestPrometheus(t *testing.T) {
	promState = newPrometheusState()
	promRegistry = prometheus.NewRegistry()
	// Reset state of global promState.
	defer promState.reset()

	prometheusRegistry := RegisterPrometheus(context.Background(), &types.Prometheus{AddEntryPointsLabels: true, AddRoutersLabels: true, AddServicesLabels: true})
	defer promRegistry.Unregister(promState)

	if !prometheusRegistry.IsEpEnabled() || !prometheusRegistry.IsRouterEnabled() || !prometheusRegistry.IsSvcEnabled() {
		t.Errorf("PrometheusRegistry should return true for IsEnabled(), IsRouterEnabled() and IsSvcEnabled()")
	}

	prometheusRegistry.ConfigReloadsCounter().Add(1)
	prometheusRegistry.ConfigReloadsFailureCounter().Add(1)
	prometheusRegistry.LastConfigReloadSuccessGauge().Set(float64(time.Now().Unix()))
	prometheusRegistry.LastConfigReloadFailureGauge().Set(float64(time.Now().Unix()))

	prometheusRegistry.
		TLSCertsNotAfterTimestampGauge().
		With("cn", "value", "serial", "value", "sans", "value").
		Set(float64(time.Now().Unix()))

	prometheusRegistry.
		EntryPointReqsCounter().
		With("code", strconv.Itoa(http.StatusOK), "method", http.MethodGet, "protocol", "http", "entrypoint", "http").
		Add(1)
	prometheusRegistry.
		EntryPointReqDurationHistogram().
		With("code", strconv.Itoa(http.StatusOK), "method", http.MethodGet, "protocol", "http", "entrypoint", "http").
		Observe(1)
	prometheusRegistry.
		EntryPointOpenConnsGauge().
		With("method", http.MethodGet, "protocol", "http", "entrypoint", "http").
		Set(1)

	prometheusRegistry.
		RouterReqsCounter().
		With("router", "demo", "service", "service1", "code", strconv.Itoa(http.StatusOK), "method", http.MethodGet, "protocol", "http").
		Add(1)
	prometheusRegistry.
		RouterReqsTLSCounter().
		With("router", "demo", "service", "service1", "tls_version", "foo", "tls_cipher", "bar").
		Add(1)
	prometheusRegistry.
		RouterReqDurationHistogram().
		With("router", "demo", "service", "service1", "code", strconv.Itoa(http.StatusOK), "method", http.MethodGet, "protocol", "http").
		Observe(10000)
	prometheusRegistry.
		RouterOpenConnsGauge().
		With("router", "demo", "service", "service1", "method", http.MethodGet, "protocol", "http").
		Set(1)

	prometheusRegistry.
		ServiceReqsCounter().
		With("service", "service1", "code", strconv.Itoa(http.StatusOK), "method", http.MethodGet, "protocol", "http").
		Add(1)
	prometheusRegistry.
		ServiceReqsTLSCounter().
		With("service", "service1", "tls_version", "foo", "tls_cipher", "bar").
		Add(1)
	prometheusRegistry.
		ServiceReqDurationHistogram().
		With("service", "service1", "code", strconv.Itoa(http.StatusOK), "method", http.MethodGet, "protocol", "http").
		Observe(10000)
	prometheusRegistry.
		ServiceOpenConnsGauge().
		With("service", "service1", "method", http.MethodGet, "protocol", "http").
		Set(1)
	prometheusRegistry.
		ServiceRetriesCounter().
		With("service", "service1").
		Add(1)
	prometheusRegistry.
		ServiceServerUpGauge().
		With("service", "service1", "url", "http://127.0.0.10:80").
		Set(1)

	delayForTrackingCompletion()

	metricsFamilies := mustScrape()

	testCases := []struct {
		name   string
		labels map[string]string
		assert func(*dto.MetricFamily)
	}{
		{
			name:   configReloadsTotalName,
			assert: buildCounterAssert(t, configReloadsTotalName, 1),
		},
		{
			name:   configReloadsFailuresTotalName,
			assert: buildCounterAssert(t, configReloadsFailuresTotalName, 1),
		},
		{
			name:   configLastReloadSuccessName,
			assert: buildTimestampAssert(t, configLastReloadSuccessName),
		},
		{
			name:   configLastReloadFailureName,
			assert: buildTimestampAssert(t, configLastReloadFailureName),
		},
		{
			name: tlsCertsNotAfterTimestamp,
			labels: map[string]string{
				"cn":     "value",
				"serial": "value",
				"sans":   "value",
			},
			assert: buildTimestampAssert(t, tlsCertsNotAfterTimestamp),
		},
		{
			name: entryPointReqsTotalName,
			labels: map[string]string{
				"code":       "200",
				"method":     http.MethodGet,
				"protocol":   "http",
				"entrypoint": "http",
			},
			assert: buildCounterAssert(t, entryPointReqsTotalName, 1),
		},
		{
			name: entryPointReqDurationName,
			labels: map[string]string{
				"code":       "200",
				"method":     http.MethodGet,
				"protocol":   "http",
				"entrypoint": "http",
			},
			assert: buildHistogramAssert(t, entryPointReqDurationName, 1),
		},
		{
			name: entryPointOpenConnsName,
			labels: map[string]string{
				"method":     http.MethodGet,
				"protocol":   "http",
				"entrypoint": "http",
			},
			assert: buildGaugeAssert(t, entryPointOpenConnsName, 1),
		},
		{
			name: routerReqsTotalName,
			labels: map[string]string{
				"code":     "200",
				"method":   http.MethodGet,
				"protocol": "http",
				"service":  "service1",
				"router":   "demo",
			},
			assert: buildCounterAssert(t, routerReqsTotalName, 1),
		},
		{
			name: routerReqsTLSTotalName,
			labels: map[string]string{
				"service":     "service1",
				"router":      "demo",
				"tls_version": "foo",
				"tls_cipher":  "bar",
			},
			assert: buildCounterAssert(t, routerReqsTLSTotalName, 1),
		},
		{
			name: routerReqDurationName,
			labels: map[string]string{
				"code":     "200",
				"method":   http.MethodGet,
				"protocol": "http",
				"service":  "service1",
				"router":   "demo",
			},
			assert: buildHistogramAssert(t, routerReqDurationName, 1),
		},
		{
			name: routerOpenConnsName,
			labels: map[string]string{
				"method":   http.MethodGet,
				"protocol": "http",
				"service":  "service1",
				"router":   "demo",
			},
			assert: buildGaugeAssert(t, routerOpenConnsName, 1),
		},
		{
			name: serviceReqsTotalName,
			labels: map[string]string{
				"code":     "200",
				"method":   http.MethodGet,
				"protocol": "http",
				"service":  "service1",
			},
			assert: buildCounterAssert(t, serviceReqsTotalName, 1),
		},
		{
			name: serviceReqsTLSTotalName,
			labels: map[string]string{
				"service":     "service1",
				"tls_version": "foo",
				"tls_cipher":  "bar",
			},
			assert: buildCounterAssert(t, serviceReqsTLSTotalName, 1),
		},
		{
			name: serviceReqDurationName,
			labels: map[string]string{
				"code":     "200",
				"method":   http.MethodGet,
				"protocol": "http",
				"service":  "service1",
			},
			assert: buildHistogramAssert(t, serviceReqDurationName, 1),
		},
		{
			name: serviceOpenConnsName,
			labels: map[string]string{
				"method":   http.MethodGet,
				"protocol": "http",
				"service":  "service1",
			},
			assert: buildGaugeAssert(t, serviceOpenConnsName, 1),
		},
		{
			name: serviceRetriesTotalName,
			labels: map[string]string{
				"service": "service1",
			},
			assert: buildGreaterThanCounterAssert(t, serviceRetriesTotalName, 1),
		},
		{
			name: serviceServerUpName,
			labels: map[string]string{
				"service": "service1",
				"url":     "http://127.0.0.10:80",
			},
			assert: buildGaugeAssert(t, serviceServerUpName, 1),
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.name, func(t *testing.T) {
			family := findMetricFamily(test.name, metricsFamilies)
			if family == nil {
				t.Errorf("gathered metrics do not contain %q", test.name)
				return
			}

			for _, label := range family.Metric[0].Label {
				val, ok := test.labels[*label.Name]
				if !ok {
					t.Errorf("%q metric contains unexpected label %q", test.name, *label.Name)
				} else if val != *label.Value {
					t.Errorf("label %q in metric %q has wrong value %q, expected %q", *label.Name, test.name, *label.Value, val)
				}
			}
			test.assert(family)
		})
	}
}

func TestPrometheusMetricRemoval(t *testing.T) {
	promState = newPrometheusState()
	promRegistry = prometheus.NewRegistry()
	// Reset state of global promState.
	defer promState.reset()

	prometheusRegistry := RegisterPrometheus(context.Background(), &types.Prometheus{AddEntryPointsLabels: true, AddServicesLabels: true})
	defer promRegistry.Unregister(promState)

	conf := dynamic.Configuration{
		HTTP: th.BuildConfiguration(
			th.WithRouters(
				th.WithRouter("foo@providerName",
					th.WithServiceName("bar")),
			),
			th.WithLoadBalancerServices(th.WithService("bar@providerName",
				th.WithServers(th.WithServer("http://localhost:9000"))),
			),
			func(cfg *dynamic.HTTPConfiguration) {
				cfg.Services["fii"] = &dynamic.Service{
					Weighted: &dynamic.WeightedRoundRobin{},
				}
			},
		),
	}

	OnConfigurationUpdate(conf, []string{"entrypoint1"})

	// Register some metrics manually that are not part of the active configuration.
	// Those metrics should be part of the /metrics output on the first scrape but
	// should be removed after that scrape.
	prometheusRegistry.
		EntryPointReqsCounter().
		With("entrypoint", "entrypoint2", "code", strconv.Itoa(http.StatusOK), "method", http.MethodGet, "protocol", "http").
		Add(1)
	prometheusRegistry.
		ServiceReqsCounter().
		With("service", "service2", "code", strconv.Itoa(http.StatusOK), "method", http.MethodGet, "protocol", "http").
		Add(1)
	prometheusRegistry.
		ServiceServerUpGauge().
		With("service", "service1", "url", "http://localhost:9999").
		Set(1)

	delayForTrackingCompletion()

	assertMetricsExist(t, mustScrape(), entryPointReqsTotalName, serviceReqsTotalName, serviceServerUpName)
	assertMetricsAbsent(t, mustScrape(), entryPointReqsTotalName, serviceReqsTotalName, serviceServerUpName)

	// To verify that metrics belonging to active configurations are not removed
	// here the counter examples.
	prometheusRegistry.
		EntryPointReqsCounter().
		With("entrypoint", "entrypoint1", "code", strconv.Itoa(http.StatusOK), "method", http.MethodGet, "protocol", "http").
		Add(1)

	delayForTrackingCompletion()

	assertMetricsExist(t, mustScrape(), entryPointReqsTotalName)
	assertMetricsExist(t, mustScrape(), entryPointReqsTotalName)
}

func TestPrometheusRemovedMetricsReset(t *testing.T) {
	// Reset state of global promState.
	defer promState.reset()

	prometheusRegistry := RegisterPrometheus(context.Background(), &types.Prometheus{AddEntryPointsLabels: true, AddServicesLabels: true})
	defer promRegistry.Unregister(promState)

	labelNamesValues := []string{
		"service", "service",
		"code", strconv.Itoa(http.StatusOK),
		"method", http.MethodGet,
		"protocol", "http",
	}
	prometheusRegistry.
		ServiceReqsCounter().
		With(labelNamesValues...).
		Add(3)

	delayForTrackingCompletion()

	metricsFamilies := mustScrape()
	assertCounterValue(t, 3, findMetricFamily(serviceReqsTotalName, metricsFamilies), labelNamesValues...)

	// There is no dynamic configuration and so this metric will be deleted
	// after the first scrape.
	assertMetricsAbsent(t, mustScrape(), serviceReqsTotalName)

	prometheusRegistry.
		ServiceReqsCounter().
		With(labelNamesValues...).
		Add(1)

	delayForTrackingCompletion()

	metricsFamilies = mustScrape()
	assertCounterValue(t, 1, findMetricFamily(serviceReqsTotalName, metricsFamilies), labelNamesValues...)
}

// Tracking and gathering the metrics happens concurrently.
// In practice this is no problem, because in case a tracked metric would miss
// the current scrape, it would just be there in the next one.
// That we can test reliably the tracking of all metrics here, we sleep
// for a short amount of time, to make sure the metric will be present
// in the next scrape.
func delayForTrackingCompletion() {
	time.Sleep(250 * time.Millisecond)
}

func mustScrape() []*dto.MetricFamily {
	families, err := promRegistry.Gather()
	if err != nil {
		panic(fmt.Sprintf("could not gather metrics families: %s", err))
	}
	return families
}

func assertMetricsExist(t *testing.T, families []*dto.MetricFamily, metricNames ...string) {
	t.Helper()

	for _, metricName := range metricNames {
		if findMetricFamily(metricName, families) == nil {
			t.Errorf("gathered metrics should contain %q", metricName)
		}
	}
}

func assertMetricsAbsent(t *testing.T, families []*dto.MetricFamily, metricNames ...string) {
	t.Helper()

	for _, metricName := range metricNames {
		if findMetricFamily(metricName, families) != nil {
			t.Errorf("gathered metrics should not contain %q", metricName)
		}
	}
}

func findMetricFamily(name string, families []*dto.MetricFamily) *dto.MetricFamily {
	for _, family := range families {
		if family.GetName() == name {
			return family
		}
	}
	return nil
}

func findMetricByLabelNamesValues(family *dto.MetricFamily, labelNamesValues ...string) *dto.Metric {
	if family == nil {
		return nil
	}

	for _, metric := range family.Metric {
		if hasMetricAllLabelPairs(metric, labelNamesValues...) {
			return metric
		}
	}

	return nil
}

func hasMetricAllLabelPairs(metric *dto.Metric, labelNamesValues ...string) bool {
	for i := 0; i < len(labelNamesValues); i += 2 {
		name, val := labelNamesValues[i], labelNamesValues[i+1]
		if !hasMetricLabelPair(metric, name, val) {
			return false
		}
	}
	return true
}

func hasMetricLabelPair(metric *dto.Metric, labelName, labelValue string) bool {
	for _, labelPair := range metric.Label {
		if labelPair.GetName() == labelName && labelPair.GetValue() == labelValue {
			return true
		}
	}
	return false
}

func assertCounterValue(t *testing.T, want float64, family *dto.MetricFamily, labelNamesValues ...string) {
	t.Helper()

	metric := findMetricByLabelNamesValues(family, labelNamesValues...)

	if metric == nil {
		t.Error("metric must not be nil")
		return
	}
	if metric.Counter == nil {
		t.Errorf("metric %s must be a counter", family.GetName())
		return
	}

	if cv := metric.Counter.GetValue(); cv != want {
		t.Errorf("metric %s has value %v, want %v", family.GetName(), cv, want)
	}
}

func buildCounterAssert(t *testing.T, metricName string, expectedValue int) func(family *dto.MetricFamily) {
	t.Helper()

	return func(family *dto.MetricFamily) {
		if cv := int(family.Metric[0].Counter.GetValue()); cv != expectedValue {
			t.Errorf("metric %s has value %d, want %d", metricName, cv, expectedValue)
		}
	}
}

func buildGreaterThanCounterAssert(t *testing.T, metricName string, expectedMinValue int) func(family *dto.MetricFamily) {
	t.Helper()

	return func(family *dto.MetricFamily) {
		if cv := int(family.Metric[0].Counter.GetValue()); cv < expectedMinValue {
			t.Errorf("metric %s has value %d, want at least %d", metricName, cv, expectedMinValue)
		}
	}
}

func buildHistogramAssert(t *testing.T, metricName string, expectedSampleCount int) func(family *dto.MetricFamily) {
	t.Helper()

	return func(family *dto.MetricFamily) {
		if sc := int(family.Metric[0].Histogram.GetSampleCount()); sc != expectedSampleCount {
			t.Errorf("metric %s has sample count value %d, want %d", metricName, sc, expectedSampleCount)
		}
	}
}

func buildGaugeAssert(t *testing.T, metricName string, expectedValue int) func(family *dto.MetricFamily) {
	t.Helper()

	return func(family *dto.MetricFamily) {
		if gv := int(family.Metric[0].Gauge.GetValue()); gv != expectedValue {
			t.Errorf("metric %s has value %d, want %d", metricName, gv, expectedValue)
		}
	}
}

func buildTimestampAssert(t *testing.T, metricName string) func(family *dto.MetricFamily) {
	t.Helper()

	return func(family *dto.MetricFamily) {
		if ts := time.Unix(int64(family.Metric[0].Gauge.GetValue()), 0); time.Since(ts) > time.Minute {
			t.Errorf("metric %s has wrong timestamp %v", metricName, ts)
		}
	}
}
