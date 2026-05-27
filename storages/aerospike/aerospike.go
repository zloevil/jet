package aerospike

import (
	"context"
	aero "github.com/aerospike/aerospike-client-go/v8"
	aeroLogger "github.com/aerospike/aerospike-client-go/v8/logger"
	kitLog "github.com/zloevil/jet"
)

type Config struct {
	Username string
	Password string
	Host     string
	Port     int
}

// Aerospike serves aerospike instance
type Aerospike interface {
	// Open opens aerospike instance
	Open(ctx context.Context, cfg *Config, logger kitLog.CLoggerFunc) error
	// Close closes instance
	Close(ctx context.Context) error
	// Instance returns instance
	Instance() *aero.Client
	// Reconnect reconnects
	Reconnect(ctx context.Context) error
	// GetKey creates a new key
	GetKey(ctx context.Context, ns string, key interface{}) (*aero.Key, error)
}

func New() Aerospike {
	return &aeroImpl{}
}

type aeroImpl struct {
	logger kitLog.CLoggerFunc
	client *aero.Client
	cfg    *Config
}

func (t *aeroImpl) Reconnect(ctx context.Context) error {
	l := t.l().C(ctx).Mth("reconnect").Dbg()
	if t.client == nil || !t.client.IsConnected() {
		return ErrAeroClosed(ctx)
	}
	err := t.Close(ctx)
	if err != nil {
		return err
	}
	err = t.Open(ctx, t.cfg, t.logger)
	if err != nil {
		return err
	}
	l.Dbg("ok")
	return nil
}

func (t *aeroImpl) l() kitLog.CLogger {
	return t.logger().Cmp("aerospike")
}

func (t *aeroImpl) Instance() *aero.Client {
	return t.client
}

func (t *aeroImpl) Open(ctx context.Context, cfg *Config, logger kitLog.CLoggerFunc) error {
	t.logger = logger
	t.cfg = cfg
	l := t.l().C(ctx).Mth("open").Dbg()
	// open connection
	aeroLogger.Logger.SetLogger(t.l())
	aeroLogger.Logger.SetLevel(aeroLogger.DEBUG)
	clientPolicy := aero.NewClientPolicy()
	client, err := aero.NewClientWithPolicy(clientPolicy, cfg.Host, cfg.Port)
	if err != nil {
		return ErrAeroConn(err, ctx)
	}
	t.client = client
	aeroLogger.Logger.SetLogger(t.l())
	aeroLogger.Logger.SetLevel(aeroLogger.DEBUG)
	l.Inf("opened")
	return nil
}

func (t *aeroImpl) Close(ctx context.Context) error {
	l := t.l().C(ctx).Mth("close").Dbg()
	if t.client == nil || t.client.IsConnected() {
		return ErrAeroClosed(ctx)
	}
	t.client.Close()
	l.Inf("closed")
	return nil
}

func (t *aeroImpl) GetKey(ctx context.Context, ns string, key interface{}) (*aero.Key, error) {
	k, err := aero.NewKey(ns, "", key)
	if err != nil {
		return nil, ErrAeroNewKey(err, ctx)
	}
	return k, nil
}
