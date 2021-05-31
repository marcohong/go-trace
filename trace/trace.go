package trace

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
)

var (
	// global tracer
	_tracer opentracing.Tracer
	maxTags = 128
	maxLogs = 256
	// DisableClientTrace .
	DisableClientTrace = false
	// CtxKey gin.Context trace key
	CtxKey = "library/net/trace.trace"
)

// Config trace config
type Config struct {
	ServiceName        string
	OpenReporter       bool
	Stdlog             bool
	ReportHost         string        // host:port -> 127.0.0.1:9941
	SamplerType        string        //const, probabilistic, rateLimiting, or remote
	SamplerParam       float64       // 0 or 1
	FlushInterval      time.Duration // second, default 1
	DisableClientTrace bool
}

// SetGlobalTracer set global tracer
func SetGlobalTracer(tracer opentracing.Tracer) {
	_tracer = tracer
	opentracing.SetGlobalTracer(tracer)
}

// GetGlobalTracer return global tracer
func GetGlobalTracer() opentracing.Tracer {
	return _tracer
}

// NewTracer return Tracer
func NewTracer(c *Config) (opentracing.Tracer, io.Closer) {
	cfg := &config.Configuration{
		ServiceName: c.ServiceName,
		Sampler: &config.SamplerConfig{
			Type:  c.SamplerType,
			Param: c.SamplerParam,
		},
		Reporter: &config.ReporterConfig{
			LocalAgentHostPort:  c.ReportHost,
			BufferFlushInterval: time.Duration(c.FlushInterval),
			LogSpans:            c.OpenReporter,
		},
	}
	// jaeger.StdLogger
	opts := []config.Option{}
	if c.Stdlog {
		opts = append(opts, config.Logger(jaeger.StdLogger))
	}
	tracer, closer, err := cfg.NewTracer(opts...)
	if err != nil {
		panic(fmt.Sprintf("Init trace error: %v\n", err))
	}
	SetGlobalTracer(tracer)
	DisableClientTrace = c.DisableClientTrace
	return tracer, closer
}

// ContextWithSpan returns a new `context.Context`
func ContextWithSpan(ctx context.Context, span opentracing.Span) context.Context {
	return opentracing.ContextWithSpan(ctx, span)
}

// Extract returns a Trace instance given `format` and `carrier`.
// return `ErrTraceNotFound` if trace not found.
func Extract(format interface{}, carrier interface{}) (opentracing.SpanContext, error) {
	return _tracer.Extract(format, carrier)
}

// StartSpan  Create, start, and return a new Span with the given `operationName` and
// incorporate the given StartSpanOption `opts`.
func StartSpan(operationName string, opts ...opentracing.StartSpanOption) Tracer {
	span := _tracer.StartSpan(operationName, opts...)
	return New(span)
}

// StartSpanFromContext if context contains parent, return child span
func StartSpanFromContext(ctx context.Context, operationName string, opts ...opentracing.StartSpanOption) (Tracer, bool) {
	parent := opentracing.SpanFromContext(ctx)
	var tracer Tracer
	if parent != nil {
		opts = append(opts, opentracing.ChildOf(parent.Context()))
	} else {
		if val, ok := ctx.Value(CtxKey).(opentracing.Span); ok {
			opts = append(opts, opentracing.ChildOf(val.(opentracing.Span).Context()))
		} else {
			return tracer, false
		}
	}
	span := _tracer.StartSpan(operationName, opts...)
	tracer = New(span)
	return tracer, true
}

// StartSpanFromContextV2 if context contains parent, return child span
func StartSpanFromContextV2(ctx context.Context, operationName string, opts ...opentracing.StartSpanOption) (Tracer, bool) {
	var tracer Tracer
	if val, ok := ctx.Value(CtxKey).(opentracing.Span); ok {
		opts = append(opts, opentracing.ChildOf(val.(opentracing.Span).Context()))
	} else {
		return tracer, false
	}
	span := _tracer.StartSpan(operationName, opts...)
	tracer = New(span)
	return tracer, true
}

// Tag return opentracing.tag struct
func Tag(key string, value interface{}) opentracing.Tag {
	return opentracing.Tag{Key: key, Value: value}
}

// Tracer struct
type Tracer struct {
	Trace   opentracing.Tracer
	span    opentracing.Span
	tags    []opentracing.Tag
	maxLogs int
}

// New returns a new Tracer
func New(span opentracing.Span) Tracer {
	t := Tracer{
		Trace: GetGlobalTracer(),
		span:  span,
	}
	return t
}

// NewWithTrace returns a new Tracer with base trace
func NewWithTrace(trace opentracing.Tracer, span opentracing.Span) Tracer {
	t := Tracer{
		Trace: trace,
		span:  span,
	}
	return t
}

