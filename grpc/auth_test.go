package grpc

import (
	"math"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/zloevil/jet"
)

type testSuite struct {
	jet.Suite
}

func (s *testSuite) SetupSuite() {
	s.Suite.Init(func() jet.CLogger { return jet.L(jet.InitLogger(&jet.LogConfig{Level: jet.TraceLevel})) })
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

func (s *testSuite) Test() {
	accessToken, err := jet.GenerateInternalAccessToken(
		s.Ctx,
		[]byte("XvderFOjwY28mJMaQDJi371IUBSaGSKw"),
		math.MaxInt,
		"test",
	)
	s.NoError(err)
	s.NotEmpty(accessToken)
	s.L().DbgF("token: %s", accessToken)
}
