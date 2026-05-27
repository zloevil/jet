package kafka

import (
	"encoding/json"
	"github.com/stretchr/testify/suite"
	"github.com/zloevil/jet"
	"testing"
)

type decodeTestSuite struct {
	jet.Suite
	logger jet.CLoggerFunc
}

func (s *decodeTestSuite) SetupSuite() {
	s.logger = func() jet.CLogger { return jet.L(jet.InitLogger(&jet.LogConfig{Level: jet.TraceLevel})) }
	s.Suite.Init(s.logger)
}

func TestDecodeSuite(t *testing.T) {
	suite.Run(t, new(decodeTestSuite))
}

func (s *decodeTestSuite) Test_WhenRawPayload() {
	rId := jet.NewId()
	rqCtx := jet.NewRequestCtx().WithRequestId(rId)
	rawPl := map[string]any{
		"key1": "val1",
		"key2": "val2",
	}
	msg := &MessageT[map[string]any]{
		Ctx:     rqCtx,
		Key:     jet.NewRandString(),
		Payload: rawPl,
	}
	msgBytes, _ := json.Marshal(msg)
	decoded, ctx, err := Decode[map[string]any](s.Ctx, msgBytes)
	s.Nil(err)
	s.NotEmpty(ctx)
	actRq, _ := jet.Request(ctx)
	s.NotEmpty(actRq)
	s.Equal(actRq.Rid, rId)
	s.Equal(decoded, msg.Payload)
}

func (s *decodeTestSuite) Test_WhenStructUnmarshalledToRawPayload() {
	rId := jet.NewId()
	rqCtx := jet.NewRequestCtx().WithRequestId(rId)
	pl := struct {
		Key string `json:"key"`
	}{
		Key: "123",
	}
	msg := &Message{
		Ctx:     rqCtx,
		Key:     jet.NewRandString(),
		Payload: pl,
	}
	msgBytes, _ := json.Marshal(msg)
	decoded, ctx, err := Decode[map[string]any](s.Ctx, msgBytes)
	s.Nil(err)
	s.NotEmpty(ctx)
	actRq, _ := jet.Request(ctx)
	s.NotEmpty(actRq)
	s.Equal(actRq.Rid, rId)
	s.Equal(decoded["key"].(string), pl.Key)
}

func (s *decodeTestSuite) Test_WhenStructUnmarshalledToStruct() {
	rqCtx, _ := jet.Request(s.Ctx)
	type plt struct {
		Key string `json:"key"`
	}
	pl := &plt{
		Key: "123",
	}
	msg := &Message{
		Ctx:     rqCtx,
		Key:     jet.NewRandString(),
		Payload: pl,
	}
	msgBytes, _ := json.Marshal(msg)
	decoded, ctx, err := Decode[plt](s.Ctx, msgBytes)
	s.Nil(err)
	s.NotEmpty(ctx)
	s.Equal(decoded.Key, pl.Key)
}
