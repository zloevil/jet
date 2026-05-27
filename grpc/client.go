package grpc

import (
	"context"
	"github.com/zloevil/jet"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"net"
)

// ClientAuthConfig client auth configuration
type ClientAuthConfig struct {
	Enabled     bool   `mapstructure:"enabled"`      // Enabled if true auth header is passed with each call
	TokenSecret string `mapstructure:"token_secret"` // TokenSecret secret to generate token
	TokenTTL    int    `mapstructure:"token_ttl"`    // TokenTTL token time to live
	Caller      string `mapstructure:"caller"`       // Caller name
}

// ClientConfig is gRPC client configuration
type ClientConfig struct {
	Host string
	Port string
	Auth ClientAuthConfig
}

// Client grpc client
type Client struct {
	readinessAwaiter
	Conn *grpc.ClientConn
}

func NewClient(cfg *ClientConfig) (*Client, error) {

	c := &Client{}

	gc, err := grpc.Dial(net.JoinHostPort(cfg.Host, cfg.Port),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithChainUnaryInterceptor(c.unaryClientInterceptor(cfg)),
		grpc.WithChainStreamInterceptor(c.streamClientInterceptor(cfg)))

	if err != nil {
		return nil, ErrGrpcClientDial(err)
	}

	c.Conn = gc
	c.readinessAwaiter = newReadinessAwaiter(gc)

	return c, nil
}

// this middleware is applied on client side
// it retrieves session params from the context (normally it's populated in HTTP middleware or by another caller) and puts it to gRPS metadata
func (c *Client) unaryClientInterceptor(cfg *ClientConfig) grpc.UnaryClientInterceptor {
	return func(parentCtx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		ctx := context.Background()

		// populate context
		md, ok := jet.ContextToGrpcMD(parentCtx)

		// setup auth header if authorization enabled
		if cfg.Auth.Enabled {
			token, err := buildAccessToken(ctx, &internalAuthConfig{
				InternalAccessTokenSecret: cfg.Auth.TokenSecret,
				InternalAccessTokenTTL:    cfg.Auth.TokenTTL,
			}, cfg.Auth.Caller)
			if err != nil {
				return ToAppError(ctx, method, err)
			}
			md.Append(AuthHeader, token)
		}

		if ok {
			ctx = metadata.NewOutgoingContext(ctx, md)
		}

		// invoke
		if err := invoker(ctx, method, req, reply, cc, opts...); err != nil {
			return ToAppError(ctx, method, err)
		}
		return nil
	}
}

func (c *Client) streamClientInterceptor(cfg *ClientConfig) grpc.StreamClientInterceptor {
	return func(parentCtx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		ctx := context.Background()

		// populate context
		md, ok := jet.ContextToGrpcMD(parentCtx)

		// setup auth header if authorization enabled
		if cfg.Auth.Enabled {
			token, err := buildAccessToken(ctx, &internalAuthConfig{
				InternalAccessTokenSecret: cfg.Auth.TokenSecret,
				InternalAccessTokenTTL:    cfg.Auth.TokenTTL,
			}, cfg.Auth.Caller)
			if err != nil {
				return nil, ToAppError(ctx, method, err)
			}
			md.Append(AuthHeader, token)
		}

		if ok {
			ctx = metadata.NewOutgoingContext(ctx, md)
		}

		// build stream
		clStream, err := streamer(ctx, desc, cc, method, opts...)
		if err != nil {
			err = ToAppError(ctx, method, err)
		}
		return clStream, err
	}
}
