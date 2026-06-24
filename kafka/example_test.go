package kafka_test

import (
	"context"

	"github.com/zloevil/jet"
	"github.com/zloevil/jet/kafka"
)

func ExampleBroker() {
	logFn := func() jet.CLogger { return jet.L(jet.InitLogger(&jet.LogConfig{Level: jet.InfoLevel})) }
	ctx := context.Background()

	broker := kafka.NewBroker(logFn)
	if err := broker.Init(ctx, &kafka.BrokerConfig{Url: "localhost:9092"}); err != nil {
		return
	}

	// subscribe to a topic
	_ = broker.AddSubscriber(ctx,
		kafka.NewTopicCfgBuilder("orders").Build(),
		kafka.NewSubscriberCfgBuilder().GroupId("my-service").Build(),
		func(payload []byte) error {
			// handle the message
			return nil
		},
	)

	_ = broker.Start(ctx)
	defer broker.Close(ctx)
}

func ExampleBroker_atLeastOnce() {
	logFn := func() jet.CLogger { return jet.L(jet.InitLogger(&jet.LogConfig{Level: jet.InfoLevel})) }
	ctx := context.Background()

	broker := kafka.NewBroker(logFn)
	if err := broker.Init(ctx, &kafka.BrokerConfig{Url: "localhost:9092"}); err != nil {
		return
	}

	// subscribe with at-least-once delivery: the offset is committed only after
	// the handler returns, so no message is lost on shutdown or crash. A message
	// may be redelivered, so the handler must be idempotent.
	_ = broker.AddSubscriber(ctx,
		kafka.NewTopicCfgBuilder("orders").Build(),
		kafka.NewSubscriberCfgBuilder().
			GroupId("my-service").
			DeliveryGuarantee(kafka.AtLeastOnce).
			Build(),
		func(payload []byte) error {
			// handle the message; the offset is committed once this returns nil
			// (returning an error retries the same message)
			return nil
		},
	)

	_ = broker.Start(ctx)
	defer broker.Close(ctx)
}
