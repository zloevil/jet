package event

import (
	"context"
	"errors"
	"github.com/stretchr/testify/suite"
	"github.com/zloevil/jet"
	"sync/atomic"
	"testing"
)

type eventTestSuite struct {
	jet.Suite
}

func (s *eventTestSuite) SetupSuite() {
	s.Suite.Init(func() jet.CLogger { return jet.L(jet.InitLogger(&jet.LogConfig{Level: jet.TraceLevel})) })
}

func (s *eventTestSuite) SetupTest() {}

func (s *eventTestSuite) TearDownSuite() {}

func TestEventSuite(t *testing.T) {
	suite.Run(t, new(eventTestSuite))
}

func (s *eventTestSuite) Test_RegisterHandle() {
	i := atomic.Int32{}
	f := func(ctx context.Context, val int32) error {
		i.Add(val)
		return nil
	}
	event := NewEventHandler[int32](s.L)

	event.Register(f)
	event.Register(f)
	event.Register(f)

	event.ExecuteAsync(s.Ctx, 10)
	event.Wait()
	s.Equal(int32(30), i.Load())
}

func (s *eventTestSuite) Test_Execute() {
	event := NewEventHandler[int](s.L)

	i := 0
	f := func(ctx context.Context, val int) error {
		i += val
		return nil
	}
	event.Register(f)
	event.Register(f)

	s.NoError(event.Execute(s.Ctx, 10))
	s.Equal(20, i)
}

func (s *eventTestSuite) Test_IfOnFails_ReturnErr() {
	event := NewEventHandler[int](s.L)

	i := 0
	f := func(ctx context.Context, val int) error {
		i += val
		return nil
	}
	fErr := func(ctx context.Context, val int) error {
		return errors.New("internal")
	}
	event.Register(f)
	event.Register(f)
	event.Register(fErr)
	event.Register(f)
	event.Register(f)

	s.Error(event.Execute(s.Ctx, 10))
	s.Equal(20, i)
}

func (s *eventTestSuite) Test_RegisterHandlerInHandler() {
	event1 := NewEventHandler[int32](s.L)
	event2 := NewEventHandler[int32](s.L)

	i := atomic.Int32{}
	f1 := func(ctx context.Context, val int32) error {
		i.Add(val)
		return nil
	}
	f2 := func(ctx context.Context, val int32) error {
		i.Add(val)
		event2.ExecuteAsync(ctx, val+5)
		return nil
	}
	event2.Register(f1)
	event1.Register(f2)

	event1.ExecuteAsync(s.Ctx, 10)

	event1.Wait()
	event2.Wait()

	s.Equal(int32(25), i.Load())
}
