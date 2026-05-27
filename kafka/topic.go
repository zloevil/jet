package kafka

import "github.com/segmentio/kafka-go"

// TopicConfig topic config
type TopicConfig struct {
	// Topic name
	Topic string
	// Partitions number of partitions
	Partitions *int
	// Configs config params
	Configs map[string]string
}

type TopicParam struct {
	Name  string
	Value string
}

// TopicBuilder simplifies preparing topic config
type TopicBuilder interface {
	// WithPartitionNum setting num of partitions
	WithPartitionNum(num int) TopicBuilder
	// WithParams setting additional params
	WithParams(params ...TopicParam) TopicBuilder
	// Build builds config
	Build() *TopicConfig
}

type topicConfigBuilder struct {
	cfg *TopicConfig
}

func NewTopicCfgBuilder(topic string) TopicBuilder {
	return &topicConfigBuilder{
		cfg: &TopicConfig{
			Topic:   topic,
			Configs: map[string]string{},
		},
	}
}

func (t *topicConfigBuilder) WithPartitionNum(num int) TopicBuilder {
	t.cfg.Partitions = &num
	return t
}

func (t *topicConfigBuilder) WithParams(params ...TopicParam) TopicBuilder {
	for _, p := range params {
		t.cfg.Configs[p.Name] = p.Value
	}
	return t
}

func (t *topicConfigBuilder) Build() *TopicConfig {
	return t.cfg
}

func getTopicCfg(t *TopicConfig) kafka.TopicConfig {
	topicCfg := kafka.TopicConfig{
		Topic:             t.Topic,
		NumPartitions:     -1,
		ReplicationFactor: -1,
	}
	if t.Partitions != nil {
		topicCfg.NumPartitions = *t.Partitions
	}
	return topicCfg
}
