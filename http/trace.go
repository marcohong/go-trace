package http

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"

	"go-trace/trace"

	"github.com/gin-gonic/gin"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

const (
	defaultComponentName = "net/http"
)

// TraceTransport ...
type TraceTransport struct {
	internalTags []opentracing.Tag
	peerService  string
	http.RoundTripper
}

// Tracer ...
type Tracer struct {
	tr trace.Tracer
}

type closeTracker struct {
	io.ReadCloser
	tr trace.Tracer
}

func (c closeTracker) Close() error {
	err := c.ReadCloser.Close()
	c.tr.SetLog(trace.LogString(trace.LogEvent, "ClosedBody"))
	c.tr.Finish(&err)
	return err
}

// Trace is trace middleware
func Trace() gin.HandlerFunc {
	return func(c *gin.Context) {
		cx := NewContext(c)
		var t trace.Tracer
		// if request header include span return child startSpan, else return parent startSpan
		spanCtx, err := trace.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(c.Request.Header))
		if err != nil {
			t = trace.StartSpan(c.Request.URL.Path)
		} else {
			t = trace.StartSpan(c.Request.URL.Path, ext.RPCServerOption(spanCtx))
		}
		t.SetTag(trace.Tag(trace.TagComponent, defaultComponentName))
		t.SetTag(trace.Tag(trace.TagHTTPMethod, c.Request.Method))
		t.SetTag(trace.Tag(trace.TagHTTPURL, c.Request.URL.String()))
		reqCtx := trace.ContextWithSpan(cx, t.GetSpan())
		if !trace.DisableClientTrace {
			clientTrace := Tracer{t}
			reqCtx = httptrace.WithClientTrace(reqCtx, clientTrace.ClientTrace())
		}
		cx.Set(trace.CtxKey, t.GetSpan())
		// set http.Request context, because client.Get(ctx) use http.Request.Context()
		cx.Request = cx.Request.WithContext(reqCtx)
		cx.Next()
		t.Finish(nil)
	}
}

// RoundTrip ...
func (t *TraceTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	rt := t.RoundTripper
	if rt == nil {
		rt = http.DefaultTransport
	}
	tr, ok := trace.StartSpanFromContext(req.Context(), fmt.Sprintf("Client-HTTP:%s", req.Method))
	if !ok {
		return rt.RoundTrip(req)
	}
	tr.SetTag(trace.Tag(trace.TagComponent, defaultComponentName))
	tr.SetTag(trace.Tag(trace.TagHTTPMethod, req.Method))
	tr.SetTag(trace.Tag(trace.TagHTTPURL, req.URL.String()))
	tr.SetTag(trace.Tag(trace.TagSpanKind, "client"))
	if t.peerService != "" {
		tr.SetTag(trace.Tag(trace.TagPeerService, t.peerService))
	}
	if len(t.internalTags) > 0 {
		tr.SetTag(t.internalTags...)
	}
	// inject trace to http header
	tr.Inject(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(req.Header))
	resp, err := rt.RoundTrip(req)
	if err != nil {
		tr.SetTag(trace.Tag(trace.TagError, true))
		tr.Finish(&err)
		return resp, err
	}
	tr.SetTag(trace.Tag(trace.TagHTTPStatusCode, int64(resp.StatusCode)))
	if resp.StatusCode >= http.StatusInternalServerError {
		tr.SetTag(trace.Tag(trace.TagError, true))
	}
	if req.Method == "HEAD" {
		tr.Finish(nil)
	} else {
		resp.Body = closeTracker{resp.Body, tr}
	}
	return resp, nil
}

// NewClientTracer .
func NewClientTracer(req *http.Request) *Tracer {
	return &Tracer{
		tr: trace.Tracer{
			Trace: trace.GetGlobalTracer(),
		},
	}
}

