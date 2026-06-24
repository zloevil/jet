package kafka

import (
	"context"
	stdErr "errors"
	"fmt"
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
	// handlerRetryDelay is the backoff between at-least-once retries of a failing message.
	handlerRetryDelay = time.Second
	// commitTimeout bounds the detached at-least-once offset commit.
	commitTimeout = time.Second * 15
)

// DeliveryGuarantee controls when a consumed message's offset is committed
// relative to handler processing.
type DeliveryGuarantee string

const (
	// AtMostOnce commits the offset as soon as a message is read, before the
	// handlers run. It is the default. It keeps the parallel per-key worker pool
	// and is the fastest option, but messages that are buffered or in-flight when
	// the process stops (SIGTERM, scale-down) or crashes are lost, because their
	// offset has already advanced.
	AtMostOnce DeliveryGuarantee = "at_most_once"
	// AtLeastOnce commits the offset only after a handler returns nil, so nothing
	// is lost on shutdown or crash — an uncommitted message is redelivered, and
	// handlers must therefore be idempotent. A handler that returns an error (or
	// panics) is retried on the same message until it succeeds; return nil to skip
	// a message that cannot be processed. Messages are processed sequentially
	// (Kafka tracks a single offset per partition, so out-of-order commits would
	// skip unprocessed messages), which lowers throughput compared to AtMostOnce.
	AtLeastOnce DeliveryGuarantee = "at_least_once"
)

// SubscriberConfig specifies subscriber config params
// use builder rather than manual population
type SubscriberConfig struct {
	GroupId           string
	BatchTimeout      *time.Duration
	MaxWait           *time.Duration
	CommitInterval    *time.Duration
	Workers           *int
	MaxAttempts       *int
	StartOffset       *int64
	JoinGroupBackoff  *time.Duration
	Logging           bool
	DeliveryGuarantee DeliveryGuarantee
}

type SubscriberConfigBuilder interface {
	// GroupId allows load balancing messages within the same group
	GroupId(groupId string) SubscriberConfigBuilder
	// BatchTimeout sets timeout of batch fetching from kafka (default: 10s)
	BatchTimeout(to time.Duration) SubscriberConfigBuilder
	// MaxWait sets maximum amount of time to wait for new data to come when fetching batches (default: 10s)
	MaxWait(to time.Duration) SubscriberConfigBuilder
	// CommitInterval sets interval to commit to kafka (default: sync). Ignored for AtLeastOnce, which always commits synchronously
	CommitInterval(to time.Duration) SubscriberConfigBuilder
	// Workers sets number of workers (default: 4). Only used with AtMostOnce delivery (AtLeastOnce is sequential)
	Workers(num int) SubscriberConfigBuilder
	// StartOffset determines from which offset a new group starts to consume. it must be set to one of FirstOffset = -2 or LastOffset = -1 (Default: FirstOffset)
	// Only used when GroupID is set
	StartOffset(v int64) SubscriberConfigBuilder
	// JoinGroupBackoff optionally sets the length of time to wait between re-joining
	JoinGroupBackoff(t time.Duration) SubscriberConfigBuilder
	// Logging if true subscriber logging enabled
	Logging(v bool) SubscriberConfigBuilder
	// DeliveryGuarantee sets delivery semantics: AtMostOnce (default) or
	// AtLeastOnce. See the DeliveryGuarantee constants for the tradeoffs.
	DeliveryGuarantee(g DeliveryGuarantee) SubscriberConfigBuilder
	// Build builds config
	Build() *SubscriberConfig
}

type subscriber struct {
	readerCfg   *kafka.ReaderConfig
	handlers    []HandlerFn
	workers     int
	atLeastOnce bool
	logger      jet.CLoggerFunc
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
	// at-least-once needs synchronous commits: kafka-go only honors an explicit
	// CommitMessages call when CommitInterval is 0; a non-zero interval would make
	// the post-handler commit async, weakening the after-success guarantee.
	if cfg.DeliveryGuarantee == AtLeastOnce {
		readerCfg.CommitInterval = 0
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
		readerCfg:   readerCfg,
		handlers:    handlers,
		workers:     subWorkersPerTopic,
		atLeastOnce: cfg.DeliveryGuarantee == AtLeastOnce,
		logger:      logger,
	}

	if cfg.Workers != nil {
		sub.workers = *cfg.Workers
	}

	return sub
}

func (s *subscriber) start(ctx context.Context, topic string) {
	s.l().C(ctx).Mth("start").F(jet.KV{"topic": topic, "atLeastOnce": s.atLeastOnce}).Dbg()

	reader := kafka.NewReader(*s.readerCfg)

	// start goroutine to fetch messages
	goroutine.New().
		WithLogger(s.l().Mth("fetch")).
		WithRetry(goroutine.Unrestricted).
		Go(ctx,
			func() {

				// close reader (may take some time)
				defer func() { _ = reader.Close() }()

				if s.atLeastOnce {
					s.consumeAtLeastOnce(ctx, topic, reader)
				} else {
					s.consumeAtMostOnce(ctx, topic, reader)
				}
			},
		)

}

