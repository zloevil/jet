package jet

import (
	"github.com/stretchr/testify/suite"
	"testing"
	"time"
)

type jwtTestSuite struct {
	Suite
}

func (s *jwtTestSuite) SetupSuite() {
	s.Suite.Init(nil)
}

func TestJwtSuite(t *testing.T) {
	suite.Run(t, new(jwtTestSuite))
}

func (s *jwtTestSuite) Test_WhenValidWithClaims() {

	secret := []byte("123")
	tkn, err := GenJwtToken(s.Ctx, &JwtRequest{
		UserId:   NewId(),
		Secret:   secret,
		ExpireAt: Now().Add(time.Minute),
		Claims: map[string]any{
			"cl1": "val1",
		},
	})
	s.NoError(err)
	s.NotEmpty(tkn)

	tkn2, claims, err := VerifyJwtToken(s.Ctx, tkn, secret)
	s.NoError(err)
	s.NotEmpty(tkn2)
	s.NotEmpty(claims)
	s.NotEmpty(claims["cl1"])
}

func (s *jwtTestSuite) Test_WhenExpired() {

	secret := []byte("123")
	tkn, err := GenJwtToken(s.Ctx, &JwtRequest{
		UserId:   NewId(),
		Secret:   secret,
		ExpireAt: Now().Add(-time.Minute),
	})
	s.NoError(err)
	s.NotEmpty(tkn)

	_, _, err = VerifyJwtToken(s.Ctx, tkn, secret)
	s.Error(err)
}

func (s *jwtTestSuite) Test_InternalToken() {

	secret := []byte("123")
	tkn, err := GenerateInternalAccessToken(s.Ctx, secret, 99999999999, "name")
	s.NoError(err)
	s.NotEmpty(tkn)

	caller, err := ParseInternalAccessToken(s.Ctx, secret, tkn)
	s.NoError(err)
	s.Equal("name", caller)
}

func (s *jwtTestSuite) Test_InternalToken_WhenZeroTtl() {

	secret := []byte("123")
	tkn, err := GenerateInternalAccessToken(s.Ctx, secret, 0, "name")
	s.NoError(err)
	s.NotEmpty(tkn)

	_, err = ParseInternalAccessToken(s.Ctx, secret, tkn)
	s.Error(err)
}
