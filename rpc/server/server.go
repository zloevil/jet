package server

import (
	"context"
	"github.com/go-viper/mapstructure/v2"
	"github.com/zloevil/jet"
	"github.com/zloevil/jet/cluster"
	"github.com/zloevil/jet/kafka"
	"github.com/zloevil/jet/rpc"
	"time"
)

type msgType struct {
	callback     rpc.Callback
	typeProvider rpc.MessageBodyTypeProvider
}
type msgTypes map[rpc.MessageType]*msgType

type rpcServer struct {
	callProducer    kafka.Producer
	rqPool          *rpc.RequestPool
	logger          jet.CLoggerFunc
	msgTypes        msgTypes
	clusterSupport  bool
	distributedKeys cluster.DistributedKeys
}

func NewServer(logger jet.CLoggerFunc, callProducer kafka.Producer, distributedKeys cluster.DistributedKeys, config *rpc.Config) rpc.Server {
	return &rpcServer{
		callProducer:    callProducer,
		logger:          logger,
		rqPool:          rpc.NewRequestPool(logger, config.CallTimeOut),
		distributedKeys: distributedKeys,
		msgTypes:        msgTypes{},
		clusterSupport:  config.ClusterSupport,
	}
}

func (r *rpcServer) l() jet.CLogger {
	return r.logger().Cmp("rpc-server")
}

func (r *rpcServer) Start(ctx context.Context) {
	r.l().C(ctx).Mth("start").Dbg()
	r.rqPool.Start(ctx)
}

func (r *rpcServer) Close(ctx context.Context) {
	r.l().C(ctx).Mth("close").Dbg()
	r.rqPool.Stop()
}

func (r *rpcServer) SetExpirationCallback(callback rpc.Callback) {
	r.rqPool.SetExpirationCallback(callback)
}

func (r *rpcServer) validateRawMessage(ctx context.Context, msg *rpc.RawMessage) error {
	if msg.RequestId == "" {
		return rpc.ErrRpcMsgNoRequestId(ctx)
	}
	if msg.Key == "" {
		return rpc.ErrRpcMsgNoKey(ctx)
	}
	return nil
}

func (r *rpcServer) RequestHandler(msg []byte) error {
	// decode response
	decoded, ctx, err := kafka.Decode[rpc.RawMessage](context.Background(), msg)
	if err != nil {
		return err
	}
	rawMsg := &decoded
	l := r.l().C(ctx).Mth("request").F(jet.KV{"type": rawMsg.Type, "key": rawMsg.Key, "rqId": rawMsg.RequestId}).Dbg()
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
	// check registered type
	// if not registered, just skip
	msgType, ok := r.msgTypes[rawMsg.Type]
	if !ok {
		l.F(jet.KV{"type": rawMsg.Type}).Dbg("skip")
		return nil
	}
	// prepare request message
	rqMsg := &rpc.Message{
		Type:             rawMsg.Type,
		RequestId:        rawMsg.RequestId,
		Key:              rawMsg.Key,
		ResponseRequired: rawMsg.ResponseRequired,
	}
	// check for registered message type
	// if provider registered, try to convert raw body to the provided type
	// otherwise response with the raw body
	if msgType.typeProvider != nil {
		bodyTyped := msgType.typeProvider()
		d, _ := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
			TagName:    "json",
			Result:     bodyTyped,
			DecodeHook: mapstructure.StringToTimeHookFunc(time.RFC3339),
		})
		err = d.Decode(rawMsg.Body)
		if err != nil {
			return rpc.ErrRpcRespInvalidBody(err, ctx, rawMsg.RequestId, rawMsg.Key)
		}
		rqMsg.Body = bodyTyped
	} else {
		rqMsg.Body = rawMsg.Body
	}
	// put request to the pool
	if rqMsg.ResponseRequired {
		r.rqPool.Queue(ctx, &rpc.Request{
			Msg: rqMsg,
			Ctx: ctx,
		})
	}
	// run callback
	if msgType.callback != nil {
		return msgType.callback(ctx, rqMsg)
	}
	l.Dbg("ok")
	return nil
}

func (r *rpcServer) RegisterType(messageType rpc.MessageType, callback rpc.Callback, provider rpc.MessageBodyTypeProvider) {
	r.msgTypes[messageType] = &msgType{
		callback:     callback,
		typeProvider: provider,
	}
}

func (r *rpcServer) Response(ctx context.Context, msg *rpc.Message) error {
	l := r.l().C(ctx).Mth("response").F(jet.KV{"type": msg.Type, "key": msg.Key, "rqId": msg.RequestId}).Dbg()
	// validate request
	if msg.RequestId == "" {
		return rpc.ErrRpcMsgNoRequestId(ctx)
	}
	if msg.Key == "" {
		return rpc.ErrRpcMsgNoKey(ctx)
	}
	// check for request in pool
	rq := r.rqPool.TryDequeue(msg.RequestId)
	if rq == nil {
		return rpc.ErrRpcRespNoRequestInPool(ctx, msg.RequestId, msg.Key)
	}
	// send to kafka
	err := r.callProducer.Send(ctx, msg.Key, msg)
	if err != nil {
		return err
	}
	l.Dbg("ok")
	return nil
}
