package metrics

import (
	"github.com/containous/traefik/types"
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

const (
	metricNamePrefix = "traefik_"

	reqsTotalName    = metricNamePrefix + "requests_total"
	reqDurationName  = metricNamePrefix + "request_duration_seconds"
	retriesTotalName = metricNamePrefix + "backend_retries_total"
)

// RegisterPrometheus registers all Prometheus metrics.
// It must be called only once and failing to register the metrics will lead to a panic.
func RegisterPrometheus(config *types.Prometheus) Registry {
	buckets := []float64{0.1, 0.3, 1.2, 5.0}
	if config.Buckets != nil {
		buckets = config.Buckets
	}

	reqCounter := prometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Name: reqsTotalName,
		Help: "How many HTTP requests processed, partitioned by status code and method.",
	}, []string{"service", "code", "method"})
	reqDurationHistogram := prometheus.NewHistogramFrom(stdprometheus.HistogramOpts{
		Name:    reqDurationName,
		Help:    "How long it took to process the request.",
		Buckets: buckets,
	}, []string{"service", "code"})
	retryCounter := prometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Name: retriesTotalName,
		Help: "How many request retries happened in total.",
	}, []string{"service"})

	return &prometheusRegistry{
		reqCounter:           reqCounter,
		reqDurationHistogram: reqDurationHistogram,
		retryCounter:         retryCounter,
	}
}

type prometheusRegistry struct {
	reqCounter           metrics.Counter
	reqDurationHistogram metrics.Histogram
	retryCounter         metrics.Counter
}

func (r *prometheusRegistry) ReqsCounter() metrics.Counter {
	return r.reqCounter
}

func (r *prometheusRegistry) ReqDurationHistogram() metrics.Histogram {
	return r.reqDurationHistogram
}

func (r *prometheusRegistry) RetriesCounter() metrics.Counter {
	return r.retryCounter
}
