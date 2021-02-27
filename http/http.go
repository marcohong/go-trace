package http

import (
	"context"
	"errors"
	"go-trace/trace"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

var (
	svr    *http.Server
	closer io.Closer
)

// Config http server configure
type Config struct {
	Addr         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	Timeout      time.Duration
}

// Engine .
type Engine struct {
	*gin.Engine
	Conf *Config
}

// NewEngine create http server engine
func NewEngine(c *Config) *Engine {
	e := gin.New()
	engine := &Engine{Engine: e, Conf: c}
	engine.Use(gin.LoggerWithFormatter(GinLogFormatter))
	engine.Use(gin.Recovery(), Trace())
	return engine
}

// Start start http server
func Start(c *Config, e *Engine) {
	svr = &http.Server{
		Handler:      e,
		Addr:         c.Addr,
		ReadTimeout:  c.ReadTimeout,
		WriteTimeout: c.WriteTimeout,
	}
	go func() {
		log.Infof("Listening and serving HTTP on %s", svr.Addr)
		if err := svr.ListenAndServe(); err != nil && errors.Is(err, http.ErrServerClosed) {
			log.Info("Server shutdown completed")
		}
	}()
}

// Shutdown shutdwon http server
func Shutdown() {
	if closer != nil {
		closer.Close()
	}
	if svr == nil {
		return
	}
	if err := svr.Shutdown(context.Background()); err != nil {
		log.Errorf("Server shutdown error:%v", err)
	}
}

// InitTracer init server trace
func InitTracer(c *trace.Config) {
	_, closer = trace.NewTracer(c)
}
