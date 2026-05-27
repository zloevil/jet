//go:build integration

package kafka

import (
	"fmt"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/suite"
	"github.com/zloevil/jet"
	"go.uber.org/atomic"
	"sync"
	"testing"
	"time"
)

type kafkaTestSuite struct {
	jet.Suite
	logger    jet.CLoggerFunc
	brokerCfg *BrokerConfig
}

func (s *kafkaTestSuite) SetupSuite() {
	s.logger = func() jet.CLogger { return jet.L(jet.InitLogger(&jet.LogConfig{Level: jet.TraceLevel})) }
	s.Suite.Init(s.logger)
	s.brokerCfg = &BrokerConfig{
		ClientId:          jet.NewRandString(),
		Url:               "localhost:9092",
		TopicAutoCreation: true,
	}
}

func TestKafkaSuite(t *testing.T) {
	suite.Run(t, new(kafkaTestSuite))
}

func (s *kafkaTestSuite) Test_OnePubOneSub_OnePartition_NoGroup_SameKey() {

	// declare topic
	part := 1
	topic := &TopicConfig{
		Topic:      jet.NewRandString(),
		Partitions: &part,
	}

	// init sub broker
	subBroker := NewBroker(s.logger)
	err := subBroker.Init(s.Ctx, s.brokerCfg)
	if err != nil {
		s.Fatal(err)
	}

	// declare sub
	wg := jet.NewWG()
	err = subBroker.AddSubscriber(s.Ctx, topic, NewSubscriberCfgBuilder().
		Workers(1).
		JoinGroupBackoff(time.Millisecond*500).
		StartOffset(kafka.FirstOffset).
		MaxWait(time.Second).
		Logging(true).
		Build(), s.handler(0, time.Millisecond*50, wg, nil))
	if err != nil {
		s.Fatal(err)
	}

	// start sub broker
	err = subBroker.Start(s.Ctx)
	if err != nil {
		s.Fatal(err)
	}
	defer func() { subBroker.Close(s.Ctx) }()

	// init pub broker
	pubBroker := NewBroker(s.logger)
	err = pubBroker.Init(s.Ctx, s.brokerCfg)
	if err != nil {
		s.Fatal(err)
	}

	// declare producer
	producer, err := pubBroker.AddProducer(s.Ctx, topic,
		NewProducerCfgBuilder().
			BatchTimeout(time.Millisecond*300).
			BatchSize(1).
			Async(true).
			Build())
	if err != nil {
		s.Fatal(err)
	}
	err = pubBroker.Start(s.Ctx)
	if err != nil {
		s.Fatal(err)
	}
	defer func() { pubBroker.Close(s.Ctx) }()

	// produce 10 messages
	err = s.produceMessages(producer, wg, s.genKeys(10, "k")...)
	if err != nil {
		s.Fatal(err)
	}
	s.True(wg.Wait(time.Millisecond * 500))
}

