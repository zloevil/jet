package kafka

import (
	"context"
	"encoding/json"
	"github.com/zloevil/jet"
)

type Message struct {
	// Ctx allows passing context from producer to subscriber
	Ctx *jet.RequestContext `json:"ctx"`
	// Key is a kafka message key which allow to specify target kafka partition
	Key string `json:"key"`
	// Payload arbitrary data
	Payload any `json:"payload"`
}

type MessageT[T any] struct {
	// Ctx allows passing context from producer to subscriber
	Ctx *jet.RequestContext `json:"ctx"`
	// Key is a kafka message key which allow to specify target kafka partition
	Key string `json:"key"`
	// Payload arbitrary data
	Payload T `json:"payload"`
}

func Decode[T any](parentCtx context.Context, msg []byte) (T, context.Context, error) {
	if parentCtx == nil {
		parentCtx = context.Background()
	}
	var m MessageT[T]
	err := json.Unmarshal(msg, &m)
	if err != nil {
		var v T
		return v, nil, ErrKafkaDecodeMsgUnmarshal(parentCtx, err)
	}
	ctx := m.Ctx.ToContext(parentCtx)
	return m.Payload, ctx, nil
}
