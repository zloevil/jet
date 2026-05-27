package client

import (
	"context"
	"github.com/go-viper/mapstructure/v2"
	"github.com/zloevil/jet"
	"github.com/zloevil/jet/cluster"
	"github.com/zloevil/jet/kafka"
	"github.com/zloevil/jet/rpc"
	"time"
)

type msgTypeBodyProviders map[rpc.MessageType]func() interface{}

type rpcClient struct {
	callProducer      kafka.Producer
	rqPool            *rpc.RequestPool
	logger            jet.CLoggerFunc
	bodyTypeProviders msgTypeBodyProviders
	distributedKeys   cluster.DistributedKeys
	clusterSupport    bool
}

func NewClient(logger jet.CLoggerFunc, callProducer kafka.Producer, distributedKeys cluster.DistributedKeys, config *rpc.Config) rpc.Client {
	return &rpcClient{
		callProducer:      callProducer,
		logger:            logger,
		rqPool:            rpc.NewRequestPool(logger, config.CallTimeOut),
		bodyTypeProviders: map[rpc.MessageType]func() interface{}{},
		distributedKeys:   distributedKeys,
		clusterSupport:    config.ClusterSupport,
	}
}

func (r *rpcClient) l() jet.CLogger {
	return r.logger().Cmp("rpc-client")
}

func (r *rpcClient) validateRawMessage(ctx context.Context, msg *rpc.RawMessage) error {
	if msg.RequestId == "" {
		return rpc.ErrRpcMsgNoRequestId(ctx)
	}
	if msg.Key == "" {
		return rpc.ErrRpcMsgNoKey(ctx)
	}
	return nil
}

func (r *rpcClient) Start(ctx context.Context) {
	r.l().C(ctx).Mth("start").Dbg()
	r.rqPool.Start(ctx)
}

func (r *rpcClient) Close(ctx context.Context) {
	r.l().C(ctx).Mth("close").Dbg()
	r.rqPool.Stop()
}

func (r *rpcClient) SetExpirationCallback(callback rpc.Callback) {
	r.rqPool.SetExpirationCallback(callback)
}

func (r *rpcClient) RegisterBodyTypeProvider(messageType rpc.MessageType, provider rpc.MessageBodyTypeProvider) {
	r.bodyTypeProviders[messageType] = provider
}

func (r *rpcClient) Call(ctx context.Context, msg *rpc.Message, callback rpc.ResponseCallback) error {
	l := r.l().C(ctx).Mth("call").F(jet.KV{"type": msg.Type, "key": msg.Key, "rqId": msg.RequestId}).Dbg()
	// validate request
	if msg.RequestId == "" {
		return rpc.ErrRpcMsgNoRequestId(ctx)
	}
	if msg.Key == "" {
		return rpc.ErrRpcMsgNoKey(ctx)
	}
	if msg.ResponseRequired && callback == nil {
		return rpc.ErrRpcCallNoCb(ctx)
	}
	if msg.ResponseRequired {
		// put request to pool
		r.rqPool.Queue(ctx, &rpc.Request{Ctx: ctx, Msg: msg, Callback: callback})
	}
	// send to kafka
	err := r.callProducer.Send(ctx, msg.Key, msg)
	if err != nil {
		return err
	}
	l.Dbg("ok")
	return nil
}

func (r *rpcClient) ResponseHandler(msg []byte) error {
	// decode response
	decoded, ctx, err := kafka.Decode[rpc.RawMessage](context.Background(), msg)
	if err != nil {
		return err
	}
	rawMsg := &decoded
	l := r.l().C(ctx).Mth("response").F(jet.KV{"type": rawMsg.Type, "key": rawMsg.Key, "rqId": rawMsg.RequestId}).Dbg()
	// validate raw message
	err = r.validateRawMessage(ctx, rawMsg)
	if err != nil {
		return err
	}
	// cluster support, check key in the connection list
	if r.clusterSupport && r.distributedKeys != nil {
		if !r.distributedKeys.Check(rawMsg.Key) {
			l.Dbg("no key, skip")
			return nil
		}
	}
	// check for request in pool
	rq := r.rqPool.TryDequeue(rawMsg.RequestId)
	if rq == nil {
		return rpc.ErrRpcRespNoRequestInPool(ctx, rawMsg.RequestId, rawMsg.Key)
	}
	// prepare response message
	rsMsg := &rpc.Message{
		Type:             rawMsg.Type,
		RequestId:        rawMsg.RequestId,
		Key:              rawMsg.Key,
		ResponseRequired: rawMsg.ResponseRequired,
	}
	// check for body type provider
	// if provider registered, try to convert raw body to the provided type
	// otherwise response with the raw body
	provider, ok := r.bodyTypeProviders[rawMsg.Type]
	if ok {
		bodyTyped := provider()
		d, _ := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
			TagName:    "json",
			Result:     bodyTyped,
			DecodeHook: mapstructure.StringToTimeHookFunc(time.RFC3339),
		})
		err = d.Decode(rawMsg.Body)
		if err != nil {
			return rpc.ErrRpcRespInvalidBody(err, ctx, rawMsg.RequestId, rawMsg.Key)
		}
		rsMsg.Body = bodyTyped
	} else {
		rsMsg.Body = rawMsg.Body
	}
	err = rq.Callback(rq.Ctx, rq.Msg, rsMsg)
	if err != nil {
		return err
	}
	l.Dbg("ok")
	return nil
}
