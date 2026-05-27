package jet

import (
	"context"
	"github.com/golang-jwt/jwt/v5"
	"time"
)

const (
	ClaimCallerName = "caller_name"

	ErrCodeJwtParse              = "JWT-001"
	ErrCodeJwtMalformed          = "JWT-002"
	ErrCodeJwtTokenGen           = "JWT-003"
	ErrCodeJwtWrongSigningMethod = "JWT-004"
)

var (
	ErrJwtParse = func(cause error) error {
		return NewAppErrBuilder(ErrCodeJwtParse, "").Wrap(cause).Err()
	}
	ErrJwtMalformed = func() error {
		return NewAppErrBuilder(ErrCodeJwtMalformed, "").Err()
	}
	ErrJwtTokenGen = func(ctx context.Context, cause error) error {
		return NewAppErrBuilder(ErrCodeJwtTokenGen, "jwt gen").Wrap(cause).Err()
	}
	ErrJwtWrongSigningMethod = func(ctx context.Context) error {
		return NewAppErrBuilder(ErrCodeJwtWrongSigningMethod, "wrong signing method").Err()
	}
)

// GenerateInternalAccessToken generates token for internal communications between services
func GenerateInternalAccessToken(ctx context.Context, secret []byte, ttl int, callerName string) (string, error) {
	return GenJwtToken(ctx, &JwtRequest{
		Secret:   secret,
		ExpireAt: time.Now().Add(time.Duration(ttl)),
		Claims: map[string]any{
			ClaimCallerName: callerName,
			"expired_at":    time.Now().Add(time.Duration(ttl)).Unix(),
			"created_at":    time.Now().Unix(),
		},
	})
}

// ParseInternalAccessToken parses an internal token
func ParseInternalAccessToken(ctx context.Context, secret []byte, token string) (string, error) {
	_, claims, err := VerifyJwtToken(ctx, token, secret)
	if err != nil {
		return "", err
	}
	return claims[ClaimCallerName].(string), nil
}

// JwtRequest request to generate a new jwt token
type JwtRequest struct {
	UserId   string
	Secret   []byte
	ExpireAt time.Time
	Claims   map[string]any
}

// GenJwtToken generates a new JWT token
func GenJwtToken(ctx context.Context, rq *JwtRequest) (string, error) {

	claims := jwt.MapClaims{}
	claims["exp"] = rq.ExpireAt.Unix()
	claims["tid"] = NewId()
	claims["cr"] = Now().Unix()
	if rq.UserId != "" {
		claims["sub"] = rq.UserId
	}

	if rq.Claims != nil {
		for k, v := range rq.Claims {
			claims[k] = v
		}
	}
	at := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	atStr, err := at.SignedString(rq.Secret)
	if err != nil {
		return "", ErrJwtTokenGen(ctx, err)
	}

	return atStr, nil
}

// VerifyJwtToken verifies a JWT token
func VerifyJwtToken(ctx context.Context, token string, secret []byte) (*jwt.Token, jwt.MapClaims, error) {
	tkn, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrJwtWrongSigningMethod(ctx)
		}
		return secret, nil
	})
	if err != nil {
		return nil, nil, ErrJwtParse(err)
	}
	claims, ok := tkn.Claims.(jwt.MapClaims)
	if ok && tkn.Valid {
		return tkn, claims, nil
	}
	return nil, nil, ErrJwtMalformed()
}
