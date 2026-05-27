package centrifugo

import (
	"context"
	"github.com/centrifugal/centrifuge-go"
	"github.com/zloevil/jet"
	"go.uber.org/atomic"
)

type Client interface {
	// Connect connects client
	Connect(ctx context.Context, token string) error
	// Close closes connection
	Close(ctx context.Context) error
	// Subscribe subscribes on channel. Does not require in case of server-side subscription
	Subscribe(ctx context.Context, token, channel string, handler func(p centrifuge.Publication) error) error
	// OnPublication allows handling publication in case of server-side subscription
	OnPublication(ctx context.Context, handler func(p centrifuge.ServerPublicationEvent) error) error
	// Connected returns true if client is connected
	Connected() bool
	// ClientId returns client id of a connected client
	ClientId() string
}

type ClientConfig struct {
	Url string
}

func NewClient(cfg *ClientConfig, logger jet.CLoggerFunc) Client {
	return &clientImpl{
		logger:    logger,
		cfg:       cfg,
		connected: atomic.NewBool(false),
		clientId:  atomic.NewString(""),
	}
}

type clientImpl struct {
	cfg       *ClientConfig
	logger    jet.CLoggerFunc
	client    *centrifuge.Client
	connected *atomic.Bool
	clientId  *atomic.String
}

func (s *clientImpl) l() jet.CLogger {
	return s.logger().Cmp("centrifugo-client")
}

func (s *clientImpl) Connect(ctx context.Context, token string) error {
	l := s.l().C(ctx).Mth("connect")

	// create a ws connection
	s.client = centrifuge.NewJsonClient(s.cfg.Url, centrifuge.Config{
		Token: token,
	})

	s.client.OnConnected(func(event centrifuge.ConnectedEvent) {
		s.connected.Store(true)
		s.clientId.Store(event.ClientID)
		l.DbgF("Connected: %s", event.ClientID)
	})
	s.client.OnConnecting(func(event centrifuge.ConnectingEvent) {
		l.DbgF("connecting: %d (%s)", event.Code, event.Reason)
	})
	s.client.OnDisconnected(func(event centrifuge.DisconnectedEvent) {
		s.connected.Store(false)
		s.clientId.Store("")
		l.DbgF("disconnected: %d (%s)", event.Code, event.Reason)
	})
	s.client.OnError(func(event centrifuge.ErrorEvent) {
		l.E(ErrCentrifugeInternal(ctx, event.Error)).Err()
	})

	err := s.client.Connect()
	if err != nil {
		return ErrCentrifugoConnect(ctx, err)
	}

	l.Dbg("ok")

	return nil
}

func (s *clientImpl) Close(ctx context.Context) error {
	l := s.l().C(ctx).Mth("close")
	if s.client != nil {
		_ = s.client.Disconnect()
		s.client.Close()
	}
	l.Dbg("ok")
	return nil
}

func (s *clientImpl) Connected() bool {
	return s.connected.Load()
}

func (s *clientImpl) ClientId() string {
	return s.clientId.Load()
}

func (s *clientImpl) Publish(ctx context.Context, channel string, payload any) error {

	bytes, _ := jet.JsonEncode(payload)

	_, err := s.client.Publish(ctx, channel, bytes)
	if err != nil {
		return ErrCentrifugoGrpcPublish(ctx, err)
	}

	return nil
}

func (s *clientImpl) Subscribe(ctx context.Context, token, channel string, handler func(p centrifuge.Publication) error) error {

	sub, err := s.client.NewSubscription(channel, centrifuge.SubscriptionConfig{
		Token: token,
	})
	if err != nil {
		return ErrCentrifugoSubscribing(ctx, err)
	}

	err = sub.Subscribe()
	if err != nil {
		return ErrCentrifugoSubscribe(ctx, err)
	}

	sub.OnPublication(func(event centrifuge.PublicationEvent) {
		if err := handler(event.Publication); err != nil {
			s.l().C(ctx).Mth("on-event").E(err).St().Err()
		}
	})

	return nil
}

func (s *clientImpl) OnPublication(ctx context.Context, handler func(p centrifuge.ServerPublicationEvent) error) error {

	s.client.OnPublication(func(event centrifuge.ServerPublicationEvent) {
		if err := handler(event); err != nil {
			s.l().C(ctx).Mth("on-event").E(err).St().Err()
		}
	})
	return nil

}
