package event

import (
	"context"
	"github.com/stretchr/testify/suite"
	"github.com/zloevil/jet"
	"testing"
	"time"
)

type busTestSuite struct {
	jet.Suite
	logger jet.CLoggerFunc
}

func (s *busTestSuite) SetupSuite() {
	s.logger = func() jet.CLogger { return jet.L(jet.InitLogger(&jet.LogConfig{Level: jet.TraceLevel})) }
	s.Suite.Init(s.logger)
}

func (s *busTestSuite) SetupTest() {}

func (s *busTestSuite) TearDownSuite() {}

func TestBusSuite(t *testing.T) {
	suite.Run(t, new(busTestSuite))
}

func (s *busTestSuite) TestNew() {
	bus := New(s.logger)
	s.NotNil(bus)
}

func (s *busTestSuite) TestHasCallback() {
	bus := New(s.logger)
	s.NoError(bus.Subscribe("topic", func() {}))
	s.False(bus.HasCallback("topic_topic"))
	s.True(bus.HasCallback("topic"))
}

func (s *busTestSuite) TestSubscribe() {
	bus := New(s.logger)
	s.NoError(bus.Subscribe("topic", func() {}))
	s.Error(bus.Subscribe("topic", "String"))
}

func (s *busTestSuite) TestSubscribeOnce() {
	bus := New(s.logger)
	s.NoError(bus.SubscribeOnce("topic", func() {}))
	s.Error(bus.SubscribeOnce("topic", "String"))
}

func (s *busTestSuite) TestSubscribeOnceAndManySubscribe() {
	bus := New(s.logger)
	event := "topic"
	flag := 0
	fn := func(ctx context.Context) { flag += 1 }
	s.NoError(bus.SubscribeOnce(event, fn))
	s.NoError(bus.Subscribe(event, fn))
	s.NoError(bus.Subscribe(event, fn))
	bus.Publish(s.Ctx, event)
	s.Equal(flag, 3)
}

func (s *busTestSuite) TestUnsubscribe() {
	bus := New(s.logger)
	handler := func(ctx context.Context) {}
	s.NoError(bus.Subscribe("topic", handler))
	s.NoError(bus.Unsubscribe("topic", handler))
	s.Error(bus.Unsubscribe("topic", handler))
}

type handler struct {
	val int
}

func (h *handler) Handle(ctx context.Context) {
	h.val++
}

func (s *busTestSuite) TestUnsubscribeMethod() {
	bus := New(s.logger)
	h := &handler{val: 0}

	s.NoError(bus.Subscribe("topic", h.Handle))
	bus.Publish(s.Ctx, "topic")
	s.NoError(bus.Unsubscribe("topic", h.Handle))
	s.Error(bus.Unsubscribe("topic", h.Handle))
	bus.Publish(s.Ctx, "topic")
	bus.WaitAsync()

	s.Equal(1, h.val)
}

func (s *busTestSuite) TestPublish() {
	bus := New(s.logger)
	s.NoError(bus.Subscribe("topic", func(ctx context.Context, a int, err error) {
		s.Equal(10, a)
		s.NoError(err)
	}))
	bus.Publish(s.Ctx, "topic", 10, nil)
}

func (s *busTestSuite) TestSubscribeOnceAsync() {
	results := make([]int, 0)

	bus := New(s.logger)
	s.NoError(bus.SubscribeOnceAsync("topic", func(ctx context.Context, a int, out *[]int) {
		*out = append(*out, a)
	}))

	bus.Publish(s.Ctx, "topic", 10, &results)
	bus.Publish(s.Ctx, "topic", 10, &results)

	bus.WaitAsync()

	s.Equal(1, len(results))

	s.False(bus.HasCallback("topic"))
}

func (s *busTestSuite) TestSubscribeAsyncTransactional() {
	results := make([]int, 0)

	bus := New(s.logger)
	s.NoError(bus.SubscribeAsync("topic", func(ctx context.Context, a int, out *[]int, dur string) {
		sleep, _ := time.ParseDuration(dur)
		time.Sleep(sleep)
		*out = append(*out, a)
	}, true))

	bus.Publish(s.Ctx, "topic", 1, &results, "1s")
	bus.Publish(s.Ctx, "topic", 2, &results, "0s")

	bus.WaitAsync()

	s.Equal(2, len(results))
	s.True(results[0] == 1 && results[1] == 2)

}

func (s *busTestSuite) TestSubscribeAsync() {
	results := make(chan int)

	bus := New(s.logger)
	s.NoError(bus.SubscribeAsync("topic", func(ctx context.Context, a int, out chan<- int) {
		out <- a
	}, false))

	bus.Publish(s.Ctx, "topic", 1, results)
	bus.Publish(s.Ctx, "topic", 2, results)

	numResults := 0

	go func() {
		for _ = range results {
			numResults++
		}
	}()

	bus.WaitAsync()

	time.Sleep(10 * time.Millisecond)

	s.Equal(2, numResults)
}

type handlerFn func(ctx context.Context, a int, out chan<- int)

func (s *busTestSuite) TestSubscribeAsyncMultipleSubscribers() {
	results := make(chan int)

	bus := New(s.logger)
	s.NoError(bus.SubscribeAsync("topic", func() handlerFn { return func(ctx context.Context, a int, out chan<- int) { out <- a } }(), false))
	s.NoError(bus.SubscribeAsync("topic", func() handlerFn { return func(ctx context.Context, a int, out chan<- int) { out <- a } }(), false))

	bus.Publish(s.Ctx, "topic", 1, results)

	numResults := 0

	go func() {
		for _ = range results {
			numResults++
		}
	}()

	err := <-jet.Await(func() (bool, error) {
		return numResults == 2, nil
	}, time.Millisecond*50, time.Second*3)
	s.NoError(err)
}
