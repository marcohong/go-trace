package cache

import (
	"time"

	"github.com/go-redis/redis/v8"
)

// Config redis config
type Config struct {
	Username     string
	Password     string
	Addr         string   // host:port address.
	Addrs        []string // cluster, sentinel, ["host1:port1","host2:port2"]
	MasterName   string   // needs in sentinel model
	Proto        string   // tcp/unix, default tcp, Single/sentinel mode available
	DB           int      // Single/sentinel mode available
	PoolSize     int
	MaxRetries   int // default 3
	MinIdleConns int
	PoolTimeout  time.Duration
	IdleTimeout  time.Duration
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// New return redis connect.
func New(c *Config) *redis.Client {
	conn := redis.NewClient(&redis.Options{
		Network:      c.Proto,
		Addr:         c.Addr,
		Username:     c.Username,
		Password:     c.Password,
		DB:           c.DB,
		IdleTimeout:  time.Duration(c.IdleTimeout),
		DialTimeout:  time.Duration((c.DialTimeout)),
		ReadTimeout:  time.Duration(c.ReadTimeout),
		WriteTimeout: time.Duration(c.WriteTimeout),
		PoolSize:     c.PoolSize,
		PoolTimeout:  time.Duration(c.PoolTimeout),
		MaxRetries:   c.MaxRetries,
		MinIdleConns: c.MinIdleConns,
	})
	conn.AddHook(NewTracingHook())
	return conn
}
