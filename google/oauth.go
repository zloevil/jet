package google

import (
	"context"
	"github.com/zloevil/jet"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	"io"
	"os"

	"golang.org/x/oauth2/google"
	oauthv2 "google.golang.org/api/oauth2/v2"
	"net/http"
	"sync"
	"time"
)

const (
	ErrCodeOAuthConfigRead     = "OAUTH-001"
	ErrCodeOAuthService        = "OAUTH-002"
	ErrCodeOAuthGetUser        = "OAUTH-003"
	ErrCodeOAuthConfigFileRead = "OAUTH-004"
)

var (
	ErrOAuthConfigRead = func(ctx context.Context, cause error) error {
		return jet.NewAppErrBuilder(ErrCodeOAuthConfigRead, "reading config").Wrap(cause).C(ctx).Business().Err()
	}
	ErrOAuthService = func(ctx context.Context, cause error) error {
		return jet.NewAppErrBuilder(ErrCodeOAuthService, "service").Wrap(cause).C(ctx).Business().Err()
	}
	ErrOAuthGetUser = func(ctx context.Context, cause error) error {
		return jet.NewAppErrBuilder(ErrCodeOAuthGetUser, "get user").Wrap(cause).C(ctx).Business().Err()
	}
	ErrOAuthConfigFileRead = func(ctx context.Context, cause error) error {
		return jet.NewAppErrBuilder(ErrCodeOAuthConfigFileRead, "reading config").Wrap(cause).C(ctx).Business().Err()
	}
)

type OAuth2 interface {
	// GetGoogleUser retrieves google user info
	GetGoogleUser(ctx context.Context, token string) (*oauthv2.Userinfo, error)
}

type oauth struct {
	cfg       *Config
	client    *http.Client
	logger    jet.CLogger
	lazy      sync.Once
	googleCfg *oauth2.Config
}

func NewOAuth(cfg *Config, logger jet.CLogger) (OAuth2, error) {
	if cfg.ConfigurationPath != "" {
		// if configuration path is set, then read configuration from the file
		file, err := os.Open(cfg.ConfigurationPath)
		if err != nil {
			return nil, ErrOAuthConfigFileRead(context.Background(), err)
		}

		fileContent, err := io.ReadAll(file)
		if err != nil {
			return nil, ErrOAuthConfigFileRead(context.Background(), err)
		}

		cfg.JsonConfiguration = string(fileContent)
	}
	return &oauth{
		cfg:    cfg,
		logger: logger,
		client: &http.Client{Timeout: time.Duration(cfg.ClientTimeout)},
	}, nil
}

func (o *oauth) l() jet.CLogger {
	return o.logger.Cmp("oauth")
}

func (o *oauth) GetGoogleUser(ctx context.Context, token string) (*oauthv2.Userinfo, error) {
	o.l().Mth("get-user").C(ctx).Dbg()

	// load oauth config
	var err error
	o.lazy.Do(func() {
		o.googleCfg, err = google.ConfigFromJSON([]byte(o.cfg.JsonConfiguration))
	})
	if err != nil {
		return nil, ErrOAuthConfigRead(ctx, err)
	}

	// client with token
	at := &oauth2.Token{
		AccessToken: token,
		TokenType:   "Bearer",
	}

	httpClient := o.googleCfg.Client(ctx, at)
	httpClient.Timeout = time.Duration(o.cfg.ClientTimeout)

	// prepare google service
	gService, err := oauthv2.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, ErrOAuthService(ctx, err)
	}

	// execute
	ui, err := gService.Userinfo.V2.Me.Get().Do()
	if err != nil {
		return nil, ErrOAuthGetUser(ctx, err)
	}

	return ui, nil
}