// ClientTrace ...
func (t *Tracer) ClientTrace() *httptrace.ClientTrace {
	return &httptrace.ClientTrace{
		GetConn:              t.getConn,
		GotConn:              t.gotConn,
		PutIdleConn:          t.putIdleConn,
		GotFirstResponseByte: t.gotFirstResponseByte,
		Got100Continue:       t.got100Continue,
		DNSStart:             t.dnsStart,
		DNSDone:              t.dnsDone,
		ConnectStart:         t.connectStart,
		ConnectDone:          t.connectDone,
		WroteHeaders:         t.wroteHeaders,
		Wait100Continue:      t.wait100Continue,
		WroteRequest:         t.wroteRequest,
	}
}

func (t *Tracer) getConn(hostPort string) {
	t.tr.SetLog(trace.LogString(trace.LogEvent, "GetConn"), trace.LogString("hostPort", hostPort))
}

func (t *Tracer) gotConn(info httptrace.GotConnInfo) {
	t.tr.SetTag(trace.Tag("net/http.reused", info.Reused))
	t.tr.SetTag(trace.Tag("net/http.was_idle", info.WasIdle))
	t.tr.SetLog(trace.LogString(trace.LogEvent, "GotConn"))
}

func (t *Tracer) putIdleConn(error) {
	t.tr.SetLog(trace.LogString(trace.LogEvent, "PutIdleConn"))
}

func (t *Tracer) gotFirstResponseByte() {
	t.tr.SetLog(trace.LogString(trace.LogEvent, "GotFirstResponseByte"))
}

func (t *Tracer) got100Continue() {
	t.tr.SetLog(trace.LogString(trace.LogEvent, "Got100Continue"))
}

func (t *Tracer) dnsStart(info httptrace.DNSStartInfo) {
	t.tr.SetLog(
		trace.LogString(trace.LogEvent, "DNSStart"),
		trace.LogString("host", info.Host),
	)
}

func (t *Tracer) dnsDone(info httptrace.DNSDoneInfo) {
	fields := trace.LogFields(trace.LogString(trace.LogEvent, "DNSDone"))
	for _, addr := range info.Addrs {
		fields = append(fields, trace.LogString(trace.LogAddr, addr.String()))
	}
	if info.Err != nil {
		fields = append(fields, trace.LogString(trace.LogErrorObject, info.Err.Error()))
	}
	t.tr.SetLog(fields...)
}

func (t *Tracer) connectStart(network, addr string) {
	t.tr.SetLog(
		trace.LogString(trace.LogEvent, "ConnectStart"),
		trace.LogString(trace.LogNetwork, network),
		trace.LogString(trace.LogAddr, addr),
	)
}

func (t *Tracer) connectDone(network, addr string, err error) {
	if err != nil {
		t.tr.SetLog(
			trace.LogString(trace.LogMessage, "ConnectDone"),
			trace.LogString(trace.LogNetwork, network),
			trace.LogString(trace.LogAddr, addr),
			trace.LogString(trace.LogEvent, "error"),
			trace.LogString(trace.LogErrorObject, err.Error()),
		)
	} else {
		t.tr.SetLog(
			trace.LogString(trace.LogEvent, "ConnectDone"),
			trace.LogString(trace.LogNetwork, network),
			trace.LogString(trace.LogAddr, addr),
		)
	}
}

func (t *Tracer) wroteHeaders() {
	t.tr.SetLog(trace.LogString(trace.LogEvent, "WroteHeaders"))
}

func (t *Tracer) wait100Continue() {
	t.tr.SetLog(trace.LogString(trace.LogEvent, "Wait100Continue"))
}

func (t *Tracer) wroteRequest(info httptrace.WroteRequestInfo) {
	if info.Err != nil {
		t.tr.SetLog(
			trace.LogString(trace.LogMessage, "WroteRequest"),
			trace.LogString(trace.LogEvent, "error"),
			trace.LogString(trace.LogErrorObject, info.Err.Error()),
		)
		t.tr.SetTag(trace.Tag(trace.TagError, true))
	} else {
		t.tr.SetLog(trace.LogString(trace.LogEvent, "WroteRequest"))
	}
}
