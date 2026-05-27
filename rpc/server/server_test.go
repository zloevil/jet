package server

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/suite"
	"github.com/zloevil/jet"
	"github.com/zloevil/jet/cluster"
	"github.com/zloevil/jet/kafka"
	"github.com/zloevil/jet/mocks"
	"github.com/zloevil/jet/rpc"
	"testing"
	"time"
)

type rpcServerTestSuite struct {
	jet.Suite
	logger       jet.CLoggerFunc
	callProducer *mocks.KafkaProducer
}

func (s *rpcServerTestSuite) SetupSuite() {
	s.logger = func() jet.CLogger { return jet.L(jet.InitLogger(&jet.LogConfig{Level: jet.TraceLevel})) }
	s.Suite.Init(s.logger)
}

func (s *rpcServerTestSuite) SetupTest() {
	s.callProducer = &mocks.KafkaProducer{}
}

func TestRpcServerSuite(t *testing.T) {
	suite.Run(t, new(rpcServerTestSuite))
}

type Body struct {
	Value string `json:"val"`
}

func (s *rpcServerTestSuite) Test_Call_NoResponseRequired_Ok() {
	rpcServer := NewServer(s.logger, s.callProducer, cluster.NewDistributedKeys(), &rpc.Config{}).(*rpcServer)
	msg := &rpc.Message{
		Type:             rpc.MessageType(1),
		Key:              jet.NewRandString(),
		RequestId:        jet.NewRandString(),
		ResponseRequired: false,
		Body:             &Body{Value: jet.NewRandString()},
	}
	var actualMsg *rpc.Message
	rpcServer.RegisterType(msg.Type, func(ctx context.Context, msg *rpc.Message) error {
		actualMsg = msg
		return nil
	}, func() interface{} { return &Body{} })
	rqCtx, _ := jet.Request(s.Ctx)
	kafkaMsg := &kafka.Message{
		Ctx:     rqCtx,
		Key:     msg.Key,
		Payload: msg,
	}
	kafkaMsgBytes, _ := json.Marshal(kafkaMsg)
	s.Nil(rpcServer.RequestHandler(kafkaMsgBytes))
	s.Equal(0, rpcServer.rqPool.Len())
	s.NotEmpty(actualMsg)
	s.Equal(msg.Key, actualMsg.Key)
	s.Equal(msg.RequestId, actualMsg.RequestId)
	s.Equal(msg.Body.(*Body).Value, actualMsg.Body.(*Body).Value)
}

func (s *rpcServerTestSuite) Test_Call_ResponseRequired_Ok() {
	rpcServer := NewServer(s.logger, s.callProducer, cluster.NewDistributedKeys(), &rpc.Config{}).(*rpcServer)
	msg := &rpc.Message{
		Type:             rpc.MessageType(1),
		Key:              jet.NewRandString(),
		RequestId:        jet.NewRandString(),
		ResponseRequired: true,
		Body:             &Body{Value: jet.NewRandString()},
	}
	s.callProducer.On("Send", s.Ctx, msg.Key, msg).Return(nil)
	var actualMsg *rpc.Message
	rpcServer.RegisterType(msg.Type, func(ctx context.Context, msg *rpc.Message) error {
		actualMsg = msg
		s.Nil(rpcServer.Response(s.Ctx, msg))
		return nil
	}, func() interface{} { return &Body{} })
	rqCtx, _ := jet.Request(s.Ctx)
	kafkaMsg := &kafka.Message{
		Ctx:     rqCtx,
		Key:     msg.Key,
		Payload: msg,
	}
	kafkaMsgBytes, _ := json.Marshal(kafkaMsg)
	s.Nil(rpcServer.RequestHandler(kafkaMsgBytes))
	s.Equal(0, rpcServer.rqPool.Len())
	s.NotEmpty(actualMsg)
	s.AssertCalled(&s.callProducer.Mock, "Send", s.Ctx, msg.Key, msg)
}

func (s *rpcServerTestSuite) Test_Call_WhenRequestTimeout_Ok() {
	rpcServer := NewServer(s.logger, s.callProducer, cluster.NewDistributedKeys(),
		&rpc.Config{CallTimeOut: time.Second}).(*rpcServer)
	msg := &rpc.Message{
		Type:             rpc.MessageType(1),
		Key:              jet.NewRandString(),
		RequestId:        jet.NewRandString(),
		ResponseRequired: true,
		Body:             &Body{Value: jet.NewRandString()},
	}
	var actualExpiredMsg *rpc.Message
	rpcServer.SetExpirationCallback(func(ctx context.Context, msg *rpc.Message) error {
		actualExpiredMsg = msg
		return nil
	})
	rpcServer.RegisterType(msg.Type, func(ctx context.Context, msg *rpc.Message) error {
		return nil
	}, func() interface{} { return &Body{} })
	rpcServer.Start(s.Ctx)
	defer rpcServer.Close(s.Ctx)
	rqCtx, _ := jet.Request(s.Ctx)
	kafkaMsg := &kafka.Message{
		Ctx:     rqCtx,
		Key:     msg.Key,
		Payload: msg,
	}
	kafkaMsgBytes, _ := json.Marshal(kafkaMsg)
	s.Nil(rpcServer.RequestHandler(kafkaMsgBytes))
	if err := <-jet.Await(func() (bool, error) {
		return actualExpiredMsg != nil, nil
	}, time.Millisecond*500, time.Second*3); err != nil {
		s.Fatal(err)
	}
	s.Equal(0, rpcServer.rqPool.Len())
	s.Equal(msg.RequestId, actualExpiredMsg.RequestId)
}
