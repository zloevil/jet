package http

import (
	"context"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/rs/cors"
	"github.com/zloevil/jet"
	"github.com/zloevil/jet/goroutine"
	"net/http"
	"time"
)

type Cors struct {
	Enabled        bool
	AllowedHeaders []string
	AllowedOrigins []string
	AllowedMethods []string
	Debug          bool
}

type TraceDetails struct {
	RequestBody  bool `mapstructure:"request_body"` // RequestBody if true, to trace full request with body. Otherwise, body isn't traced
	Response     bool // Response if true, to trace response (can have significant impact on logging volume and performance)
	ResponseBody bool `mapstructure:"response_body"` // ResponseBody if true, to trace full response with body. Otherwise, body isn't traced
}

type Config struct {
	Port                 string
	Cors                 *Cors
	Trace                bool
	TraceDetails         *TraceDetails `mapstructure:"trace_details"`
	WriteTimeoutSec      int           `mapstructure:"write_timeout_sec"`
	ReadTimeoutSec       int           `mapstructure:"read_timeout_sec"`
	ReadBufferSizeBytes  int           `mapstructure:"read_buffer_size_bytes"`
	WriteBufferSizeBytes int           `mapstructure:"write_buffer_size_bytes"`
}

// Server represents HTTP server
type Server struct {
	Srv        *http.Server        // Srv - internal server
	RootRouter *mux.Router         // RootRouter - root router
	WsUpgrader *websocket.Upgrader // WsUpgrader - websocket upgrader
	logger     jet.CLoggerFunc     // logger
}

type RouteSetter interface {
	Set() error
}

type WsUpgrader interface {
	Set(router *mux.Router, upgrader *websocket.Upgrader)
}

// getOptions getting cors options preconfigured
func getOptions(cfg *Config) cors.Options {
	return cors.Options{
		AllowedOrigins:   cfg.Cors.AllowedOrigins,
		AllowedMethods:   cfg.Cors.AllowedMethods,
		AllowedHeaders:   cfg.Cors.AllowedHeaders,
		AllowCredentials: true,
		Debug:            cfg.Cors.Debug,
	}
}

func NewHttpServer(cfg *Config, logger jet.CLoggerFunc) *Server {

	// define router
	r := mux.NewRouter()
	var baseHandler http.Handler = r

	// CORS config if specified
	if cfg.Cors != nil && cfg.Cors.Enabled {
		baseHandler = cors.New(getOptions(cfg)).Handler(baseHandler)
	}

	// build server
	s := &Server{
		Srv: &http.Server{
			Addr:         fmt.Sprintf(":%s", cfg.Port),
			Handler:      baseHandler,
			WriteTimeout: time.Duration(cfg.WriteTimeoutSec) * time.Second,
			ReadTimeout:  time.Duration(cfg.ReadTimeoutSec) * time.Second,
		},
		WsUpgrader: &websocket.Upgrader{
			ReadBufferSize:  cfg.ReadBufferSizeBytes,
			WriteBufferSize: cfg.WriteBufferSizeBytes,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		logger: logger,
	}

	// tracing
	if cfg.Trace {
		r.Use(s.createTraceMiddleware(cfg.TraceDetails))
	}
	s.RootRouter = r
	return s
}

func (s *Server) SetWsUpgrader(upgradeSetter WsUpgrader) {
	upgradeSetter.Set(s.RootRouter, s.WsUpgrader)
}

func (s *Server) Listen() {
	goroutine.New().
		WithLoggerFn(s.logger).
		WithRetry(goroutine.Unrestricted).
		Cmp("http-server").
		Mth("listen").
		Go(context.Background(),
			func() {
				l := s.logger().Pr("http").Cmp("server").Mth("listen").F(jet.KV{"url": s.Srv.Addr})
				l.Inf("start listening")
			start:
				if err := s.Srv.ListenAndServe(); err != nil {
					if !errors.Is(err, http.ErrServerClosed) {
						l.E(ErrHttpSrvListen(err)).St().Err()
						time.Sleep(time.Second * 5)
						goto start
					} else {
						l.Dbg("server closed")
					}
					return
				}
			})
}

func (s *Server) Close() {
	_ = s.Srv.Close()
}
