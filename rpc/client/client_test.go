package client

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

type rpcClientTestSuite struct {
	jet.Suite
	logger       jet.CLoggerFunc
	callProducer *mocks.KafkaProducer
}

func (s *rpcClientTestSuite) SetupSuite() {
	s.logger = func() jet.CLogger { return jet.L(jet.InitLogger(&jet.LogConfig{Level: jet.TraceLevel})) }
	s.Suite.Init(s.logger)
}

func (s *rpcClientTestSuite) SetupTest() {
	s.callProducer = &mocks.KafkaProducer{}
}

func TestRpcClientSuite(t *testing.T) {
	suite.Run(t, new(rpcClientTestSuite))
}

type Body struct {
	Value string `json:"val"`
}

func (s *rpcClientTestSuite) Test_Call_NoResponseRequired_Ok() {
	rpcCl := NewClient(s.logger, s.callProducer, cluster.NewDistributedKeys(), &rpc.Config{}).(*rpcClient)
	msg := &rpc.Message{
		Type:             rpc.MessageType(1),
		Key:              jet.NewRandString(),
		RequestId:        jet.NewRandString(),
		ResponseRequired: false,
		Body:             &Body{Value: jet.NewRandString()},
	}
	s.callProducer.On("Send", s.Ctx, msg.Key, msg).Return(nil)
	err := rpcCl.Call(s.Ctx, msg, nil)
	if err != nil {
		s.Fatal(err)
	}
	s.AssertCalled(&s.callProducer.Mock, "Send", s.Ctx, msg.Key, msg)
	s.Equal(0, rpcCl.rqPool.Len())
}

func (s *rpcClientTestSuite) Test_Call_WithResponse_Ok() {
	rpcCl := NewClient(s.logger, s.callProducer, cluster.NewDistributedKeys(), &rpc.Config{}).(*rpcClient)
	msg := &rpc.Message{
		Type:             rpc.MessageType(1),
		Key:              jet.NewRandString(),
		RequestId:        jet.NewRandString(),
		ResponseRequired: true,
		Body:             &Body{Value: jet.NewRandString()},
	}
	s.callProducer.On("Send", s.Ctx, msg.Key, msg).Return(nil)
	var actualRsMsg *rpc.Message
	rpcCl.RegisterBodyTypeProvider(msg.Type, func() interface{} { return &Body{} })
	err := rpcCl.Call(s.Ctx, msg, func(ctx context.Context, rqMsg, rsMsg *rpc.Message) error {
		actualRsMsg = rsMsg
		return nil
	})
	if err != nil {
		s.Fatal(err)
	}
	s.AssertCalled(&s.callProducer.Mock, "Send", s.Ctx, msg.Key, msg)
	s.Equal(1, rpcCl.rqPool.Len())
	// call handler
	rsMsg := &rpc.Message{
		Key:       msg.Key,
		RequestId: msg.RequestId,
		Type:      rpc.MessageType(1),
		Body:      &Body{Value: jet.NewRandString()},
	}
	kafkaMsg := &kafka.Message{
		Ctx:     nil,
		Key:     rsMsg.Key,
		Payload: rsMsg,
	}
	kafkaMsgBytes, _ := json.Marshal(kafkaMsg)
	s.Nil(rpcCl.ResponseHandler(kafkaMsgBytes))
	s.NotEmpty(actualRsMsg)
	s.Equal(rsMsg.Key, actualRsMsg.Key)
	s.Equal(rsMsg.RequestId, actualRsMsg.RequestId)
	s.Equal(rsMsg.Body.(*Body).Value, actualRsMsg.Body.(*Body).Value)
	// check pool empty
	s.Equal(0, rpcCl.rqPool.Len())
}

func (s *rpcClientTestSuite) Test_Call_WhenRequestExpired_Fail() {
	rpcCl := NewClient(s.logger, s.callProducer, cluster.NewDistributedKeys(), &rpc.Config{}).(*rpcClient)
	msg := &rpc.Message{
		Type:             rpc.MessageType(1),
		Key:              jet.NewRandString(),
		RequestId:        jet.NewRandString(),
		ResponseRequired: true,
		Body:             &Body{Value: jet.NewRandString()},
	}
	s.callProducer.On("Send", s.Ctx, msg.Key, msg).Return(nil)
	rpcCl.RegisterBodyTypeProvider(msg.Type, func() interface{} { return &Body{} })
	err := rpcCl.Call(s.Ctx, msg, func(ctx context.Context, rqMsg, rsMsg *rpc.Message) error {
		return nil
	})
	if err != nil {
		s.Fatal(err)
	}
	s.AssertCalled(&s.callProducer.Mock, "Send", s.Ctx, msg.Key, msg)
	s.Equal(1, rpcCl.rqPool.Len())
	// remove request from pool
	rpcCl.rqPool.Remove(msg.RequestId)
	// call handler
	rsMsg := &rpc.Message{
		Key:       msg.Key,
		RequestId: msg.RequestId,
		Type:      rpc.MessageType(1),
		Body:      &Body{Value: jet.NewRandString()},
	}
	rqCtx, _ := jet.Request(s.Ctx)
	kafkaMsg := &kafka.Message{
		Ctx:     rqCtx,
		Key:     rsMsg.Key,
		Payload: rsMsg,
	}
	kafkaMsgBytes, _ := json.Marshal(kafkaMsg)
	err = rpcCl.ResponseHandler(kafkaMsgBytes)
	s.AssertAppErr(err, rpc.ErrCodeRpcRespNoRequestInPool)
}

func (s *rpcClientTestSuite) Test_Call_WhenRequestTimeout_Ok() {
	rpcCl := NewClient(s.logger, s.callProducer, cluster.NewDistributedKeys(),
		&rpc.Config{CallTimeOut: time.Second}).(*rpcClient)
	msg := &rpc.Message{
		Type:             rpc.MessageType(1),
		Key:              jet.NewRandString(),
		RequestId:        jet.NewRandString(),
		ResponseRequired: true,
		Body:             &Body{Value: jet.NewRandString()},
	}
	var actualExpiredMsg *rpc.Message
	rpcCl.SetExpirationCallback(func(ctx context.Context, msg *rpc.Message) error {
		actualExpiredMsg = msg
		return nil
	})
	s.callProducer.On("Send", s.Ctx, msg.Key, msg).Return(nil)
	rpcCl.RegisterBodyTypeProvider(msg.Type, func() interface{} { return &Body{} })
	rpcCl.Start(s.Ctx)
	defer rpcCl.Close(s.Ctx)
	err := rpcCl.Call(s.Ctx, msg, func(ctx context.Context, rqMsg, rsMsg *rpc.Message) error {
		return nil
	})
	if err != nil {
		s.Fatal(err)
	}
	if err := <-jet.Await(func() (bool, error) {
		return actualExpiredMsg != nil, nil
	}, time.Millisecond*500, time.Second*3); err != nil {
		s.Fatal(err)
	}
	s.Equal(0, rpcCl.rqPool.Len())
	s.Equal(msg.RequestId, actualExpiredMsg.RequestId)
}
