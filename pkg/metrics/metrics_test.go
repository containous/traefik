package metrics

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/generic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScalableHistogram(t *testing.T) {
	h := generic.NewHistogram("test", 1)
	sh, err := NewHistogramWithScale(h, time.Millisecond)
	require.NoError(t, err)

	ticker := time.NewTicker(500 * time.Millisecond)
	<-ticker.C
	start := time.Now()
	<-ticker.C
	sh.ObserveFromStart(start)

	var b bytes.Buffer
	h.Print(&b)

	extractedDurationString := strings.Split(strings.Split(b.String(), "\n")[1], " ")
	measuredDuration, err := time.ParseDuration(extractedDurationString[0] + "ms")
	assert.NoError(t, err)

	assert.InDelta(t, 500*time.Millisecond, measuredDuration, float64(15*time.Millisecond))
}

func TestNewMultiRegistry(t *testing.T) {
	registries := []Registry{newCollectingRetryMetrics(), newCollectingRetryMetrics()}
	registry := NewMultiRegistry(registries)

	registry.ServiceReqsCounter().With("key", "requests").Add(1)
	registry.ServiceReqDurationHistogram().With("key", "durations").Observe(float64(2))
	registry.ServiceRetriesCounter().With("key", "retries").Add(3)

	for _, collectingRegistry := range registries {
		cReqsCounter := collectingRegistry.ServiceReqsCounter().(*counterMock)
		cReqDurationHistogram := collectingRegistry.ServiceReqDurationHistogram().(*histogramMock)
		cRetriesCounter := collectingRegistry.ServiceRetriesCounter().(*counterMock)

		wantCounterValue := float64(1)
		if cReqsCounter.counterValue != wantCounterValue {
			t.Errorf("Got value %f for ReqsCounter, want %f", cReqsCounter.counterValue, wantCounterValue)
		}
		wantHistogramValue := float64(2)
		if cReqDurationHistogram.lastHistogramValue != wantHistogramValue {
			t.Errorf("Got last observation %f for ReqDurationHistogram, want %f", cReqDurationHistogram.lastHistogramValue, wantHistogramValue)
		}
		wantCounterValue = float64(3)
		if cRetriesCounter.counterValue != wantCounterValue {
			t.Errorf("Got value %f for RetriesCounter, want %f", cRetriesCounter.counterValue, wantCounterValue)
		}

		assert.Equal(t, []string{"key", "requests"}, cReqsCounter.lastLabelValues)
		assert.Equal(t, []string{"key", "durations"}, cReqDurationHistogram.lastLabelValues)
		assert.Equal(t, []string{"key", "retries"}, cRetriesCounter.lastLabelValues)
	}
}

func newCollectingRetryMetrics() Registry {
	return &standardRegistry{
		serviceReqsCounter:          &counterMock{},
		serviceReqDurationHistogram: &histogramMock{},
		serviceRetriesCounter:       &counterMock{},
	}
}

type counterMock struct {
	counterValue    float64
	lastLabelValues []string
}

func (c *counterMock) With(labelValues ...string) metrics.Counter {
	c.lastLabelValues = labelValues
	return c
}

func (c *counterMock) Add(delta float64) {
	c.counterValue += delta
}

type histogramMock struct {
	lastHistogramValue float64
	lastLabelValues    []string
}

func (c *histogramMock) With(labelValues ...string) ScalableHistogram {
	c.lastLabelValues = labelValues
	return c
}

func (c *histogramMock) Start() {}

func (c *histogramMock) ObserveFromStart(t time.Time) {}

func (c *histogramMock) Observe(v float64) {
	c.lastHistogramValue = v
}
