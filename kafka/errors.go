package kafka

import (
	"context"
	"github.com/zloevil/jet"
)

const (
	ErrCodeKafkaFetchMessage             = "KF-001"
	ErrCodeKafkaCommitMessage            = "KF-002"
	ErrCodeKafkaNotInitialized           = "KF-003"
	ErrCodeKafkaInvalidConfig            = "KF-004"
	ErrCodeKafkaMessageContextInvalid    = "KF-005"
	ErrCodeKafkaMessageMarshal           = "KF-006"
	ErrCodeKafkaProducerTopicEmpty       = "KF-008"
	ErrCodeKafkaSubTopicEmpty            = "KF-009"
	ErrCodeKafkaSubNoHandlers            = "KF-010"
	ErrCodeKafkaConnection               = "KF-011"
	ErrCodeKafkaCreateTopics             = "KF-012"
	ErrCodeKafkaMessageWrite             = "KF-013"
	ErrCodeKafkaDecodeMsgUnmarshal       = "KF-014"
	ErrCodeKafkaMsgUnmarshalPayload      = "KF-015"
	ErrCodeKafkaProduceMsg               = "KF-016"
	ErrCodeKafkaSaslNotSupportedType     = "KF-017"
	ErrCodeKafkaSaslGetMechanism         = "KF-018"
	ErrCodeKafkaInvalidDeliveryGuarantee = "KF-019"
	ErrCodeKafkaHandlerPanic             = "KF-020"
)

var (
	ErrKafkaFetchMessage = func(cause error) error {
		return jet.NewAppErrBuilder(ErrCodeKafkaFetchMessage, "").Wrap(cause).Err()
	}
	ErrKafkaCommitMessage = func(cause error) error {
		return jet.NewAppErrBuilder(ErrCodeKafkaCommitMessage, "").Wrap(cause).Err()
	}
	ErrKafkaNotInitialized = func(ctx context.Context) error {
		return jet.NewAppErrBuilder(ErrCodeKafkaNotInitialized, "not initialized").C(ctx).Err()
	}
	ErrKafkaProducerTopicEmpty = func(ctx context.Context) error {
		return jet.NewAppErrBuilder(ErrCodeKafkaProducerTopicEmpty, "topic empty").C(ctx).Err()
	}
	ErrKafkaInvalidConfig = func(ctx context.Context) error {
		return jet.NewAppErrBuilder(ErrCodeKafkaInvalidConfig, "config invalid").C(ctx).Err()
	}
	ErrKafkaMessageContextInvalid = func(ctx context.Context, topic string) error {
		return jet.NewAppErrBuilder(ErrCodeKafkaMessageContextInvalid, "message context invalid").F(jet.KV{"topic": topic}).C(ctx).Err()
	}
	ErrKafkaMessageMarshal = func(ctx context.Context, cause error, topic string) error {
		return jet.NewAppErrBuilder(ErrCodeKafkaMessageMarshal, "").Wrap(cause).F(jet.KV{"topic": topic}).C(ctx).Err()
	}
	ErrKafkaMessageWrite = func(ctx context.Context, cause error, topic string) error {
		return jet.NewAppErrBuilder(ErrCodeKafkaMessageWrite, "").Wrap(cause).F(jet.KV{"topic": topic}).C(ctx).Err()
	}
	ErrKafkaConnection = func(ctx context.Context, cause error) error {
		return jet.NewAppErrBuilder(ErrCodeKafkaConnection, "").Wrap(cause).C(ctx).Err()
	}
	ErrKafkaCreateTopics = func(ctx context.Context, cause error) error {
		return jet.NewAppErrBuilder(ErrCodeKafkaCreateTopics, "").Wrap(cause).C(ctx).Err()
	}
	ErrKafkaDecodeMsgUnmarshal = func(ctx context.Context, cause error) error {
		return jet.NewAppErrBuilder(ErrCodeKafkaDecodeMsgUnmarshal, "").Wrap(cause).C(ctx).Err()
	}
	ErrKafkaMsgUnmarshalPayload = func(ctx context.Context, cause error) error {
		return jet.NewAppErrBuilder(ErrCodeKafkaMsgUnmarshalPayload, "").Wrap(cause).C(ctx).Err()
	}
	ErrKafkaSubTopicEmpty = func(ctx context.Context) error {
		return jet.NewAppErrBuilder(ErrCodeKafkaSubTopicEmpty, "topic empty").C(ctx).Err()
	}
	ErrKafkaSubNoHandlers = func(ctx context.Context) error {
		return jet.NewAppErrBuilder(ErrCodeKafkaSubNoHandlers, "no handlers specified").C(ctx).Err()
	}
	ErrKafkaProduceMsg = func(ctx context.Context, cause error) error {
		return jet.NewAppErrBuilder(ErrCodeKafkaProduceMsg, "").Wrap(cause).C(ctx).Err()
	}
	ErrKafkaSaslNotSupportedType = func(ctx context.Context) error {
		return jet.NewAppErrBuilder(ErrCodeKafkaSaslNotSupportedType, "not supported sasl type").C(ctx).Err()
	}
	ErrKafkaSaslGetMechanism = func(ctx context.Context, cause error) error {
		return jet.NewAppErrBuilder(ErrCodeKafkaSaslGetMechanism, "sasl mechanism").C(ctx).Err()
	}
	ErrKafkaInvalidDeliveryGuarantee = func(ctx context.Context, value string) error {
		return jet.NewAppErrBuilder(ErrCodeKafkaInvalidDeliveryGuarantee, "invalid delivery guarantee").F(jet.KV{"value": value}).C(ctx).Business().Err()
	}
	ErrKafkaHandlerPanic = func(cause error) error {
		return jet.NewAppErrBuilder(ErrCodeKafkaHandlerPanic, "handler panic").Wrap(cause).Err()
	}
)
