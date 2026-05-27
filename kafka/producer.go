package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/segmentio/kafka-go"
	"github.com/zloevil/jet"
	"time"
)

const (
	defaultRetryTimes   = 3
	defaultRetryTimeout = time.Second
)

// ProducerConfig specifies producer config params
// use builder rather than manual population
type ProducerConfig struct {
	BatchSize    *int
	BatchTimeout *time.Duration
	RequiredAcks *int
	MaxAttempts  *int
	Async        bool
	RetryTimes   *int
	RetryTimeout *time.Duration
}

type ProducerConfigBuilder interface {
	// BatchTimeout sets batch timeout value (default: 1s)
	BatchTimeout(to time.Duration) ProducerConfigBuilder
	// BatchSize sets batch size (default: 100)
	BatchSize(size int) ProducerConfigBuilder
	// Async if true, WriteMessages call will never block but errors aren't returned (default: false)
	Async(v bool) ProducerConfigBuilder
	// Retry sets retry params (default: time=3, timeout = 1s)
	Retry(time int, timeout time.Duration) ProducerConfigBuilder
	// Build builds config
	Build() *ProducerConfig
}

// Producer allows sending message to broker
type Producer interface {
	// Send sends a message to broker
	Send(ctx context.Context, key string, payload interface{}) error
	// SendMany sends bulk of messages to broker
	SendMany(ctx context.Context, messages ...*Message) error
}

type producerImpl struct {
	topic           *TopicConfig
	writer          *kafka.Writer
	logger          jet.CLoggerFunc
	cancellationCtx context.Context
	retryTimes      int
	retryTimeout    time.Duration
}

func (p *producerImpl) l() jet.CLogger {
	return p.logger().Cmp("kafka-producer")
}

func newProducer(ctx context.Context, logger jet.CLoggerFunc, topic *TopicConfig, cfg *ProducerConfig, urls []string, transport *kafka.Transport) Producer {

	// populate writer params
	writer := &kafka.Writer{
		Addr:        kafka.TCP(urls...),
		Topic:       topic.Topic,
		ErrorLogger: kafka.LoggerFunc(logger().Mth("producer").F(jet.KV{"topic": topic.Topic}).PrintfErr),
		Transport:   transport,
		Completion: func(messages []kafka.Message, err error) {
			if err != nil {
				logger().Mth("producer-completion").F(jet.KV{"topic": topic.Topic}).E(ErrKafkaProduceMsg(ctx, err)).Err()
			}
		},
		Balancer: &kafka.Hash{}, // set up hash balancer to guaranty the messages with the same key are always in the same partition
		Async:    cfg.Async,
	}
	if cfg.BatchSize != nil {
		writer.BatchSize = *cfg.BatchSize
	}
	if cfg.BatchTimeout != nil {
		writer.BatchTimeout = *cfg.BatchTimeout
	}
	if cfg.RequiredAcks != nil {
		writer.RequiredAcks = kafka.RequiredAcks(*cfg.RequiredAcks)
	}
	if cfg.MaxAttempts != nil {
		writer.MaxAttempts = *cfg.MaxAttempts
	}

	r := &producerImpl{
		writer:          writer,
		logger:          logger,
		topic:           topic,
		cancellationCtx: ctx,
	}

	if cfg.RetryTimes != nil {
		r.retryTimes = *cfg.RetryTimes
	} else {
		r.retryTimes = defaultRetryTimes
	}
	if cfg.RetryTimeout != nil {
		r.retryTimeout = *cfg.RetryTimeout
	} else {
		r.retryTimeout = defaultRetryTimeout
	}

	return r
}

func (p *producerImpl) Send(ctx context.Context, key string, payload interface{}) error {
	l := p.l().Mth("publish").F(jet.KV{"topic": p.topic.Topic}).Dbg()

	// prepare message
	ctxRq, err := p.rqCtx(ctx)
	if err != nil {
		return err
	}

	msg := &Message{
		Ctx:     ctxRq,
		Payload: payload,
		Key:     key,
	}

	// write message
	err = p.sendWithRetry(ctx, msg)
	if err != nil {
		return err
	}

	l.Dbg("ok").TrcObj("%+v", msg)

	return nil
}

func (p *producerImpl) SendMany(ctx context.Context, messages ...*Message) error {
	l := p.l().Mth("send-many").F(jet.KV{"topic": p.topic.Topic}).Dbg()

	// prepare message
	ctxRq, err := p.rqCtx(ctx)
	if err != nil {
		return err
	}

	for _, m := range messages {
		m.Ctx = ctxRq
	}

	// write message
	err = p.sendWithRetry(ctx, messages...)
	if err != nil {
		return err
	}

	l.Dbg("ok").TrcObj("%+v", messages)

	return nil
}

func (p *producerImpl) rqCtx(ctx context.Context) (*jet.RequestContext, error) {
	if rCtx, ok := jet.Request(ctx); ok {
		return rCtx, nil
	}
	return nil, ErrKafkaMessageContextInvalid(ctx, p.topic.Topic)
}

func (p *producerImpl) sendWithRetry(ctx context.Context, messages ...*Message) error {
	messagesToSend := make([]kafka.Message, 0, len(messages))
	now := jet.Now()
	for _, msg := range messages {

		m, err := json.Marshal(msg)
		if err != nil {
			return ErrKafkaMessageMarshal(ctx, err, p.topic.Topic)
		}

		messagesToSend = append(messagesToSend, kafka.Message{
			Key:   []byte(msg.Key),
			Value: m,
			Time:  now,
		})
	}

	// send with retry
	for i := 0; i < p.retryTimes; i++ {
		err := p.writer.WriteMessages(p.cancellationCtx, messagesToSend...)
		if err != nil {
			if errors.Is(err, kafka.LeaderNotAvailable) {
				time.Sleep(p.retryTimeout)
				continue
			} else {
				return ErrKafkaMessageWrite(ctx, err, p.topic.Topic)
			}
		}
		break
	}
	return nil
}

type producerConfigBuilder struct {
	cfg *ProducerConfig
}

func NewProducerCfgBuilder() ProducerConfigBuilder {
	return &producerConfigBuilder{
		cfg: &ProducerConfig{},
	}
}

func (p *producerConfigBuilder) Retry(times int, timeout time.Duration) ProducerConfigBuilder {
	p.cfg.RetryTimes = &times
	p.cfg.RetryTimeout = &timeout
	return p
}

func (p *producerConfigBuilder) BatchTimeout(to time.Duration) ProducerConfigBuilder {
	p.cfg.BatchTimeout = &to
	return p
}

func (p *producerConfigBuilder) BatchSize(size int) ProducerConfigBuilder {
	p.cfg.BatchSize = &size
	return p
}

func (p *producerConfigBuilder) Async(v bool) ProducerConfigBuilder {
	p.cfg.Async = v
	return p
}

func (p *producerConfigBuilder) Build() *ProducerConfig {
	return p.cfg
}
