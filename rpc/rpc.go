package rpc

import (
	"context"
	"time"
)

type MessageType uint

type Message struct {
	Type             MessageType `json:"type"`
	RequestId        string      `json:"rqId"`
	Key              string      `json:"key"`
	ResponseRequired bool        `json:"respReq"`
	Body             interface{} `json:"body"`
}

type RawMessage struct {
	Type             MessageType            `json:"type"`
	RequestId        string                 `json:"rqId"`
	Key              string                 `json:"key"`
	ResponseRequired bool                   `json:"respReq"`
	Body             map[string]interface{} `json:"body"`
}

// MessageBodyTypeProvider should return an instance of the type to which reply body is cast
type MessageBodyTypeProvider func() interface{}

type Callback func(ctx context.Context, msg *Message) error
type ResponseCallback func(ctx context.Context, rqMsg, rsMsg *Message) error

type Client interface {
	// Call makes a rpcClient call
	Call(ctx context.Context, msg *Message, callback ResponseCallback) error
	// ResponseHandler must be setup as a subscriber of a response topic
	ResponseHandler(msg []byte) error
	// RegisterBodyTypeProvider allows providing a type to a response body to convert to
	// otherwise raw body is returned (map[string]interface{})
	RegisterBodyTypeProvider(MessageType, MessageBodyTypeProvider)
	// SetExpirationCallback sets up a callback which is called each time when a Request timeout expires
	SetExpirationCallback(callback Callback)
	// Start starts internal background processes
	Start(ctx context.Context)
	// Close closes all internal routines
	Close(ctx context.Context)
}

type Config struct {
	// CallTimeOut call timeout
	// after timeout expires TimeoutCallback is called
	CallTimeOut time.Duration
	// ClusterSupport additionally supports internal connection list with tracing messages keys
	// Thus, handler skips messages with keys which aren't presented in connection list
	// Note, it's client's responsibility to support connection list in sync
	ClusterSupport bool
}

type Server interface {
	// RequestHandler must be setup as a subscriber of a Request topic
	RequestHandler(msg []byte) error
	// RegisterType registers message type and allows setting up a callback method for the message type
	// if a received message doesn't have a callback specified, message is ignored
	// MessageBodyTypeProvider allows providing a type to a Request body to convert to
	// otherwise raw body is returned (map[string]interface{})
	RegisterType(MessageType, Callback, MessageBodyTypeProvider)
	// Response sends a response to a client topic
	// if no Request found, error is returned
	Response(ctx context.Context, msg *Message) error
	// SetExpirationCallback sets up a callback which is called each time when a Request timeout expires
	SetExpirationCallback(callback Callback)
	// Start starts internal background processes
	Start(ctx context.Context)
	// Close closes all internal routines
	Close(ctx context.Context)
}
