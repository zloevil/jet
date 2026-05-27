package rpc

import (
	"context"
	"github.com/zloevil/jet"
)

const (
	ErrCodeRpcMsgNoKey            = "RPC-001"
	ErrCodeRpcMsgNoRequestId      = "RPC-002"
	ErrCodeRpcRespNoRequestInPool = "RPC-003"
	ErrCodeRpcRespInvalidBody     = "RPC-004"
	ErrCodeRpcCallNoCb            = "RPC-005"
)

var (
	ErrRpcMsgNoKey = func(ctx context.Context) error {
		return jet.NewAppErrBuilder(ErrCodeRpcMsgNoKey, "key empty").C(ctx).Err()
	}
	ErrRpcMsgNoRequestId = func(ctx context.Context) error {
		return jet.NewAppErrBuilder(ErrCodeRpcMsgNoRequestId, "Request id empty").C(ctx).Err()
	}
	ErrRpcCallNoCb = func(ctx context.Context) error {
		return jet.NewAppErrBuilder(ErrCodeRpcCallNoCb, "callback id empty").C(ctx).Err()
	}
	ErrRpcRespNoRequestInPool = func(ctx context.Context, rqId, key string) error {
		return jet.NewAppErrBuilder(ErrCodeRpcRespNoRequestInPool, "no Request in pool").C(ctx).F(jet.KV{"rqId": rqId, "key": key}).Err()
	}
	ErrRpcRespInvalidBody = func(cause error, ctx context.Context, rqId, key string) error {
		return jet.NewAppErrBuilder(ErrCodeRpcRespInvalidBody, "no Request in pool").Wrap(cause).C(ctx).F(jet.KV{"rqId": rqId, "key": key}).Err()
	}
)
