package batch

import (
	"context"
	"github.com/stretchr/testify/suite"
	"github.com/zloevil/jet"
	"testing"
	"time"
)

type batchWorkerTestSuite struct {
	jet.Suite
}

type message struct {
	Value string
}

func (s *batchWorkerTestSuite) SetupSuite() {
	s.Suite.Init(func() jet.CLogger { return jet.L(jet.InitLogger(&jet.LogConfig{Level: jet.TraceLevel})) })
}

type mockWriter struct {
	items []*message
}

func (b *mockWriter) Write(ctx context.Context, items []*message) error {
	b.items = append(b.items, items...)
	return nil
}

func (s *batchWorkerTestSuite) SetupTest() {
}

func (s *batchWorkerTestSuite) TearDownSuite() {}

func TestBatchWriterSuite(t *testing.T) {
	suite.Run(t, new(batchWorkerTestSuite))
}

func (s *batchWorkerTestSuite) Test_WhenWriteByTimeout() {

	mw := &mockWriter{}

	worker := NewBatchWorker[message](mw, &Options{
		Interval:    time.Millisecond * 500,
		MaxItems:    10,
		MaxCapacity: 999,
	}, s.L)

	worker.Start(s.Ctx)
	defer worker.Close(s.Ctx)

	// put5 entries
	for i := 0; i < 5; i++ {
		worker.Write(s.Ctx, &message{Value: jet.NewRandString()})
	}
	// await
	if err := <-jet.Await(func() (bool, error) {
		return len(mw.items) == 5, nil
	}, time.Millisecond*100, time.Second*2); err != nil {
		s.Fatal(err)
	}
}

func (s *batchWorkerTestSuite) Test_WhenWriteByMaxItems() {

	mw := &mockWriter{}

	worker := NewBatchWorker[message](mw, &Options{
		Interval:    time.Second * 5,
		MaxItems:    5,
		MaxCapacity: 999,
	}, s.L)

	worker.Start(s.Ctx)
	defer worker.Close(s.Ctx)

	// put5 entries
	for i := 0; i < 5; i++ {
		worker.Write(s.Ctx, &message{Value: jet.NewRandString()})
	}
	// await
	if err := <-jet.Await(func() (bool, error) {
		return len(mw.items) == 5, nil
	}, time.Millisecond*100, time.Second*2); err != nil {
		s.Fatal(err)
	}
}

func (s *batchWorkerTestSuite) Test_WhenWRiteOnClose() {

	mw := &mockWriter{}

	worker := NewBatchWorker[message](mw, &Options{
		Interval:    time.Second * 5,
		MaxItems:    10,
		MaxCapacity: 999,
	}, s.L)

	worker.Start(s.Ctx)

	// put5 entries
	for i := 0; i < 5; i++ {
		worker.Write(s.Ctx, &message{Value: jet.NewRandString()})
	}

	worker.Close(s.Ctx)

	// await
	if err := <-jet.Await(func() (bool, error) {
		return len(mw.items) == 5, nil
	}, time.Millisecond*100, time.Second*2); err != nil {
		s.Fatal(err)
	}
}
