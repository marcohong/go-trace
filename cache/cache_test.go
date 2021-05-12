package cache

import (
	"context"
	"go-trace/trace"
	"io"
	"testing"
	"time"
)

var (
	closer io.Closer
	tr     trace.Tracer
)

func init() {
	conf := &trace.Config{
		ServiceName:        "Trace-cache-service",
		OpenReporter:       true,                           // open jaeger reporter
		Stdlog:             true,                           // log stdout
		ReportHost:         "127.0.0.1:6831",               // host:port -> 127.0.0.1:6831
		SamplerType:        "const",                        //const, probabilistic, rateLimiting, or remote
		SamplerParam:       1,                              // 0 or 1
		FlushInterval:      time.Duration(1 * time.Second), // second, default 1
		DisableClientTrace: false,
	}
	_, closer = trace.NewTracer(conf)
	tr = trace.StartSpan("redis-trace")
}

func TestRedisCache(t *testing.T) {
	conf := &Config{
		Proto: "tcp",
		Addr:  "127.0.0.1:6379",
		DB:    1,
	}
	ctx := context.WithValue(context.Background(), trace.CtxKey, tr.GetSpan())
	defer closer.Close()

	client := New(conf)
	defer client.Close()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Fatalf("connect redis  error: %v", err)
	}
	client.Set(ctx, "test-1", "test", time.Minute)
	val := client.Get(ctx, "test-1").Val()
	if val == "" || val != "test" {
		t.Fatalf("redis value error: not euqal or nil")
	}
	client.Del(ctx, "test-1")
	t.Logf("redis value: %v", val)
}
