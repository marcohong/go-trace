package cache

import (
	"context"
	"fmt"

	"go-trace/trace"

	"github.com/go-redis/redis/extra/rediscmd/v8"
	"github.com/go-redis/redis/v8"
	// "go.opentelemetry.io/otel/trace"
)

// TracingHook .
type TracingHook struct{}

var _ redis.Hook = (*TracingHook)(nil)

// NewTracingHook .
func NewTracingHook() *TracingHook {
	return new(TracingHook)
}

func setTags(tr *trace.Tracer) {
	tr.SetTag(trace.Tag(trace.TagPeerService, "redis"))
	tr.SetTag(trace.Tag(trace.TagComponent, "cache/redis"))
	tr.SetTag(trace.Tag(trace.TagSpanKind, "client"))
}

// BeforeProcess .
func (TracingHook) BeforeProcess(ctx context.Context, cmd redis.Cmder) (context.Context, error) {
	tr, ok := trace.StartSpanFromContextV2(ctx, fmt.Sprintf("Redis:%s", cmd.FullName()))
	if !ok {
		return ctx, nil
	}
	setTags(&tr)
	tr.SetTag(trace.Tag(trace.TagDBStatement, rediscmd.CmdString(cmd)))
	ctx = tr.ContextWithSpan(ctx)
	return ctx, nil
}

// AfterProcess .
func (TracingHook) AfterProcess(ctx context.Context, cmd redis.Cmder) (err error) {
	tr, ok := trace.SpanFromContext(ctx)
	if !ok {
		return nil
	}
	defer tr.Finish(&err)
	if err = cmd.Err(); err != nil {
		recordError(ctx, &tr, err)
	}
	return nil
}

// BeforeProcessPipeline .
func (TracingHook) BeforeProcessPipeline(ctx context.Context, cmds []redis.Cmder) (context.Context, error) {
	tr, ok := trace.StartSpanFromContextV2(ctx, "Redis:Pipeline")
	if !ok {
		return ctx, nil
	}
	setTags(&tr)
	_, cmdsString := rediscmd.CmdsString(cmds)
	tr.SetTag(trace.Tag(trace.TagDBStatement, cmdsString))
	tr.SetTag(trace.Tag("db.redis.num_cmd", len(cmds)))
	ctx = tr.ContextWithSpan(ctx)
	return ctx, nil
}

// AfterProcessPipeline .
func (TracingHook) AfterProcessPipeline(ctx context.Context, cmds []redis.Cmder) (err error) {
	span, ok := trace.SpanFromContext(ctx)
	if !ok {
		return nil
	}
	defer span.Finish(&err)
	if err := cmds[0].Err(); err != nil {
		recordError(ctx, &span, err)
	}
	return nil
}

func recordError(ctx context.Context, t *trace.Tracer, err error) {
	if err != redis.Nil {
		t.SetTag(trace.Tag(trace.TagError, true))
		t.SetLog(trace.LogString(trace.LogEvent, "redis error"), trace.LogString(trace.LogMessage, err.Error()))
	}
}
