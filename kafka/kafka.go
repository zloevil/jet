package kafka

import (
	"context"
	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/segmentio/kafka-go/sasl/scram"
	"github.com/zloevil/jet"
	"github.com/zloevil/jet/goroutine"
	"strings"
	"sync"
)

const (
	SaslTypePlain       = "plain"
	SaslTypeScramSha256 = "sha256"
	SaslTypeScramSha512 = "sha512"
)

type Sasl struct {
	Enabled  bool   // Enabled true if Sasl must be applied
	Type     string // Type of mechanism (plain, sha256, sha512)
	Username string // Username
	Password string // Password
}

// BrokerConfig kafka broker configuration
type BrokerConfig struct {
	ClientId          string `mapstructure:"client_id"`           // ClientId identifies client
	TopicAutoCreation bool   `mapstructure:"topic_auto_creation"` // TopicAutoCreation if true topics are created by code (otherwise must be preliminary declared)
	Url               string // Url comma separated host-port pairs ("localhost:9092,localhost:9093")
	Sasl              Sasl   // Sasl configuration
}

// HandlerFn handler function
type HandlerFn func(payload []byte) error

// Broker kafka
type Broker interface {
	// Init initializes broker
	Init(ctx context.Context, cfg *BrokerConfig) error
	// AddProducer adds a producer with configuration
	AddProducer(ctx context.Context, topic *TopicConfig, cfg *ProducerConfig) (Producer, error)
	// AddSubscriber adds a subscriber with configuration
	AddSubscriber(ctx context.Context, topic *TopicConfig, cfg *SubscriberConfig, handlers ...HandlerFn) error
	// DeclareTopics declares topics in kafka broker
	// must be executed after all producer and subscribers added
	DeclareTopics(ctx context.Context) error
	// Start starts listening
	Start(ctx context.Context) error
	// Close closes broker
	Close(ctx context.Context)
}

// subKey used as a key for handlers
type subKey struct {
	Topic   string // Topic
	GroupId string // GroupId load balancing group
}

type brokerImpl struct {
	sync.RWMutex
	logger          jet.CLoggerFunc
	cfg             *BrokerConfig
	urls            []string // urls list of broker urls
	transport       *kafka.Transport
	cancellationCtx context.Context
	cancelFunc      context.CancelFunc
	subscribers     map[subKey]*subscriber
	topics          map[string]kafka.TopicConfig
	saslMechanism   sasl.Mechanism
	conn            *kafka.Conn
	dialer          *kafka.Dialer
}

func NewBroker(logger jet.CLoggerFunc) Broker {
	return &brokerImpl{
		logger:      logger,
		subscribers: map[subKey]*subscriber{},
		topics:      map[string]kafka.TopicConfig{},
	}
}

func (b *brokerImpl) l() jet.CLogger {
	return b.logger().Cmp("kafka")
}

func (b *brokerImpl) Init(ctx context.Context, cfg *BrokerConfig) error {
	l := b.l().Mth("init").F(jet.KV{"client": cfg.ClientId, "url": cfg.Url}).Dbg()

	b.cfg = cfg
	var err error

	// validate
	if b.cfg == nil {
		return ErrKafkaInvalidConfig(ctx)
	}
	if b.cfg.Url == "" {
		return ErrKafkaInvalidConfig(ctx)
	}

	b.Lock()
	defer b.Unlock()

	// get sasl mechanism
	b.saslMechanism, err = b.getSaslMechanism(ctx)
	if err != nil {
		return err
	}

	b.urls = strings.Split(cfg.Url, ",")
	b.transport = &kafka.Transport{
		ClientID: b.cfg.ClientId,
		SASL:     b.saslMechanism,
	}

	// setup connection
	b.dialer = &kafka.Dialer{
		DualStack:     true,
		SASLMechanism: b.saslMechanism,
	}
	b.conn, err = b.dialer.Dial("tcp", b.urls[0])
	if err != nil {
		return ErrKafkaConnection(ctx, err)
	}

	// setup cancellation context
	b.cancellationCtx, b.cancelFunc = context.WithCancel(ctx)

	l.Inf("ok")
	return nil
}

