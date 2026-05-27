package kafka

import (
	"context"
	stdErr "errors"
	"github.com/segmentio/kafka-go"
	"github.com/zloevil/jet"
	"github.com/zloevil/jet/goroutine"
	"hash"
	"hash/fnv"
	"io"
	"sync"
	"time"
)

const (
	subWorkersPerTopic            = 4
	waitPeriodBeforeReaderRestart = time.Second * 30
)

// SubscriberConfig specifies subscriber config params
// use builder rather than manual population
type SubscriberConfig struct {
	GroupId          string
	BatchTimeout     *time.Duration
	MaxWait          *time.Duration
	CommitInterval   *time.Duration
	Workers          *int
	MaxAttempts      *int
	StartOffset      *int64
	JoinGroupBackoff *time.Duration
	Logging          bool
}

type SubscriberConfigBuilder interface {
	// GroupId allows load balancing messages within the same group
	GroupId(groupId string) SubscriberConfigBuilder
	// BatchTimeout sets timeout of batch fetching from kafka (default: 10s)
	BatchTimeout(to time.Duration) SubscriberConfigBuilder
	// MaxWait sets maximum amount of time to wait for new data to come when fetching batches (default: 10s)
	MaxWait(to time.Duration) SubscriberConfigBuilder
	// CommitInterval sets interval to commit to kafka (default: sync)
	CommitInterval(to time.Duration) SubscriberConfigBuilder
	// Workers sets number of workers (default: 4)
	Workers(num int) SubscriberConfigBuilder
	// StartOffset determines from which offset a new group starts to consume. it must be set to one of FirstOffset = -2 or LastOffset = -1 (Default: FirstOffset)
	// Only used when GroupID is set
	StartOffset(v int64) SubscriberConfigBuilder
	// JoinGroupBackoff optionally sets the length of time to wait between re-joining
	JoinGroupBackoff(t time.Duration) SubscriberConfigBuilder
	// Logging if true subscriber logging enabled
	Logging(v bool) SubscriberConfigBuilder
	// Build builds config
	Build() *SubscriberConfig
}

type subscriber struct {
	readerCfg *kafka.ReaderConfig
	handlers  []HandlerFn
	workers   int
	logger    jet.CLoggerFunc
}

func (s *subscriber) l() jet.CLogger {
	return s.logger().Cmp("kafka-sub")
}

func newSubscriber(logger jet.CLoggerFunc, topic *TopicConfig, cfg *SubscriberConfig, urls []string, dialer *kafka.Dialer, handlers ...HandlerFn) *subscriber {

	// setup reader
	readerCfg := &kafka.ReaderConfig{
		Brokers:     urls,
		GroupID:     cfg.GroupId,
		Topic:       topic.Topic,
		Dialer:      dialer,
		ErrorLogger: kafka.LoggerFunc(logger().Mth("subscriber").F(jet.KV{"topic": topic.Topic, "groupId": cfg.GroupId}).PrintfErr),
	}
	if cfg.CommitInterval != nil {
		readerCfg.CommitInterval = *cfg.CommitInterval
	}
	if cfg.BatchTimeout != nil {
		readerCfg.ReadBatchTimeout = *cfg.BatchTimeout
	}
	if cfg.MaxAttempts != nil {
		readerCfg.MaxAttempts = *cfg.MaxAttempts
	}
	if cfg.MaxWait != nil {
		readerCfg.MaxWait = *cfg.MaxWait
	}
	if cfg.JoinGroupBackoff != nil {
		readerCfg.JoinGroupBackoff = *cfg.JoinGroupBackoff
	}
	if cfg.StartOffset != nil {
		readerCfg.StartOffset = *cfg.StartOffset
	} else {
		readerCfg.StartOffset = kafka.LastOffset
	}
	if cfg.Logging {
		readerCfg.Logger = kafka.LoggerFunc(logger().Mth("subscriber").F(jet.KV{"topic": topic.Topic, "groupId": cfg.GroupId}).Printf)
	}

	// subscriber
	sub := &subscriber{
		readerCfg: readerCfg,
		handlers:  handlers,
		workers:   subWorkersPerTopic,
		logger:    logger,
	}

	if cfg.Workers != nil {
		sub.workers = *cfg.Workers
	}

	return sub
}

