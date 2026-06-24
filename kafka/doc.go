// Package kafka provides Kafka producers and subscribers built on
// segmentio/kafka-go.
//
// A Broker creates producers and subscribers for topics. It supports SASL
// authentication, optional topic auto-creation, worker-based parallel
// consumption (ordered per message key) and request-context propagation via a
// message envelope. Use the builders (NewProducerCfgBuilder,
// NewSubscriberCfgBuilder, NewTopicCfgBuilder) to configure each component.
//
// Subscribers support two delivery guarantees, selected via
// SubscriberConfigBuilder.DeliveryGuarantee: AtMostOnce (the default — parallel,
// fastest, but may drop in-flight messages on shutdown/crash) and AtLeastOnce
// (sequential, commits only after processing, so nothing is lost on
// shutdown/crash at the cost of possible redelivery). See DeliveryGuarantee.
package kafka