func (b *brokerImpl) AddProducer(ctx context.Context, topic *TopicConfig, cfg *ProducerConfig) (Producer, error) {
	b.l().Mth("add-producer").F(jet.KV{"topic": topic.Topic}).Dbg()

	// validate
	if b.transport == nil {
		return nil, ErrKafkaNotInitialized(ctx)
	}
	if topic.Topic == "" {
		return nil, ErrKafkaProducerTopicEmpty(ctx)
	}

	b.Lock()
	defer b.Unlock()

	// register topic
	b.topics[topic.Topic] = getTopicCfg(topic)

	// create and return producer
	return newProducer(b.cancellationCtx, b.logger, topic, cfg, b.urls, b.transport), nil
}

func (b *brokerImpl) AddSubscriber(ctx context.Context, topic *TopicConfig, cfg *SubscriberConfig, handlers ...HandlerFn) error {
	b.l().Mth("add-subscriber").F(jet.KV{"topic": topic.Topic}).Dbg()

	// validation
	if b.transport == nil {
		return ErrKafkaNotInitialized(ctx)
	}
	if topic.Topic == "" {
		return ErrKafkaSubTopicEmpty(ctx)
	}
	if len(handlers) == 0 {
		return ErrKafkaSubNoHandlers(ctx)
	}
	if cfg.DeliveryGuarantee != "" && cfg.DeliveryGuarantee != AtMostOnce && cfg.DeliveryGuarantee != AtLeastOnce {
		return ErrKafkaInvalidDeliveryGuarantee(ctx, string(cfg.DeliveryGuarantee))
	}

	b.Lock()
	defer b.Unlock()

	// register topic
	b.topics[topic.Topic] = getTopicCfg(topic)

	// register subscriber
	b.subscribers[subKey{Topic: topic.Topic, GroupId: cfg.GroupId}] = newSubscriber(b.logger, topic, cfg, b.urls, b.dialer, handlers...)
	return nil
}

func (b *brokerImpl) DeclareTopics(ctx context.Context) error {
	l := b.l().C(ctx).Mth("declare").Dbg()

	// skip if auto-creation isn't configured
	if !b.cfg.TopicAutoCreation {
		l.Dbg("skip")
		return nil
	}

	b.Lock()
	defer b.Unlock()

	// create topics
	var topics []kafka.TopicConfig
	for _, t := range b.topics {
		topics = append(topics, t)
	}
	err := b.conn.CreateTopics(topics...)
	if err != nil {
		return ErrKafkaCreateTopics(ctx, err)
	}

	l.Dbg("ok")

	return nil
}

func (b *brokerImpl) Start(ctx context.Context) error {
	b.l().C(ctx).Mth("start").Dbg()

	// declare topics first
	err := b.DeclareTopics(ctx)
	if err != nil {
		return err
	}

	b.Lock()
	defer b.Unlock()

	// start all subscribers
	for key, sub := range b.subscribers {
		sub.start(b.cancellationCtx, key.Topic)
	}
	return nil
}

func (b *brokerImpl) Close(ctx context.Context) {
	l := b.l().C(ctx).Mth("close").Dbg()
	if b.cancellationCtx == nil {
		return
	}

	b.Lock()
	defer b.Unlock()

	// close all the readers
	eg := goroutine.NewGroup(ctx).WithLogger(l)
	for _, sub := range b.subscribers {
		eg.Go(func() error {
			sub.close()
			return nil
		})
	}
	_ = eg.Wait()

	_ = b.conn.Close()
	b.cancelFunc()
}

func (b *brokerImpl) getSaslMechanism(ctx context.Context) (sasl.Mechanism, error) {
	if !b.cfg.Sasl.Enabled {
		return nil, nil
	}
	switch b.cfg.Sasl.Type {
	case SaslTypePlain:
		return plain.Mechanism{Username: b.cfg.Sasl.Username, Password: b.cfg.Sasl.Password}, nil
	case SaslTypeScramSha256:
		m, err := scram.Mechanism(scram.SHA256, b.cfg.Sasl.Username, b.cfg.Sasl.Password)
		if err != nil {
			return nil, ErrKafkaSaslGetMechanism(ctx, err)
		}
		return m, nil
	case SaslTypeScramSha512:
		m, err := scram.Mechanism(scram.SHA512, b.cfg.Sasl.Username, b.cfg.Sasl.Password)
		if err != nil {
			return nil, ErrKafkaSaslGetMechanism(ctx, err)
		}
		return m, nil
	default:
		return nil, ErrKafkaSaslNotSupportedType(ctx)
	}
}