// Fork a new Tracer
func (t *Tracer) Fork(operationName string, opts ...opentracing.StartSpanOption) Tracer {
	opts = append(opts, opentracing.ChildOf(t.span.Context()))
	span := t.Trace.StartSpan(operationName, opts...)
	return NewWithTrace(t.Trace, span)
}

// Extract returns a Trace instance given `format` and `carrier`.
// return `ErrTraceNotFound` if trace not found.
func (t *Tracer) Extract(format interface{}, carrier interface{}) (opentracing.SpanContext, error) {
	return t.Trace.Extract(format, carrier)
}

// Inject span inject
func (t *Tracer) Inject(format interface{}, carrier interface{}) error {
	return t.span.Tracer().Inject(t.span.Context(), format, carrier)
}

// StartSpan  Create, start, and return a new Span with the given `operationName` and
// incorporate the given StartSpanOption `opts`.
func (t *Tracer) StartSpan(operationName string, opts ...opentracing.StartSpanOption) Tracer {
	span := t.Trace.StartSpan(operationName, opts...)
	return NewWithTrace(t.Trace, span)
}

// SpanFromContext .
func SpanFromContext(ctx context.Context) (t Tracer, ok bool) {
	parent := opentracing.SpanFromContext(ctx)
	if parent != nil {
		t = New(parent)
		ok = true
	}
	return
}

// StartSpanFromContext if context contains parent, return child span
func (t *Tracer) StartSpanFromContext(ctx context.Context, operationName string, opts ...opentracing.StartSpanOption) (Tracer, bool) {
	parent := opentracing.SpanFromContext(ctx)
	if parent != nil {
		opts = append(opts, opentracing.ChildOf(parent.Context()))
	} else {
		if val, ok := ctx.Value(CtxKey).(opentracing.Span); ok {
			opts = append(opts, opentracing.ChildOf(val.(opentracing.Span).Context()))
		}
	}
	span := t.Trace.StartSpan(operationName, opts...)
	//  ContextWithSpan(ctx, span)
	return NewWithTrace(t.Trace, span), true
}

// ContextWithSpan return span context
func (t *Tracer) ContextWithSpan(ctx context.Context) context.Context {
	return opentracing.ContextWithSpan(ctx, t.span)

}

// Finish when trace finish call it.
func (t *Tracer) Finish(err *error) {
	if t.span != nil {
		t.span.Finish()
	}
}

// SetTag Adds a tag to the trace.
func (t *Tracer) SetTag(tags ...opentracing.Tag) *Tracer {
	if len(tags) < maxTags {
		t.tags = append(t.tags, tags...)
	}
	if len(tags) == maxTags {
		t.tags = append(t.tags, opentracing.Tag{Key: "trace.error", Value: "too many tags"})
	}
	for _, tag := range tags {
		t.span.SetTag(tag.Key, tag.Value)
	}
	return t
}

// SetLog is an efficient and type-checked way to record key:value
// NOTE current unsupport
func (t *Tracer) SetLog(logs ...log.Field) *Tracer {
	if t.maxLogs+len(logs) < maxLogs {
		t.span.LogFields(logs...)
	} else {
		t.span.LogFields(LogString("trace.error", "too many logs"))
	}
	return t
}

// GetSpan return trace span
func (t *Tracer) GetSpan() opentracing.Span {
	return t.span
}

// SetTitle reset trace title
func (t *Tracer) SetTitle(title string) {
	t.span.SetOperationName(title)
}

// LogFields returns fields
func LogFields(field log.Field) []log.Field {
	return []log.Field{field}
}

// LogString .
func LogString(key, val string) log.Field { return log.String(key, val) }

// LogBool .
func LogBool(key string, val bool) log.Field { return log.Bool(key, val) }

// LogInt .
func LogInt(key string, val int) log.Field { return log.Int(key, val) }

// LogInt32 .
func LogInt32(key string, val int32) log.Field { return log.Int32(key, val) }

// LogInt64 .
func LogInt64(key string, val int64) log.Field { return log.Int64(key, val) }

// LogUint32 .
func LogUint32(key string, val uint32) log.Field { return log.Uint32(key, val) }

// LogUint64 .
func LogUint64(key string, val uint64) log.Field { return log.Uint64(key, val) }

//LogFloat32 .
func LogFloat32(key string, val float32) log.Field { return log.Float32(key, val) }

// LogFloat64 .
func LogFloat64(key string, val float64) log.Field { return log.Float64(key, val) }

//LogError .
func LogError(err error) log.Field { return log.Error(err) }

// LogObject .
func LogObject(key string, val interface{}) log.Field { return log.Object(key, val) }
