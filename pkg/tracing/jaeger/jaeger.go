package jaeger

import (
	"fmt"
	"io"

	"github.com/opentracing/opentracing-go"
	"github.com/traefik/traefik/v2/pkg/log"
	jaegercli "github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	"github.com/uber/jaeger-client-go/zipkin"
	jaegermet "github.com/uber/jaeger-lib/metrics"
)

// Name sets the name of this tracer.
const Name = "jaeger"

// Config provides configuration settings for a jaeger tracer.
type Config struct {
	SamplingServerURL          string     `description:"Set the sampling server url." json:"samplingServerURL,omitempty" toml:"samplingServerURL,omitempty" yaml:"samplingServerURL,omitempty"`
	SamplingType               string     `description:"Set the sampling type." json:"samplingType,omitempty" toml:"samplingType,omitempty" yaml:"samplingType,omitempty" export:"true"`
	SamplingParam              float64    `description:"Set the sampling parameter." json:"samplingParam,omitempty" toml:"samplingParam,omitempty" yaml:"samplingParam,omitempty" export:"true"`
	LocalAgentHostPort         string     `description:"Set jaeger-agent's host:port that the reporter will used." json:"localAgentHostPort,omitempty" toml:"localAgentHostPort,omitempty" yaml:"localAgentHostPort,omitempty"`
	Gen128Bit                  bool       `description:"Generate 128 bit span IDs." json:"gen128Bit,omitempty" toml:"gen128Bit,omitempty" yaml:"gen128Bit,omitempty" export:"true"`
	Propagation                string     `description:"Which propagation format to use (jaeger/b3)." json:"propagation,omitempty" toml:"propagation,omitempty" yaml:"propagation,omitempty" export:"true"`
	TraceContextHeaderName     string     `description:"Set the header to use for the trace-id." json:"traceContextHeaderName,omitempty" toml:"traceContextHeaderName,omitempty" yaml:"traceContextHeaderName,omitempty" export:"true"`
	Collector                  *Collector `description:"Define the collector information" json:"collector,omitempty" toml:"collector,omitempty" yaml:"collector,omitempty" export:"true"`
	DisableAttemptReconnecting bool       `description:"Disable the periodic re-resolution of the agent's hostname and reconnection if there was a change." json:"disableAttemptReconnecting,omitempty" toml:"disableAttemptReconnecting,omitempty" yaml:"disableAttemptReconnecting,omitempty" export:"true"`
}

// SetDefaults sets the default values.
func (c *Config) SetDefaults() {
	c.SamplingServerURL = "http://localhost:5778/sampling"
	c.SamplingType = "const"
	c.SamplingParam = 1.0
	c.LocalAgentHostPort = "127.0.0.1:6831"
	c.Propagation = "jaeger"
	c.Gen128Bit = false
	c.TraceContextHeaderName = jaegercli.TraceContextHeaderName
	c.DisableAttemptReconnecting = true
}

// Collector provides configuration settings for jaeger collector.
type Collector struct {
	Endpoint string `description:"Instructs reporter to send spans to jaeger-collector at this URL." json:"endpoint,omitempty" toml:"endpoint,omitempty" yaml:"endpoint,omitempty"`
	User     string `description:"User for basic http authentication when sending spans to jaeger-collector." json:"user,omitempty" toml:"user,omitempty" yaml:"user,omitempty"`
	Password string `description:"Password for basic http authentication when sending spans to jaeger-collector." json:"password,omitempty" toml:"password,omitempty" yaml:"password,omitempty"`
}

// SetDefaults sets the default values.
func (c *Collector) SetDefaults() {
	c.Endpoint = ""
	c.User = ""
	c.Password = ""
}

// Setup sets up the tracer.
func (c *Config) Setup(componentName string) (opentracing.Tracer, io.Closer, error) {
	reporter := &jaegercfg.ReporterConfig{
		LogSpans:                   true,
		LocalAgentHostPort:         c.LocalAgentHostPort,
		DisableAttemptReconnecting: c.DisableAttemptReconnecting,
	}

	if c.Collector != nil {
		reporter.CollectorEndpoint = c.Collector.Endpoint
		reporter.User = c.Collector.User
		reporter.Password = c.Collector.Password
	}

	jcfg := &jaegercfg.Configuration{
		Sampler: &jaegercfg.SamplerConfig{
			SamplingServerURL: c.SamplingServerURL,
			Type:              c.SamplingType,
			Param:             c.SamplingParam,
		},
		Reporter: reporter,
		Headers: &jaegercli.HeadersConfig{
			TraceContextHeaderName: c.TraceContextHeaderName,
		},
	}

	// Overrides existing tracer's Configuration with environment variables.
	_, err := jcfg.FromEnv()
	if err != nil {
		return nil, nil, err
	}

	jMetricsFactory := jaegermet.NullFactory

	opts := []jaegercfg.Option{
		jaegercfg.Logger(newJaegerLogger()),
		jaegercfg.Metrics(jMetricsFactory),
		jaegercfg.Gen128Bit(c.Gen128Bit),
	}

	switch c.Propagation {
	case "b3":
		p := zipkin.NewZipkinB3HTTPHeaderPropagator()
		opts = append(opts,
			jaegercfg.Injector(opentracing.HTTPHeaders, p),
			jaegercfg.Extractor(opentracing.HTTPHeaders, p),
		)
	case "jaeger", "":
	default:
		return nil, nil, fmt.Errorf("unknown propagation format: %s", c.Propagation)
	}

	// Initialize tracer with a logger and a metrics factory
	closer, err := jcfg.InitGlobalTracer(
		componentName,
		opts...,
	)
	if err != nil {
		log.WithoutContext().Warnf("Could not initialize jaeger tracer: %v", err)
		return nil, nil, err
	}
	log.WithoutContext().Debug("Jaeger tracer configured")

	return opentracing.GlobalTracer(), closer, nil
}