func (s *kafkaTestSuite) Test_OnePubMultipleSubs_MultiplePartitions_NoGroup() {

	// declare topic
	part := 3
	topic := &TopicConfig{
		Topic:      jet.NewRandString(),
		Partitions: &part,
	}

	// init subs broker
	received := atomic.NewInt32(0)
	counter := func(i int, p []byte) {
		received.Inc()
	}

	var subBrokers []Broker
	for i := 0; i < 3; i++ {

		subBroker := NewBroker(s.logger)
		subBrokers = append(subBrokers, subBroker)
		err := subBroker.Init(s.Ctx, s.brokerCfg)
		if err != nil {
			s.Fatal(err)
		}

		// declare sub
		err = subBroker.AddSubscriber(s.Ctx, topic, NewSubscriberCfgBuilder().
			JoinGroupBackoff(time.Millisecond*500).
			StartOffset(kafka.FirstOffset).
			MaxWait(time.Second).
			Logging(true).
			Workers(1).
			Build(), s.handler(i, time.Millisecond*200, nil, counter))
		if err != nil {
			s.Fatal(err)
		}
		err = subBroker.Start(s.Ctx)
		if err != nil {
			s.Fatal(err)
		}
		defer func() { subBroker.Close(s.Ctx) }()
	}

	// init pub broker
	pubBroker := NewBroker(s.logger)
	err := pubBroker.Init(s.Ctx, s.brokerCfg)
	if err != nil {
		s.Fatal(err)
	}

	// declare producer
	producer, err := pubBroker.AddProducer(s.Ctx, topic,
		NewProducerCfgBuilder().
			BatchSize(1).
			Async(true).
			BatchTimeout(time.Millisecond*200).
			Build())
	if err != nil {
		s.Fatal(err)
	}
	err = pubBroker.Start(s.Ctx)
	if err != nil {
		s.Fatal(err)
	}
	defer func() { pubBroker.Close(s.Ctx) }()

	// produce 10 messages
	err = s.produceMessages(producer, nil, s.genKeys(2, "k")...)
	if err != nil {
		s.Fatal(err)
	}

	// await number of messages
	if err := <-jet.Await(func() (bool, error) {
		s.L().DbgF("received: %d", received.Load())
		return received.Load() == 6, nil
	}, time.Millisecond*100, time.Second*3); err != nil {
		s.Fatal(err)
	}
}

func (s *kafkaTestSuite) Test_MultiplePub_SubGroupOneWorker_MultiplePartitions_SameKey() {

	// declare topic
	part := 3
	topic := &TopicConfig{
		Topic:      jet.NewRandString(),
		Partitions: &part,
	}

	wg := jet.NewWG()
	groupId := jet.NewRandString()

	// report
	subReport := make(map[int]int)
	mtx := &sync.Mutex{}
	callback := func(i int, bytes []byte) {
		mtx.Lock()
		defer mtx.Unlock()
		subReport[i]++
	}

	// init sub brokers
	for i := 0; i < 3; i++ {
		subBroker := NewBroker(s.logger)
		err := subBroker.Init(s.Ctx, s.brokerCfg)
		if err != nil {
			s.Fatal(err)
		}

		// declare sub
		err = subBroker.AddSubscriber(s.Ctx, topic, NewSubscriberCfgBuilder().
			GroupId(groupId).
			JoinGroupBackoff(time.Millisecond*500).
			StartOffset(kafka.FirstOffset).
			MaxWait(time.Second).
			Logging(true).
			Workers(1).
			Build(), s.handler(i, time.Millisecond*100, wg, callback))
		if err != nil {
			s.Fatal(err)
		}

		// start sub broker
		err = subBroker.Start(s.Ctx)
		if err != nil {
			s.Fatal(err)
		}
		defer func() { subBroker.Close(s.Ctx) }()
	}

	// init pub brokers
	var producers []Producer
	for i := 0; i < 3; i++ {

		pubBroker := NewBroker(s.logger)
		err := pubBroker.Init(s.Ctx, s.brokerCfg)
		if err != nil {
			s.Fatal(err)
		}

		// declare producer
		producer, err := pubBroker.AddProducer(s.Ctx, topic,
			NewProducerCfgBuilder().
				BatchSize(1).
				Async(true).
				BatchTimeout(time.Millisecond*200).
				Build())
		if err != nil {
			s.Fatal(err)
		}
		producers = append(producers, producer)
		err = pubBroker.Start(s.Ctx)
		if err != nil {
			s.Fatal(err)
		}
		defer func() { pubBroker.Close(s.Ctx) }()
	}

	// produce messages
	pubReport := make(map[int]int)
	for i := 0; i < 10; i++ {
		for j := 0; j < len(producers); j++ {
			err := s.produceMessages(producers[j], wg, s.genKeys(3, fmt.Sprintf("k%d", j))...)
			if err != nil {
				s.Fatal(err)
			}
			pubReport[j] = pubReport[j] + 3
		}
	}

	s.True(wg.Wait(time.Second * 20))
	s.L().TrcObj("sent: %+v, received: %+v", pubReport, subReport)
}

