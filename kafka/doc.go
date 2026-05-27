// Package kafka provides Kafka producers and subscribers built on
// segmentio/kafka-go.
//
// A Broker creates producers and subscribers for topics. It supports SASL
// authentication, optional topic auto-creation, worker-based parallel
// consumption (ordered per message key) and request-context propagation via a
// message envelope. Use the builders (NewProducerCfgBuilder,
// NewSubscriberCfgBuilder, NewTopicCfgBuilder) to configure each component.
package kafka