func (s *subscriber) start(ctx context.Context, topic string) {
	s.l().C(ctx).Mth("start").F(jet.KV{"topic": topic}).Dbg()

	reader := kafka.NewReader(*s.readerCfg)

	// start goroutine to fetch messages
	goroutine.New().
		WithLogger(s.l().Mth("fetch")).
		WithRetry(goroutine.Unrestricted).
		Go(ctx,
			func() {

				// close reader (may take some time)
				defer func() { _ = reader.Close() }()

				// run workers
				workersChannels := make([]chan kafka.Message, s.workers)
				for i := 0; i < s.workers; i++ {
					workersChannels[i] = make(chan kafka.Message, 255)
					s.subscriberWorker(ctx, topic, s.handlers, i, workersChannels[i])
				}

				// close all worker channels
				defer jet.ForAll(workersChannels, func(c chan kafka.Message) { close(c) })

				l := s.l().C(ctx).Mth("fetch").F(jet.KV{"topic": topic}).Dbg("started")
				for {

					// check if context is already cancelled
					if ctx.Err() != nil {
						l.Dbg("stopped")
						return
					}

					// read message
					m, err := reader.ReadMessage(ctx)
					if err != nil {

						// reader has been closed, restart
						if stdErr.Is(err, io.EOF) || stdErr.Is(err, io.ErrUnexpectedEOF) {
							l.Dbg("EOF -> restart")
							time.AfterFunc(waitPeriodBeforeReaderRestart, func() { s.start(ctx, topic) })
							return
						}

						s.l().Mth("fetch").F(jet.KV{"topic": topic}).E(ErrKafkaFetchMessage(err)).Err("fetch")
						continue
					}
					l.TrcObj("%+v", m)

					// send message to channel to process by workers
					if len(m.Value) != 0 && len(m.Key) != 0 {

						// send message to proper channel
						workersChannels[s.chanIndexByKey(m.Key)] <- m

					}
				}
			},
		)

}

func (s *subscriber) close() {}

var (
	fnv1aPool = &sync.Pool{
		New: func() interface{} {
			return fnv.New32a()
		},
	}
)

// chanIndexByKey calculates index in channel slice by hashing message key
func (s *subscriber) chanIndexByKey(key []byte) int {

	h := fnv1aPool.Get().(hash.Hash32)
	defer fnv1aPool.Put(h)

	h.Reset()
	_, _ = h.Write(key)

	ind := int32(h.Sum32()) % int32(s.workers)
	if ind < 0 {
		ind = -ind
	}

	return int(ind)
}

func (s *subscriber) subscriberWorker(ctx context.Context, topic string, handlers []HandlerFn, workerTag int, receiverChan chan kafka.Message) {

	goroutine.New().
		WithLogger(s.l().Mth("sub-worker")).
		WithRetry(goroutine.Unrestricted).
		Go(ctx,
			func() {
				l := s.l().Mth("worker").F(jet.KV{"tag": workerTag, "topic": topic}).Dbg("started")
				for {
					select {
					case msg, ok := <-receiverChan:

						if !ok {
							l.Dbg("closed")
							return
						}

						l.DbgF("key: %s", string(msg.Key)).TrcF("%s", string(msg.Value))

						// run handler
						for _, handler := range handlers {
							if err := handler(msg.Value); err != nil {
								s.l().C(ctx).Mth("worker").E(err).St().Err()
							}
						}

					case <-ctx.Done():
						l.Dbg("stopped")
						return
					}
				}
			},
		)
}

type subscriberConfigBuilder struct {
	cfg *SubscriberConfig
}

func NewSubscriberCfgBuilder() SubscriberConfigBuilder {
	w := subWorkersPerTopic
	return &subscriberConfigBuilder{
		cfg: &SubscriberConfig{
			Workers: &w,
		},
	}
}

func (p *subscriberConfigBuilder) MaxWait(to time.Duration) SubscriberConfigBuilder {
	p.cfg.MaxWait = &to
	return p
}

func (p *subscriberConfigBuilder) GroupId(groupId string) SubscriberConfigBuilder {
	p.cfg.GroupId = groupId
	return p
}

func (p *subscriberConfigBuilder) CommitInterval(to time.Duration) SubscriberConfigBuilder {
	p.cfg.CommitInterval = &to
	return p
}

func (p *subscriberConfigBuilder) Workers(num int) SubscriberConfigBuilder {
	p.cfg.Workers = &num
	return p
}

func (p *subscriberConfigBuilder) BatchTimeout(to time.Duration) SubscriberConfigBuilder {
	p.cfg.BatchTimeout = &to
	return p
}

func (p *subscriberConfigBuilder) StartOffset(v int64) SubscriberConfigBuilder {
	p.cfg.StartOffset = &v
	return p
}

func (p *subscriberConfigBuilder) JoinGroupBackoff(t time.Duration) SubscriberConfigBuilder {
	p.cfg.JoinGroupBackoff = &t
	return p
}

func (p *subscriberConfigBuilder) Logging(v bool) SubscriberConfigBuilder {
	p.cfg.Logging = v
	return p
}

func (p *subscriberConfigBuilder) Build() *SubscriberConfig {
	if p.cfg.GroupId == "" {
		p.cfg.GroupId = jet.NewRandString()
	}
	return p.cfg
}