func (s *kafkaTestSuite) Test_MultiplePub_SubGroupMultipleWorkers_MultiplePartitions_SameKey() {

	// declare topic
	part := 3
	topic := &TopicConfig{
		Topic:      jet.NewRandString(),
		Partitions: &part,
	}

	wg := jet.NewWG()
	groupId := jet.NewRandString()

	// report
	subReport := make(map[int]int)
	mtx := &sync.Mutex{}
	callback := func(i int, bytes []byte) {
		mtx.Lock()
		defer mtx.Unlock()
		subReport[i]++
	}

	// init sub brokers
	for i := 0; i < 3; i++ {
		subBroker := NewBroker(s.logger)
		err := subBroker.Init(s.Ctx, s.brokerCfg)
		if err != nil {
			s.Fatal(err)
		}

		// declare sub
		err = subBroker.AddSubscriber(s.Ctx, topic, NewSubscriberCfgBuilder().
			GroupId(groupId).
			Workers(3).
			JoinGroupBackoff(time.Millisecond*500).
			StartOffset(kafka.FirstOffset).
			MaxWait(time.Second).
			Logging(true).
			Build(), s.handler(i, time.Millisecond*100, wg, callback))
		if err != nil {
			s.Fatal(err)
		}

		// start sub broker
		err = subBroker.Start(s.Ctx)
		if err != nil {
			s.Fatal(err)
		}
		defer func() { subBroker.Close(s.Ctx) }()
	}

	// init pub brokers
	var producers []Producer
	for i := 0; i < 3; i++ {

		pubBroker := NewBroker(s.logger)
		err := pubBroker.Init(s.Ctx, s.brokerCfg)
		if err != nil {
			s.Fatal(err)
		}

		// declare producer
		producer, err := pubBroker.AddProducer(s.Ctx, topic,
			NewProducerCfgBuilder().
				BatchSize(1).
				Async(true).
				BatchTimeout(time.Millisecond*300).
				Build())
		if err != nil {
			s.Fatal(err)
		}
		producers = append(producers, producer)
		err = pubBroker.Start(s.Ctx)
		if err != nil {
			s.Fatal(err)
		}
		defer func() { pubBroker.Close(s.Ctx) }()
	}

	// produce messages
	pubReport := make(map[int]int)
	for i := 0; i < 10; i++ {
		for j := 0; j < len(producers); j++ {
			err := s.produceMessages(producers[j], wg, s.genKeys(3, fmt.Sprintf("k%d", j))...)
			if err != nil {
				s.Fatal(err)
			}
			pubReport[j] = pubReport[j] + 3
		}
	}

	s.True(wg.Wait(time.Second * 20))
	s.L().TrcObj("sent: %+v, received: %+v", pubReport, subReport)
}

func (s *kafkaTestSuite) Test_SinglePub_SubGroup_MultipleWorkers_ProducedByBatch_LongRunningHandler() {

	// declare topic
	part := 3
	topic := &TopicConfig{
		Topic:      jet.NewRandString(),
		Partitions: &part,
	}
	wg := jet.NewWG()
	groupId := jet.NewRandString()

	// report
	subReport := make(map[int]int)
	mtx := &sync.Mutex{}
	callback := func(i int, bytes []byte) {
		mtx.Lock()
		defer mtx.Unlock()
		subReport[i]++
	}

	// init sub brokers
	for i := 0; i < 3; i++ {

		subBroker := NewBroker(s.logger)
		err := subBroker.Init(s.Ctx, s.brokerCfg)
		if err != nil {
			s.Fatal(err)
		}
		// declare sub
		err = subBroker.AddSubscriber(s.Ctx, topic, NewSubscriberCfgBuilder().
			GroupId(groupId).
			Workers(4).
			JoinGroupBackoff(time.Millisecond*500).
			StartOffset(kafka.FirstOffset).
			MaxWait(time.Second).
			Logging(true).
			Build(), s.handler(i, time.Second, wg, callback))
		if err != nil {
			s.Fatal(err)
		}
		// start sub broker
		err = subBroker.Start(s.Ctx)
		if err != nil {
			s.Fatal(err)
		}
		defer func() { subBroker.Close(s.Ctx) }()
	}

	// init pub brokers
	pubBroker := NewBroker(s.logger)
	err := pubBroker.Init(s.Ctx, s.brokerCfg)
	if err != nil {
		s.Fatal(err)
	}
	// declare producer
	producer, err := pubBroker.AddProducer(s.Ctx, topic,
		NewProducerCfgBuilder().
			BatchSize(3).
			Async(true).
			BatchTimeout(time.Millisecond*500).
			Build())
	if err != nil {
		s.Fatal(err)
	}
	err = pubBroker.Start(s.Ctx)
	if err != nil {
		s.Fatal(err)
	}
	defer func() { pubBroker.Close(s.Ctx) }()

	// produce messages
	pubReport := make(map[int]int)
	for i := 0; i < 10; i++ {
		err := s.produceMessages(producer, wg, s.genKeys(3, fmt.Sprintf("k%d", i%3))...)
		if err != nil {
			s.Fatal(err)
		}
		pubReport[i%3] = pubReport[i%3] + 3
	}
	s.True(wg.Wait(time.Second * 60))
	s.L().TrcObj("sent: %+v, received: %+v", pubReport, subReport)
}

