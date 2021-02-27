package main

import (
	"go-trace/http"
	"go-trace/trace"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

var (
	client  *http.Client
	baseURL = "http://localhost:8888"
)

type resp struct {
	Code int    `json:"code"`
	Data string `json:"data"`
}

func hello(c *http.Context) {
	var res resp
	if err := client.Get(c.Request.Context(), baseURL+"/permit", nil, res); err != nil {
		log.Infof("hello request permit, resp:%s", res.Data)
	}
	c.Jsonify("hello")
}

func permit(c *http.Context) {
	var res resp
	if err := client.Get(c.Request.Context(), baseURL+"/verify", nil, res); err != nil {
		log.Infof("hello request verify, resp:%s", res.Data)
	}
	c.Jsonify("permit")
}

func verify(c *http.Context) {
	c.Jsonify("verify")
}

func addRoutes(e *http.Engine) {
	e.GET("/hello", http.Handle(hello))
	e.GET("/verify", http.Handle(verify))
	e.GET("/permit", http.Handle(permit))
}

func main() {
	conf := &http.Config{
		Addr:    ":8888",
		Timeout: time.Duration(1),
	}
	clientConf := &http.ClientConfig{
		Dial:      time.Duration(100 * time.Millisecond),
		Timeout:   time.Duration(300 * time.Millisecond),
		KeepAlive: time.Duration(60 * time.Second),
	}
	client = http.NewClient(clientConf)
	engine := http.NewEngine(conf)
	addRoutes(engine)
	http.InitTracer(&trace.Config{
		ServiceName:        "Trace-test-server",
		OpenReporter:       true,
		Stdlog:             true,
		ReportHost:         "127.0.0.1:6831", // host:port -> 127.0.0.1:9941
		SamplerType:        "const",          //const, probabilistic, rateLimiting, or remote
		SamplerParam:       1,                // 0 or 1
		FlushInterval:      time.Duration(1), // second, default 1
		DisableClientTrace: false,
	})
	http.Start(conf, engine)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM)
	for {
		s := <-quit
		log.Info("Got a signal: %s", s.String())
		switch s {
		case syscall.SIGQUIT, syscall.SIGINT, syscall.SIGTERM, syscall.SIGSTOP:
			http.Shutdown()
			return
		case syscall.SIGHUP:
		default:
			return
		}
	}
}
