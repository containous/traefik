package tracing

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/traefik/traefik/v2/pkg/log"
)

type contextKey int

const (
	// SpanKindNoneEnum Span kind enum none.
	SpanKindNoneEnum ext.SpanKindEnum = "none"
	tracingKey       contextKey       = iota
)

// WithTracing Adds Tracing into the context.
func WithTracing(ctx context.Context, tracing *Tracing) context.Context {
	return context.WithValue(ctx, tracingKey, tracing)
}

// FromContext Gets Tracing from context.
func FromContext(ctx context.Context) (*Tracing, error) {
	if ctx == nil {
		panic("nil context")
	}

	tracer, ok := ctx.Value(tracingKey).(*Tracing)
	if !ok {
		return nil, errors.New("unable to find tracing in the context")
	}
	return tracer, nil
}

// Backend is an abstraction for tracking backend (Jaeger, Zipkin, ...).
type Backend interface {
	Setup(componentName string) (opentracing.Tracer, io.Closer, error)
}

// Tracing middleware.
type Tracing struct {
	ServiceName   string `description:"Set the name for this service" export:"true"`
	SpanNameLimit int    `description:"Set the maximum character limit for Span names (default 0 = no limit)" export:"true"`

	tracer opentracing.Tracer
	closer io.Closer
}

// NewTracing Creates a Tracing.
func NewTracing(serviceName string, spanNameLimit int, tracingBackend Backend) (*Tracing, error) {
	tracing := &Tracing{
		ServiceName:   serviceName,
		SpanNameLimit: spanNameLimit,
	}

	var err error
	tracing.tracer, tracing.closer, err = tracingBackend.Setup(serviceName)
	if err != nil {
		return nil, err
	}
	return tracing, nil
}

// StartSpan delegates to opentracing.Tracer.
func (t *Tracing) StartSpan(operationName string, opts ...opentracing.StartSpanOption) opentracing.Span {
	return t.tracer.StartSpan(operationName, opts...)
}

// StartSpanf delegates to StartSpan.
func (t *Tracing) StartSpanf(r *http.Request, spanKind ext.SpanKindEnum, opPrefix string, opParts []string, separator string, opts ...opentracing.StartSpanOption) (opentracing.Span, *http.Request, func()) {
	operationName := generateOperationName(opPrefix, opParts, separator, t.SpanNameLimit)

	return StartSpan(r, operationName, spanKind, opts...)
}

// Inject delegates to opentracing.Tracer.
func (t *Tracing) Inject(sm opentracing.SpanContext, format, carrier interface{}) error {
	return t.tracer.Inject(sm, format, carrier)
}

// Extract delegates to opentracing.Tracer.
func (t *Tracing) Extract(format, carrier interface{}) (opentracing.SpanContext, error) {
	return t.tracer.Extract(format, carrier)
}

// IsEnabled determines if tracing was successfully activated.
func (t *Tracing) IsEnabled() bool {
	return t != nil && t.tracer != nil
}

// Close tracer.
func (t *Tracing) Close() {
	if t.closer != nil {
		err := t.closer.Close()
		if err != nil {
			log.WithoutContext().Warn(err)
		}
	}
}

// LogRequest used to create span tags from the request.
func LogRequest(span opentracing.Span, r *http.Request) {
	if span != nil && r != nil {
		ext.HTTPMethod.Set(span, r.Method)
		ext.HTTPUrl.Set(span, r.URL.String())
		span.SetTag("http.host", r.Host)
	}
}

// LogResponseCode used to log response code in span.
func LogResponseCode(span opentracing.Span, code int) {
	if span != nil {
		ext.HTTPStatusCode.Set(span, uint16(code))
		if code >= http.StatusInternalServerError {
			ext.Error.Set(span, true)
		}
	}
}

// GetSpan used to retrieve span from request context.
func GetSpan(r *http.Request) opentracing.Span {
	return opentracing.SpanFromContext(r.Context())
}

// InjectRequestHeaders used to inject OpenTracing headers into the request.
func InjectRequestHeaders(r *http.Request) {
	if span := GetSpan(r); span != nil {
		err := opentracing.GlobalTracer().Inject(
			span.Context(),
			opentracing.HTTPHeaders,
			opentracing.HTTPHeadersCarrier(r.Header))
		if err != nil {
			log.FromContext(r.Context()).Error(err)
		}
	}
}

// LogEventf logs an event to the span in the request context.
func LogEventf(r *http.Request, format string, args ...interface{}) {
	if span := GetSpan(r); span != nil {
		span.LogKV("event", fmt.Sprintf(format, args...))
	}
}

// StartSpan starts a new span from the one in the request context.
func StartSpan(r *http.Request, operationName string, spanKind ext.SpanKindEnum, opts ...opentracing.StartSpanOption) (opentracing.Span, *http.Request, func()) {
	span, ctx := opentracing.StartSpanFromContext(r.Context(), operationName, opts...)

	switch spanKind {
	case ext.SpanKindRPCClientEnum:
		ext.SpanKindRPCClient.Set(span)
	case ext.SpanKindRPCServerEnum:
		ext.SpanKindRPCServer.Set(span)
	case ext.SpanKindProducerEnum:
		ext.SpanKindProducer.Set(span)
	case ext.SpanKindConsumerEnum:
		ext.SpanKindConsumer.Set(span)
	default:
		// noop
	}

	r = r.WithContext(ctx)
	return span, r, func() { span.Finish() }
}

// SetError flags the span associated with this request as in error.
func SetError(r *http.Request) {
	if span := GetSpan(r); span != nil {
		ext.Error.Set(span, true)
	}
}

// SetErrorWithEvent flags the span associated with this request as in error and log an event.
func SetErrorWithEvent(r *http.Request, format string, args ...interface{}) {
	SetError(r)
	LogEventf(r, format, args...)
}
