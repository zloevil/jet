package google

import (
	"context"
	"encoding/json"
	"github.com/zloevil/jet"
	"net/http"
	"net/url"
	"time"
)

const (
	verifyApi = "https://www.google.com/recaptcha/api/siteverify"

	v2 = "v2"
	v3 = "v3"

	ErrCodeCaptchaNotSupportedVer = "CPT-001"
	ErrCodeCaptchaRequest         = "CPT-002"
	ErrCodeCaptchaResponseFormat  = "CPT-003"
	ErrCodeCaptchaResponseStatus  = "CPT-004"
)

var (
	ErrCaptchaNotSupportedVer = func(ctx context.Context) error {
		return jet.NewAppErrBuilder(ErrCodeCaptchaNotSupportedVer, "not supported version").C(ctx).Business().Err()
	}
	ErrCaptchaRequest = func(ctx context.Context, err error) error {
		return jet.NewAppErrBuilder(ErrCodeCaptchaRequest, "request error").Wrap(err).C(ctx).Err()
	}
	ErrCaptchaResponseFormat = func(ctx context.Context, err error) error {
		return jet.NewAppErrBuilder(ErrCodeCaptchaResponseFormat, "response format").Wrap(err).C(ctx).Err()
	}
	ErrCaptchaResponseStatus = func(ctx context.Context, status string) error {
		return jet.NewAppErrBuilder(ErrCodeCaptchaResponseStatus, "response status: %s", status).C(ctx).Err()
	}
)

type CaptchaRequest struct {
	Captcha  string // Captcha - token from client to be checked
	ClientIP string // ClientIP - client real ip
	Version  string // Version - captcha version
}

type Captcha interface {
	// Verify captcha
	Verify(ctx context.Context, rq *CaptchaRequest) (bool, error)
}

type captcha struct {
	cfg      *Config
	client   *http.Client
	logger   jet.CLogger
	versions map[string]string
}

type resultVerify struct {
	Success     bool      `json:"success"`
	Score       float64   `json:"score"`
	Action      string    `json:"action"`
	ChallengeTs time.Time `json:"challenge-ts"`
	Hostname    string    `json:"hostname"`
	ErrorCodes  []string  `json:"error-codes"`
}

func NewCaptcha(cfg *Config, logger jet.CLogger) Captcha {
	return &captcha{
		cfg:    cfg,
		logger: logger,
		client: &http.Client{Timeout: time.Duration(cfg.ClientTimeout)},
		versions: map[string]string{
			v2: cfg.ReCaptchaSecretV2,
			v3: cfg.ReCaptchaSecretV3,
		},
	}
}

func (c *captcha) l() jet.CLogger {
	return c.logger.Cmp("captcha")
}

func (c *captcha) Verify(ctx context.Context, rq *CaptchaRequest) (bool, error) {
	l := c.l().C(ctx).Mth("verify").Dbg()
	// validate input
	if err := jet.NewValidator(ctx).Mth("captcha.verify").
		NotEmptyString("captcha", rq.Captcha).
		NotEmptyString("ip", rq.ClientIP).
		NotEmptyString("version", rq.Version).E(); err != nil {
		return false, err
	}

	// secret only equal v2 or v3 constants
	secret, ok := c.versions[rq.Version]
	if !ok {
		return false, ErrCaptchaNotSupportedVer(ctx)
	}

	data := url.Values{
		"secret":   {secret},
		"response": {rq.Captcha},
		"remoteip": {rq.ClientIP},
	}

	resp, err := c.client.PostForm(verifyApi, data)
	if err != nil {
		return false, ErrCaptchaRequest(ctx, err)
	}

	resp.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if resp.StatusCode != http.StatusOK {
		return false, ErrCaptchaResponseStatus(ctx, resp.Status)
	}

	defer func() { _ = resp.Body.Close() }()

	var r resultVerify
	err = json.NewDecoder(resp.Body).Decode(&r)
	if err != nil {
		return false, ErrCaptchaResponseFormat(ctx, err)
	}

	if !r.Success || r.Score < c.cfg.ReCaptchaScore {
		return false, nil
	}

	l.Dbg("ok")

	return true, nil
}