func (s *kafkaTestSuite) Test_SubscriberRestart() {

	s.T().Skip()

	// declare topic
	part := 3
	topic := &TopicConfig{
		Topic:      jet.NewRandString(),
		Partitions: &part,
	}
	wg := jet.NewWG()
	groupId := jet.NewRandString()

	// report
	subReport := make(map[int]int)
	mtx := &sync.Mutex{}
	callback := func(i int, bytes []byte) {
		mtx.Lock()
		defer mtx.Unlock()
		subReport[i]++
	}

	newSub := func(i int) Broker {
		subBroker := NewBroker(s.logger)
		err := subBroker.Init(s.Ctx, s.brokerCfg)
		if err != nil {
			s.Fatal(err)
		}
		// declare sub
		err = subBroker.AddSubscriber(s.Ctx, topic, NewSubscriberCfgBuilder().
			GroupId(groupId).
			Workers(1).
			JoinGroupBackoff(time.Millisecond*500).
			StartOffset(kafka.FirstOffset).
			Logging(false).
			Build(), s.handler(i, time.Second, wg, callback))
		if err != nil {
			s.Fatal(err)
		}
		// start sub broker
		err = subBroker.Start(s.Ctx)
		if err != nil {
			s.Fatal(err)
		}
		return subBroker
	}

	sub1 := newSub(1)
	defer func() { sub1.Close(s.Ctx) }()

	sub2 := newSub(2)
	defer func() { sub2.Close(s.Ctx) }()

	sub3 := newSub(3)
	time.AfterFunc(time.Second*30, func() {
		sub3.Close(s.Ctx)
	})

	time.Sleep(time.Second * 1000)

}

func (s *kafkaTestSuite) handler(i int, workTime time.Duration, wg *jet.WaitGroup, callback func(int, []byte)) HandlerFn {
	return func(payload []byte) error {
		time.Sleep(workTime)
		s.L().InfF("[worker: %d] ok", i)
		if wg != nil {
			wg.Done()
		}
		if callback != nil {
			callback(i, payload)
		}
		return nil
	}
}

type payload struct {
	Value string
}

func (s *kafkaTestSuite) produceMessages(producer Producer, wg *jet.WaitGroup, keys ...string) error {
	for _, k := range keys {
		err := producer.Send(s.Ctx, k, &payload{Value: jet.NewRandString()})
		if err != nil {
			return err
		}
		if wg != nil {
			wg.Add(1)
		}
	}
	return nil
}

func (s *kafkaTestSuite) genKeys(num int, v string) []string {
	var r []string
	for i := 0; i < num; i++ {
		if v == "" {
			r = append(r, jet.NewRandString())
		} else {
			r = append(r, v)
		}
	}
	return r
}