// consumeAtMostOnce reads messages and dispatches them to per-key workers for
// parallel processing. The offset is committed on read (by ReadMessage), before
// the handlers run: it is the fastest option, but in-flight/buffered messages
// are lost if the process stops or crashes since the offset has already moved.
func (s *subscriber) consumeAtMostOnce(ctx context.Context, topic string, reader *kafka.Reader) {

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

		// read message (commits the offset immediately, before processing)
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
}

// consumeAtLeastOnce reads messages, runs the handlers and commits the offset
// only after a handler succeeds, so nothing is lost on shutdown/crash at the
// cost of possible redelivery (handlers must be idempotent). Messages are
// processed sequentially to keep offsets committed in order: Kafka tracks a
// single offset per partition, so committing out of order would skip unprocessed
// messages. A handler that errors or panics is retried on the same message; a
// handler returns nil to skip a message it cannot process.
func (s *subscriber) consumeAtLeastOnce(ctx context.Context, topic string, reader *kafka.Reader) {

	l := s.l().C(ctx).Mth("fetch").F(jet.KV{"topic": topic}).Dbg("started (at-least-once)")
	for {

		// check if context is already cancelled
		if ctx.Err() != nil {
			l.Dbg("stopped")
			return
		}

		// fetch a message WITHOUT committing its offset
		m, err := reader.FetchMessage(ctx)
		if err != nil {

			// stop on cancellation (shutdown) — the in-flight message is redelivered
			if ctx.Err() != nil {
				l.Dbg("stopped")
				return
			}

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

		// process the message; commit ONLY after a handler succeeds. on failure the
		// same message is retried, so the offset never advances past an unprocessed
		// message — nothing is lost.
		if !s.handleUntilDoneOrStopped(ctx, topic, m) {
			return // shutdown during processing — leave the offset uncommitted for redelivery
		}

		// commit after successful processing. use a context detached from
		// cancellation (bounded by commitTimeout) so a message that just succeeded
		// is still committed during graceful shutdown instead of being redelivered.
		commitCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), commitTimeout)
		err = reader.CommitMessages(commitCtx, m)
		cancel()
		if err != nil {
			s.l().Mth("commit").F(jet.KV{"topic": topic}).E(ErrKafkaCommitMessage(err)).Err("commit")
		}
	}
}

// handleUntilDoneOrStopped runs the handlers for m, retrying the same message on
// failure until a handler succeeds. It returns true once the message has been
// processed (and may be committed), or false if the context is cancelled first
// (shutdown) — in which case the offset must stay uncommitted for redelivery.
func (s *subscriber) handleUntilDoneOrStopped(ctx context.Context, topic string, m kafka.Message) bool {

	// match the at-most-once filter: only keyed, non-empty messages reach the
	// handlers; anything else is committed as-is to advance the offset.
	if len(m.Value) == 0 || len(m.Key) == 0 {
		return true
	}

	for {
		if ctx.Err() != nil {
			return false
		}

		if err := s.runHandlers(m.Value); err == nil {
			return true
		} else {
			s.l().C(ctx).Mth("worker").F(jet.KV{"topic": topic}).E(err).Warn("handler failed, retrying message")
		}

		// ctx-aware backoff before retrying the same message
		if !sleepCtx(ctx, handlerRetryDelay) {
			return false
		}
	}
}

// runHandlers invokes each handler for the payload, recovering a panic into an
// error so a single bad message cannot crash the consumer goroutine. It returns
// the first handler error (or recovered panic), or nil when all handlers succeed.
func (s *subscriber) runHandlers(value []byte) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = ErrKafkaHandlerPanic(fmt.Errorf("%v", r))
		}
	}()
	for _, handler := range s.handlers {
		if e := handler(value); e != nil {
			return e
		}
	}
	return nil
}

func (s *subscriber) close() {}

// sleepCtx waits for d or until ctx is cancelled, returning false if cancelled.
func sleepCtx(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return true
	case <-ctx.Done():
		return false
	}
}

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

func (p *subscriberConfigBuilder) DeliveryGuarantee(g DeliveryGuarantee) SubscriberConfigBuilder {
	p.cfg.DeliveryGuarantee = g
	return p
}

func (p *subscriberConfigBuilder) Build() *SubscriberConfig {
	if p.cfg.GroupId == "" {
		p.cfg.GroupId = jet.NewRandString()
	}
	if p.cfg.DeliveryGuarantee == "" {
		p.cfg.DeliveryGuarantee = AtMostOnce
	}
	return p.cfg
}
