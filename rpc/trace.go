package rpc

import (
	"context"
	"strings"

	"go-trace/trace"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// MDReaderWriter ...
type MDReaderWriter struct {
	metadata.MD
}

// Set conforms to the TextMapWriter interface.
func (c MDReaderWriter) Set(key, val string) {
	key = strings.ToLower(key)
	c.MD[key] = append(c.MD[key], val)
}

// ForeachKey conforms to the TextMapReader interface.
func (c MDReaderWriter) ForeachKey(handler func(key, val string) error) error {
	for k, vals := range c.MD {
		for _, v := range vals {
			if err := handler(k, v); err != nil {
				return err
			}
		}
	}
	return nil
}

// OpentracingServerInterceptor rewrite server's interceptor with open tracing
func OpentracingServerInterceptor(t trace.Tracer) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		}
		tag := trace.Tag(string(trace.TagComponent), "gRPC")
		spanCtx, err := t.Extract(opentracing.TextMap, MDReaderWriter{md})
		if err != nil {
			t = t.StartSpan(info.FullMethod, tag)
		} else {
			t = t.StartSpan(info.FullMethod, ext.RPCServerOption(spanCtx), tag, ext.SpanKindRPCServer)
		}
		defer t.Finish(nil)
		// ctx = trace.ContextWithSpan(ctx, t.GetSpan())
		ctx = context.WithValue(ctx, trace.CtxKey, t.GetSpan())
		return handler(ctx, req)
	}
}

// OpentracingClientInterceptor rewrite client's interceptor with open tracing
func OpentracingClientInterceptor(t trace.Tracer) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		tr, _ := t.StartSpanFromContext(ctx, method, ext.SpanKindRPCClient)
		_ = tr.SetTag(trace.Tag(trace.TagComponent, "gRPC"))
		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		} else {
			md = md.Copy()
		}
		mdWriter := MDReaderWriter{md}
		err := tr.Inject(opentracing.TextMap, mdWriter)
		if err != nil {
			tr.SetTag(trace.Tag(trace.TagError, true))
		}
		ctx = metadata.NewOutgoingContext(ctx, md)
		err = invoker(ctx, method, req, reply, cc, opts...)
		if err != nil {
			tr.SetTag(trace.Tag(trace.TagError, err.Error()))
		}
		tr.Finish(&err)
		return err
	}
}
