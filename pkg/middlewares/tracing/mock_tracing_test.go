package tracing

import (
	"io"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
)

type MockTracer struct {
	Span *MockSpan
}

// StartSpan belongs to the Tracer interface.
func (n MockTracer) StartSpan(operationName string, opts ...opentracing.StartSpanOption) opentracing.Span {
	n.Span.OpName = operationName
	return n.Span
}

// Inject belongs to the Tracer interface.
func (n MockTracer) Inject(sp opentracing.SpanContext, format, carrier interface{}) error {
	return nil
}

// Extract belongs to the Tracer interface.
func (n MockTracer) Extract(format, carrier interface{}) (opentracing.SpanContext, error) {
	return nil, opentracing.ErrSpanContextNotFound
}

// MockSpanContext a span context mock.
type MockSpanContext struct{}

func (n MockSpanContext) ForeachBaggageItem(handler func(k, v string) bool) {}

// MockSpan a span mock.
type MockSpan struct {
	OpName string
	Tags   map[string]interface{}
}

func (n MockSpan) Context() opentracing.SpanContext { return MockSpanContext{} }
func (n MockSpan) SetBaggageItem(key, val string) opentracing.Span {
	return MockSpan{Tags: make(map[string]interface{})}
}
func (n MockSpan) BaggageItem(key string) string { return "" }
func (n MockSpan) SetTag(key string, value interface{}) opentracing.Span {
	n.Tags[key] = value
	return n
}
func (n MockSpan) LogFields(fields ...log.Field)                          {}
func (n MockSpan) LogKV(keyVals ...interface{})                           {}
func (n MockSpan) Finish()                                                {}
func (n MockSpan) FinishWithOptions(opts opentracing.FinishOptions)       {}
func (n MockSpan) SetOperationName(operationName string) opentracing.Span { return n }
func (n MockSpan) Tracer() opentracing.Tracer                             { return MockTracer{} }
func (n MockSpan) LogEvent(event string)                                  {}
func (n MockSpan) LogEventWithPayload(event string, payload interface{})  {}
func (n MockSpan) Log(data opentracing.LogData)                           {}
func (n *MockSpan) Reset() {
	n.Tags = make(map[string]interface{})
}

type trackingBackenMock struct {
	tracer opentracing.Tracer
}

func (t *trackingBackenMock) Setup(componentName string) (opentracing.Tracer, io.Closer, error) {
	opentracing.SetGlobalTracer(t.tracer)
	return t.tracer, nil, nil
}
