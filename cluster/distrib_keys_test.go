package cluster

import (
	"github.com/stretchr/testify/suite"
	"github.com/zloevil/jet"
	"testing"
)

type distributedKeysTestSuite struct {
	jet.Suite
	logger jet.CLoggerFunc
	svc    DistributedKeys
}

func (s *distributedKeysTestSuite) SetupSuite() {
	s.logger = func() jet.CLogger { return jet.L(jet.InitLogger(&jet.LogConfig{Level: jet.TraceLevel})) }
	s.Suite.Init(s.logger)
	s.svc = NewDistributedKeys()
}

func TestDistributedKeysSuite(t *testing.T) {
	suite.Run(t, new(distributedKeysTestSuite))
}

func (s *distributedKeysTestSuite) Test_CheckWhenEmpty() {
	s.False(s.svc.Check(jet.NewRandString()))
}

func (s *distributedKeysTestSuite) Test_RemoveWhenEmpty() {
	s.svc.Remove(jet.NewRandString())
}

func (s *distributedKeysTestSuite) Test_SetRemoveCheck() {
	key := jet.NewRandString()
	s.svc.Set(key)
	s.True(s.svc.Check(key))
	s.svc.Remove(key)
	s.False(s.svc.Check(key))
}
