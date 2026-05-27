package redis

import (
	"context"
	"github.com/redis/go-redis/v9"
	"github.com/zloevil/jet"
	"net"
	"time"
)

const (
	NotFound = redis.Nil
)

type Redis struct {
	Instance *redis.Client
	Ttl      time.Duration
	logger   jet.CLoggerFunc
}

// Config redis config
type Config struct {
	Host     string
	Port     string
	Username string
	Password string
	Db       int
	Ttl      uint
}

func (r *Redis) l() jet.CLogger {
	return r.logger().Cmp("redis")
}

func Open(ctx context.Context, params *Config, logger jet.CLoggerFunc) (*Redis, error) {

	l := logger().Cmp("redis").Mth("open")

	client := redis.NewClient(&redis.Options{
		Addr:     net.JoinHostPort(params.Host, params.Port),
		Username: params.Username,
		Password: params.Password,
		DB:       params.Db,
	})
	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, ErrRedisPingErr(err)
	}

	l.Inf("ok")
	return &Redis{
		Instance: client,
		Ttl:      time.Duration(params.Ttl) * time.Second,
		logger:   logger,
	}, nil
}

func (r *Redis) Close() {
	l := r.l().Mth("close")
	if r.Instance != nil {
		_ = r.Instance.Close()
	}
	l.Inf("ok")
}
