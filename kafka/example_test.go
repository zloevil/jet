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
