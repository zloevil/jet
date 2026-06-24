package kafka

import (
	"github.com/stretchr/testify/suite"
	"github.com/zloevil/jet"
	"testing"
)

type subscriberTestSuite struct {
	jet.Suite
	logger jet.CLoggerFunc
}

func (s *subscriberTestSuite) SetupSuite() {
	s.logger = func() jet.CLogger { return jet.L(jet.InitLogger(&jet.LogConfig{Level: jet.TraceLevel})) }
	s.Suite.Init(s.logger)
}

func TestSubscriberSuite(t *testing.T) {
	suite.Run(t, new(subscriberTestSuite))
}

func (s *subscriberTestSuite) Test_IndexByKey() {

	test := func(workers int, keys []string, exp ...int) {
		sub := &subscriber{workers: workers}
		for i, k := range keys {
			s.Equal(exp[i], sub.chanIndexByKey([]byte(k)))
		}
	}

	test(1, []string{"1", "2", "33244", jet.NewRandString(), "AAaaFFFff"}, 0, 0, 0, 0, 0)
	test(2, []string{"1", "1", "2", "2"}, 0, 0, 1, 1)
	test(2, []string{"aaFFaaFF", "bbCCbbCD", "aaFFaaFF", "bbCCbbCD"}, 1, 0, 1, 0)

	randKey := jet.NewRandString()
	sub := &subscriber{workers: 10}
	s.Equal(sub.chanIndexByKey([]byte(randKey)), sub.chanIndexByKey([]byte(randKey)))

}

func (s *subscriberTestSuite) Test_DeliveryGuarantee_DefaultsToAtMostOnce() {
	cfg := NewSubscriberCfgBuilder().GroupId("g").Build()
	s.Equal(AtMostOnce, cfg.DeliveryGuarantee)

	sub := newSubscriber(s.logger, &TopicConfig{Topic: "t"}, cfg, []string{"localhost:9092"}, nil,
		func([]byte) error { return nil })
	s.False(sub.atLeastOnce)
}

func (s *subscriberTestSuite) Test_DeliveryGuarantee_AtLeastOnce() {
	cfg := NewSubscriberCfgBuilder().GroupId("g").DeliveryGuarantee(AtLeastOnce).Build()
	s.Equal(AtLeastOnce, cfg.DeliveryGuarantee)

	sub := newSubscriber(s.logger, &TopicConfig{Topic: "t"}, cfg, []string{"localhost:9092"}, nil,
		func([]byte) error { return nil })
	s.True(sub.atLeastOnce)
}
