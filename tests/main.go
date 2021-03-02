package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"time"

	rpcConf "go-trace/rpc"
	pb "go-trace/tests/test"
	"go-trace/trace"

	"github.com/opentracing/opentracing-go"
	grpc "google.golang.org/grpc"
)

const (
	// Port rpc
	Port = ":50001"
)

//TestServer ...
type TestServer struct{}

// SayHello ...
func (t *TestServer) SayHello(cxt context.Context, in *pb.HelloRequest) (*pb.HelloResponse, error) {
	return &pb.HelloResponse{Name: "hello: " + in.Name}, nil
}

func newTracer(serviceName string) (opentracing.Tracer, io.Closer) {
	t, c := trace.NewTracer(&trace.Config{
		ServiceName:        serviceName,
		OpenReporter:       true,                           // open jaeger reporter
		Stdlog:             true,                           // log stdout
		ReportHost:         "127.0.0.1:6831",               // host:port -> 127.0.0.1:6831
		SamplerType:        "const",                        //const, probabilistic, rateLimiting, or remote
		SamplerParam:       1,                              // 0 or 1
		FlushInterval:      time.Duration(1 * time.Second), // second, default 1
		DisableClientTrace: false,                          // open client trace
	})
	return t, c
}

func main() {
	runServer()
	runClient()
}

func runServer() {
	tr, close := newTracer("Trace-test-server")
	defer close.Close()
	lis, err := net.Listen("tcp", Port)
	if err != nil {
		panic(err)
	}
	inter := rpcConf.OpentracingServerInterceptor(trace.Tracer{Trace: tr})
	unary := rpcConf.ChainUnaryServer(inter)
	s := grpc.NewServer(grpc.ChainUnaryInterceptor(unary))
	pb.RegisterTestServer(s, &TestServer{})
	go s.Serve(lis)
}

func runClient() {
	tr, close := newTracer("Trace-test-client")
	defer close.Close()
	inter := rpcConf.OpentracingClientInterceptor(trace.Tracer{Trace: tr})
	unary := rpcConf.ChainUnaryClient(inter)
	conn, err := grpc.Dial("localhost:50001", grpc.WithInsecure(), grpc.WithChainUnaryInterceptor(unary))
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	c := pb.NewTestClient(conn)
	for i := 0; i < 3; i++ {
		start := time.Now().UnixNano() / 1e3
		name := fmt.Sprintf("data-%v", i)
		r, err := c.SayHello(context.Background(), &pb.HelloRequest{Name: name})
		if err != nil {
			panic(err)
		}
		ended := time.Now().UnixNano()/1e3 - start
		fmt.Printf("No: %d, Resp: '%s', Times: %dµs\n", i, r.Name, ended)
	}
}
