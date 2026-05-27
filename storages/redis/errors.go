package redis

import (
	"context"
	"github.com/zloevil/jet"
)

const (
	ErrCodeRedisPingErr                   = "RDS-001"
	ErrCodeRedisPriorityQueuePushErr      = "RDS-002"
	ErrCodeRedisPriorityQueuePopErr       = "RDS-003"
	ErrCodeRedisPriorityQueuePopRemoveErr = "RDS-004"
)

var (
	ErrRedisPingErr = func(cause error) error {
		return jet.NewAppErrBuilder(ErrCodeRedisPingErr, "").Wrap(cause).Err()
	}
	ErrRedisPriorityQueuePushErr = func(ctx context.Context, cause error) error {
		return jet.NewAppErrBuilder(ErrCodeRedisPriorityQueuePushErr, "priority queue: push").Wrap(cause).Err()
	}
	ErrRedisPriorityQueuePopErr = func(ctx context.Context, cause error) error {
		return jet.NewAppErrBuilder(ErrCodeRedisPriorityQueuePopErr, "priority queue: pop").Wrap(cause).Err()
	}
	ErrRedisPriorityQueuePopRemoveErr = func(ctx context.Context, cause error) error {
		return jet.NewAppErrBuilder(ErrCodeRedisPriorityQueuePopRemoveErr, "priority queue: pop and remove").Wrap(cause).Err()
	}
)
