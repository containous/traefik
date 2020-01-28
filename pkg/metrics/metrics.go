package metrics

import (
	"time"

	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/multi"
)

// Registry has to implemented by any system that wants to monitor and expose metrics.
type Registry interface {
	// IsEpEnabled shows whether metrics instrumentation is enabled on entry points.
	IsEpEnabled() bool
	// IsSvcEnabled shows whether metrics instrumentation is enabled on services.
	IsSvcEnabled() bool

	// server metrics
	ConfigReloadsCounter() metrics.Counter
	ConfigReloadsFailureCounter() metrics.Counter
	LastConfigReloadSuccessGauge() metrics.Gauge
	LastConfigReloadFailureGauge() metrics.Gauge

	// entry point metrics
	EntryPointReqsCounter() metrics.Counter
	EntryPointReqDurationHistogram() ScalableHistogram
	EntryPointOpenConnsGauge() metrics.Gauge

	// service metrics
	ServiceReqsCounter() metrics.Counter
	ServiceReqDurationHistogram() ScalableHistogram
	ServiceOpenConnsGauge() metrics.Gauge
	ServiceRetriesCounter() metrics.Counter
	ServiceServerUpGauge() metrics.Gauge
}

// NewVoidRegistry is a noop implementation of metrics.Registry.
// It is used to avoid nil checking in components that do metric collections.
func NewVoidRegistry() Registry {
	return NewMultiRegistry([]Registry{})
}

// NewMultiRegistry is an implementation of metrics.Registry that wraps multiple registries.
// It handles the case when a registry hasn't registered some metric and returns nil.
// This allows for feature imparity between the different metric implementations.
func NewMultiRegistry(registries []Registry) Registry {
	var configReloadsCounter []metrics.Counter
	var configReloadsFailureCounter []metrics.Counter
	var lastConfigReloadSuccessGauge []metrics.Gauge
	var lastConfigReloadFailureGauge []metrics.Gauge
	var entryPointReqsCounter []metrics.Counter
	var entryPointReqDurationHistogram []ScalableHistogram
	var entryPointOpenConnsGauge []metrics.Gauge
	var serviceReqsCounter []metrics.Counter
	var serviceReqDurationHistogram []ScalableHistogram
	var serviceOpenConnsGauge []metrics.Gauge
	var serviceRetriesCounter []metrics.Counter
	var serviceServerUpGauge []metrics.Gauge

	for _, r := range registries {
		if r.ConfigReloadsCounter() != nil {
			configReloadsCounter = append(configReloadsCounter, r.ConfigReloadsCounter())
		}
		if r.ConfigReloadsFailureCounter() != nil {
			configReloadsFailureCounter = append(configReloadsFailureCounter, r.ConfigReloadsFailureCounter())
		}
		if r.LastConfigReloadSuccessGauge() != nil {
			lastConfigReloadSuccessGauge = append(lastConfigReloadSuccessGauge, r.LastConfigReloadSuccessGauge())
		}
		if r.LastConfigReloadFailureGauge() != nil {
			lastConfigReloadFailureGauge = append(lastConfigReloadFailureGauge, r.LastConfigReloadFailureGauge())
		}
		if r.EntryPointReqsCounter() != nil {
			entryPointReqsCounter = append(entryPointReqsCounter, r.EntryPointReqsCounter())
		}
		if r.EntryPointReqDurationHistogram() != nil {
			entryPointReqDurationHistogram = append(entryPointReqDurationHistogram, r.EntryPointReqDurationHistogram())
		}
		if r.EntryPointOpenConnsGauge() != nil {
			entryPointOpenConnsGauge = append(entryPointOpenConnsGauge, r.EntryPointOpenConnsGauge())
		}
		if r.ServiceReqsCounter() != nil {
			serviceReqsCounter = append(serviceReqsCounter, r.ServiceReqsCounter())
		}
		if r.ServiceReqDurationHistogram() != nil {
			serviceReqDurationHistogram = append(serviceReqDurationHistogram, r.ServiceReqDurationHistogram())
		}
		if r.ServiceOpenConnsGauge() != nil {
			serviceOpenConnsGauge = append(serviceOpenConnsGauge, r.ServiceOpenConnsGauge())
		}
		if r.ServiceRetriesCounter() != nil {
			serviceRetriesCounter = append(serviceRetriesCounter, r.ServiceRetriesCounter())
		}
		if r.ServiceServerUpGauge() != nil {
			serviceServerUpGauge = append(serviceServerUpGauge, r.ServiceServerUpGauge())
		}
	}

	return &standardRegistry{
		epEnabled:                      len(entryPointReqsCounter) > 0 || len(entryPointReqDurationHistogram) > 0 || len(entryPointOpenConnsGauge) > 0,
		svcEnabled:                     len(serviceReqsCounter) > 0 || len(serviceReqDurationHistogram) > 0 || len(serviceOpenConnsGauge) > 0 || len(serviceRetriesCounter) > 0 || len(serviceServerUpGauge) > 0,
		configReloadsCounter:           multi.NewCounter(configReloadsCounter...),
		configReloadsFailureCounter:    multi.NewCounter(configReloadsFailureCounter...),
		lastConfigReloadSuccessGauge:   multi.NewGauge(lastConfigReloadSuccessGauge...),
		lastConfigReloadFailureGauge:   multi.NewGauge(lastConfigReloadFailureGauge...),
		entryPointReqsCounter:          multi.NewCounter(entryPointReqsCounter...),
		entryPointReqDurationHistogram: NewMultiHistogram(entryPointReqDurationHistogram...),
		entryPointOpenConnsGauge:       multi.NewGauge(entryPointOpenConnsGauge...),
		serviceReqsCounter:             multi.NewCounter(serviceReqsCounter...),
		serviceReqDurationHistogram:    NewMultiHistogram(serviceReqDurationHistogram...),
		serviceOpenConnsGauge:          multi.NewGauge(serviceOpenConnsGauge...),
		serviceRetriesCounter:          multi.NewCounter(serviceRetriesCounter...),
		serviceServerUpGauge:           multi.NewGauge(serviceServerUpGauge...),
	}
}

