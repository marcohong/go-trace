package orm

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
		ServiceName:        "Trace-database-service",
		OpenReporter:       true,                           // open jaeger reporter
		Stdlog:             true,                           // log stdout
		ReportHost:         "127.0.0.1:6831",               // host:port -> 127.0.0.1:6831
		SamplerType:        "const",                        //const, probabilistic, rateLimiting, or remote
		SamplerParam:       1,                              // 0 or 1
		FlushInterval:      time.Duration(1 * time.Second), // second, default 1
		DisableClientTrace: false,
	}
	_, closer = trace.NewTracer(conf)
	tr = trace.StartSpan("gorm-trace")
}

// User .
type User struct {
	Host string `gorm:"Host"`
	User string `gorm:"User"`
}

func TestMySQL(t *testing.T) {
	ctx := context.WithValue(context.Background(), trace.CtxKey, tr.GetSpan())
	defer closer.Close()
	conf := &Config{
		DSN:    "root:@tcp(127.0.0.1:3306)/mysql?charset=utf8&parseTime=True&loc=Local",
		Idle:   5,
		Active: 30,
	}
	conn := NewMySQL(conf)
	db, _ := conn.DB()
	defer db.Close()
	conn = conn.WithContext(ctx)
	users := []User{}
	conn.Model(&users).Find(&users)
	for _, user := range users {
		t.Logf("user: %+v", user)
	}
}