type standardRegistry struct {
	epEnabled                      bool
	svcEnabled                     bool
	configReloadsCounter           metrics.Counter
	configReloadsFailureCounter    metrics.Counter
	lastConfigReloadSuccessGauge   metrics.Gauge
	lastConfigReloadFailureGauge   metrics.Gauge
	entryPointReqsCounter          metrics.Counter
	entryPointReqDurationHistogram ScalableHistogram
	entryPointOpenConnsGauge       metrics.Gauge
	serviceReqsCounter             metrics.Counter
	serviceReqDurationHistogram    ScalableHistogram
	serviceOpenConnsGauge          metrics.Gauge
	serviceRetriesCounter          metrics.Counter
	serviceServerUpGauge           metrics.Gauge
}

func (r *standardRegistry) IsEpEnabled() bool {
	return r.epEnabled
}

func (r *standardRegistry) IsSvcEnabled() bool {
	return r.svcEnabled
}

func (r *standardRegistry) ConfigReloadsCounter() metrics.Counter {
	return r.configReloadsCounter
}

func (r *standardRegistry) ConfigReloadsFailureCounter() metrics.Counter {
	return r.configReloadsFailureCounter
}

func (r *standardRegistry) LastConfigReloadSuccessGauge() metrics.Gauge {
	return r.lastConfigReloadSuccessGauge
}

func (r *standardRegistry) LastConfigReloadFailureGauge() metrics.Gauge {
	return r.lastConfigReloadFailureGauge
}

func (r *standardRegistry) EntryPointReqsCounter() metrics.Counter {
	return r.entryPointReqsCounter
}

func (r *standardRegistry) EntryPointReqDurationHistogram() ScalableHistogram {
	return r.entryPointReqDurationHistogram
}

func (r *standardRegistry) EntryPointOpenConnsGauge() metrics.Gauge {
	return r.entryPointOpenConnsGauge
}

func (r *standardRegistry) ServiceReqsCounter() metrics.Counter {
	return r.serviceReqsCounter
}

func (r *standardRegistry) ServiceReqDurationHistogram() ScalableHistogram {
	return r.serviceReqDurationHistogram
}

func (r *standardRegistry) ServiceOpenConnsGauge() metrics.Gauge {
	return r.serviceOpenConnsGauge
}

func (r *standardRegistry) ServiceRetriesCounter() metrics.Counter {
	return r.serviceRetriesCounter
}

func (r *standardRegistry) ServiceServerUpGauge() metrics.Gauge {
	return r.serviceServerUpGauge
}

// ScalableHistogram is a Histogram with a predefined time unit used when
// producing observations allowing the call to Start() and ObserveDuration()
// without explicitly setting the observed value.
type ScalableHistogram interface {
	With(labelValues ...string) ScalableHistogram
	Start()
	Observe(v float64)
	ObserveDuration()
}

// HistogramWithScale is a histogram that will convert its observed value to the specified unit.
type HistogramWithScale struct {
	histogram metrics.Histogram
	unit      time.Duration
	timer     *metrics.Timer
}

// With implements ScalableHistogram.
func (s *HistogramWithScale) With(labelValues ...string) ScalableHistogram {
	s.histogram = s.histogram.With(labelValues...)
	return s
}

// Start implements ScalableHistogram.
func (s *HistogramWithScale) Start() {
	s.timer = metrics.NewTimer(s.histogram)
	s.timer.Unit(s.unit)
}

// ObserveDuration implements ScalableHistogram.
func (s *HistogramWithScale) ObserveDuration() {
	if s.timer != nil {
		s.timer.ObserveDuration()
	}
}

// Observe implements ScalableHistogram.
func (s *HistogramWithScale) Observe(v float64) {
	s.histogram.Observe(v)
}

// NewHistogramWithScale returns a ScalableHistogram
func NewHistogramWithScale(h metrics.Histogram, u time.Duration) ScalableHistogram {
	return &HistogramWithScale{
		histogram: h,
		unit:      u,
	}
}

// MultiHistogram collects multiple individual histograms and treats them as a unit.
type MultiHistogram []ScalableHistogram

// NewMultiHistogram returns a multi-histogram, wrapping the passed histograms.
func NewMultiHistogram(h ...ScalableHistogram) MultiHistogram {
	return MultiHistogram(h)
}

// Start implements ScalableHistogram.
func (h MultiHistogram) Start() {
	for _, histogram := range h {
		histogram.Start()
	}
}

// ObserveDuration implements ScalableHistogram.
func (h MultiHistogram) ObserveDuration() {
	for _, histogram := range h {
		histogram.ObserveDuration()
	}
}

// Observe implements ScalableHistogram.
func (h MultiHistogram) Observe(v float64) {
	for _, histogram := range h {
		histogram.Observe(v)
	}
}

// With implements ScalableHistogram.
func (h MultiHistogram) With(labelValues ...string) ScalableHistogram {
	next := make(MultiHistogram, len(h))
	for i := range h {
		next[i] = h[i].With(labelValues...)
	}
	return next
}
